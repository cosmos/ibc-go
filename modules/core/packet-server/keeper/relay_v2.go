package keeper

import (
	"bytes"
	"context"
	"slices"
	"strconv"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

func (k Keeper) SendPacketV2(
	ctx context.Context,
	sourceID string,
	timeoutTimestamp uint64,
	data []channeltypes.PacketData,
) (uint64, error) {
	// Lookup counterparty associated with our source channel to retrieve the destination channel
	counterparty, ok := k.GetCounterparty(ctx, sourceID)
	if !ok {
		return 0, errorsmod.Wrap(types.ErrCounterpartyNotFound, sourceID)
	}
	// retrieve the sequence send for this channel
	// if no packets have been sent yet, initialize the sequence to 1.
	sequence, found := k.ChannelKeeper.GetNextSequenceSend(ctx, host.SentinelV2PortID, counterparty.ClientId)
	if !found {
		sequence = 1
	}

	// construct packet from given fields and channel state
	packet := channeltypesv2.NewPacketV2(sequence, sourceID, counterparty.ClientId, timeoutTimestamp, data...)

	if err := packet.ValidateBasic(); err != nil {
		return 0, errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "constructed packet failed basic validation: %v", err)
	}

	// check that the client of counterparty chain is still active
	if status := k.ClientKeeper.GetClientStatus(ctx, counterparty.ClientId); status != exported.Active {
		return 0, errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", counterparty.ClientId, status)
	}

	// retrieve latest height and timestamp of the client of counterparty chain
	latestHeight := k.ClientKeeper.GetClientLatestHeight(ctx, counterparty.ClientId)
	if latestHeight.IsZero() {
		return 0, errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "cannot send packet using client (%s) with zero height", counterparty.ClientId)
	}
	latestTimestamp, err := k.ClientKeeper.GetClientTimestampAtHeight(ctx, counterparty.ClientId, latestHeight)
	if err != nil {
		return 0, err
	}
	// check if packet is timed out on the receiving chain
	timeout := channeltypes.NewTimeoutWithTimestamp(timeoutTimestamp)
	if timeout.Elapsed(clienttypes.ZeroHeight(), latestTimestamp) {
		return 0, errorsmod.Wrap(timeout.ErrTimeoutElapsed(latestHeight, latestTimestamp), "invalid packet timeout")
	}
	commitment := channeltypes.CommitPacketV2(packet)

	// bump the sequence and set the packet commitment so it is provable by the counterparty
	k.ChannelKeeper.SetNextSequenceSend(ctx, host.SentinelV2PortID, counterparty.ClientId, sequence+1)
	k.ChannelKeeper.SetPacketCommitment(ctx, host.SentinelV2PortID, counterparty.ClientId, packet.GetSequence(), commitment)
	//	k.Logger(ctx).Info("packet sent", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_port", packetV2SentinelPort, "src_channel", packet.SourceChannel, "dst_port", packet.DestinationPort, "dst_channel", packet.DestinationChannel)

	//	channelkeeper.EmitSendPacketEventV2(ctx, packet, sentinelChannel(sourceID), timeoutHeight)
	return sequence, nil
}

func (k Keeper) RecvPacketV2(
	ctx context.Context,
	packet channeltypes.PacketV2,
	proof []byte,
	proofHeight exported.Height,
) error {
	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	counterparty, ok := k.GetCounterparty(ctx, packet.DestinationId)
	if !ok {
		return errorsmod.Wrap(types.ErrCounterpartyNotFound, packet.DestinationId)
	}
	if counterparty.ClientId != packet.SourceId {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// check if packet timed out by comparing it with the latest height of the chain
	selfTimestamp := uint64(sdkCtx.BlockTime().UnixNano())
	timeout := channeltypes.NewTimeoutWithTimestamp(packet.GetTimeoutTimestamp())
	if timeout.Elapsed(clienttypes.ZeroHeight(), selfTimestamp) {
		return errorsmod.Wrap(timeout.ErrTimeoutElapsed(clienttypes.ZeroHeight(), selfTimestamp), "packet timeout elapsed")
	}

	// REPLAY PROTECTION: Packet receipts will indicate that a packet has already been received
	// on unordered channels. Packet receipts must not be pruned, unless it has been marked stale
	// by the increase of the recvStartSequence.
	_, found := k.ChannelKeeper.GetPacketReceipt(ctx, host.SentinelV2PortID, packet.DestinationId, packet.Sequence)
	if found {
		// TODO: figure out events
		// channelkeeper.EmitRecvPacketEventV2(ctx, packet, sentinelChannel(packet.DestinationChannel))
		// This error indicates that the packet has already been relayed. Core IBC will
		// treat this error as a no-op in order to prevent an entire relay transaction
		// from failing and consuming unnecessary fees.
		return channeltypes.ErrNoOpMsg
	}

	path := host.PacketCommitmentKey(host.SentinelV2PortID, packet.SourceId, packet.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	commitment := channeltypes.CommitPacketV2(packet)

	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		packet.DestinationId,
		proofHeight,
		0, 0,
		proof,
		merklePath,
		commitment,
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet commitment verification for client (%s)", packet.DestinationId)
	}

	// Set Packet Receipt to prevent timeout from occurring on counterparty
	k.ChannelKeeper.SetPacketReceipt(ctx, host.SentinelV2PortID, packet.DestinationId, packet.Sequence)

	k.Logger(ctx).Info("packet received", "sequence", strconv.FormatUint(packet.Sequence, 10), "source-id", packet.SourceId, "dst-id", packet.DestinationId)

	// TODO: figure out events
	// channelkeeper.EmitRecvPacketEvent(ctx, packet, sentinelChannel(packet.DestinationChannel))

	return nil
}

// WriteAcknowledgementV2 writes the multi acknowledgement to the store. In the synchronous case, this is done
// in the core IBC handler. Async applications should call WriteAcknowledgementAsyncV2 to update
// the RecvPacketResult of the relevant application's recvResult.
func (k Keeper) WriteAcknowledgementV2(
	ctx context.Context,
	packet channeltypes.PacketV2,
	multiAck channeltypes.MultiAcknowledgement,
) error {
	// TODO: this should probably error out if any of the acks are async.
	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	counterparty, ok := k.GetCounterparty(ctx, packet.DestinationId)
	if !ok {
		return errorsmod.Wrap(types.ErrCounterpartyNotFound, packet.DestinationId)
	}

	if counterparty.ClientId != packet.SourceId {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	// NOTE: IBC app modules might have written the acknowledgement synchronously on
	// the OnRecvPacket callback so we need to check if the acknowledgement is already
	// set on the store and return an error if so.
	if k.ChannelKeeper.HasPacketAcknowledgement(ctx, host.SentinelV2PortID, packet.DestinationId, packet.Sequence) {
		return channeltypes.ErrAcknowledgementExists
	}

	if _, found := k.ChannelKeeper.GetPacketReceipt(ctx, host.SentinelV2PortID, packet.DestinationId, packet.Sequence); !found {
		return errorsmod.Wrap(channeltypes.ErrInvalidPacket, "receipt not found for packet")
	}

	multiAckBz := k.cdc.MustMarshal(&multiAck)
	// set the acknowledgement so that it can be verified on the other side
	k.ChannelKeeper.SetPacketAcknowledgement(
		ctx, host.SentinelV2PortID, packet.DestinationId, packet.GetSequence(),
		channeltypes.CommitAcknowledgement(multiAckBz),
	)

	k.Logger(ctx).Info("acknowledgement written", "sequence", strconv.FormatUint(packet.Sequence, 10), "dst_id", packet.DestinationId)

	// TODO: figure out events, we MUST emit the MultiAck structure here
	// channelkeeper.EmitWriteAcknowledgementEventV2(ctx, packet, sentinelChannel(packet.DestinationChannel), multiAck)

	return nil
}

func (k Keeper) AcknowledgePacketV2(
	ctx context.Context,
	packet channeltypes.PacketV2,
	multiAck channeltypes.MultiAcknowledgement,
	proofAcked []byte,
	proofHeight exported.Height,
) error {
	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	counterparty, ok := k.GetCounterparty(ctx, packet.SourceId)
	if !ok {
		return errorsmod.Wrap(types.ErrCounterpartyNotFound, packet.SourceId)
	}

	if counterparty.ClientId != packet.DestinationId {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	commitment := k.ChannelKeeper.GetPacketCommitment(ctx, host.SentinelV2PortID, packet.SourceId, packet.Sequence)
	if len(commitment) == 0 {
		// TODO: figure out events
		// channelkeeper.EmitAcknowledgePacketEventV2(ctx, packet, sentinelChannel(packet.SourceChannel))

		// This error indicates that the acknowledgement has already been relayed
		// or there is a misconfigured relayer attempting to prove an acknowledgement
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypes.CommitPacketV2(packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "commitment bytes are not equal: got (%v), expected (%v)", packetCommitment, commitment)
	}

	path := host.PacketAcknowledgementKey(host.SentinelV2PortID, packet.DestinationId, packet.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	bz := k.cdc.MustMarshal(&multiAck)
	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		packet.SourceId,
		proofHeight,
		0, 0,
		proofAcked,
		merklePath,
		channeltypes.CommitAcknowledgement(bz),
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet acknowledgement verification for client (%s)", packet.SourceId)
	}

	k.ChannelKeeper.DeletePacketCommitment(ctx, host.SentinelV2PortID, packet.SourceId, packet.Sequence)

	k.Logger(ctx).Info("packet acknowledged", "sequence", strconv.FormatUint(packet.GetSequence(), 10), "src_id", packet.SourceId, "dst_id", packet.DestinationId)

	// TODO: figure out events
	// channelkeeper.EmitAcknowledgePacketEventV2(ctx, packet, sentinelChannel(packet.SourceChannel))

	return nil
}

// WriteAcknowledgementAsyncV2 updates the recv packet result for the given app name in the multi acknowledgement.
// If all acknowledgements are now either success or failed acks, it writes the final multi ack.
func (k *Keeper) WriteAcknowledgementAsyncV2(
	ctx context.Context,
	packet channeltypes.PacketV2,
	appName string,
	recvResult channeltypes.RecvPacketResult,
) error {
	// we should have stored the multi ack structure in OnRecvPacket
	ackResults, found := k.ChannelKeeper.GetMultiAcknowledgement(ctx, host.SentinelV2PortID, packet.DestinationId, packet.GetSequence())
	if !found {
		return errorsmod.Wrapf(channeltypes.ErrInvalidAcknowledgement, "multi-acknowledgement not found for %s", appName)
	}

	// find the index that corresponds to the app.
	index := slices.IndexFunc(ackResults.AcknowledgementResults, func(result channeltypes.AcknowledgementResult) bool {
		return result.AppName == appName
	})

	if index == -1 {
		return errorsmod.Wrapf(channeltypes.ErrInvalidAcknowledgement, "acknowledgement not found for %s", appName)
	}

	existingResult := ackResults.AcknowledgementResults[index]

	// ensure that the existing status is async.
	if existingResult.RecvPacketResult.Status != channeltypes.PacketStatus_Async {
		return errorsmod.Wrapf(channeltypes.ErrInvalidAcknowledgement, "acknowledgement for %s is not async", appName)
	}

	// modify the result and set it back.
	ackResults.AcknowledgementResults[index].RecvPacketResult = recvResult
	k.ChannelKeeper.SetMultiAcknowledgement(ctx, host.SentinelV2PortID, packet.DestinationId, packet.GetSequence(), ackResults)

	// check if all acknowledgements are now sync.
	isAsync := slices.ContainsFunc(ackResults.AcknowledgementResults, func(ackResult channeltypes.AcknowledgementResult) bool {
		return ackResult.RecvPacketResult.Status == channeltypes.PacketStatus_Async
	})

	if !isAsync {
		// if there are no more async acks, we can write the final multi ack.
		return k.WriteAcknowledgementV2(ctx, packet, ackResults)
	}

	// we have updated one app's result, but there are still async results pending acknowledgement.
	return nil
}

// TimeoutPacketV2 implements the timeout logic required by a packet handler.
// The packet is checked for correctness including asserting that the packet was
// sent and received on clients which are counterparties for one another.
// If no packet commitment exists, a no-op error is returned, otherwise
// an absence proof of the packet receipt is performed to ensure that the packet
// was never delivered to the counterparty. If successful, the packet commitment
// is deleted and the packet has completed its lifecycle.
func (k Keeper) TimeoutPacketV2(
	ctx context.Context,
	packet channeltypes.PacketV2,
	proof []byte,
	proofHeight exported.Height,
) error {
	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	counterparty, ok := k.GetCounterparty(ctx, packet.SourceId)
	if !ok {
		return errorsmod.Wrap(types.ErrCounterpartyNotFound, packet.SourceId)
	}

	if counterparty.ClientId != packet.DestinationId {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	// check that timeout height or timeout timestamp has passed on the other end
	proofTimestamp, err := k.ClientKeeper.GetClientTimestampAtHeight(ctx, packet.SourceId, proofHeight)
	if err != nil {
		return err
	}

	timeout := channeltypes.NewTimeoutWithTimestamp(packet.GetTimeoutTimestamp())
	if !timeout.Elapsed(clienttypes.ZeroHeight(), proofTimestamp) {
		return errorsmod.Wrap(timeout.ErrTimeoutNotReached(proofHeight.(clienttypes.Height), proofTimestamp), "packet timeout not reached")
	}

	// check that the commitment has not been cleared and that it matches the packet sent by relayer
	commitment := k.ChannelKeeper.GetPacketCommitment(ctx, host.SentinelV2PortID, packet.SourceId, packet.Sequence)

	if len(commitment) == 0 {
		// TODO: pending decision on event structure for V2.
		// channelkeeper.EmitTimeoutPacketEvent(ctx, packet, nil)
		// This error indicates that the timeout has already been relayed
		// or there is a misconfigured relayer attempting to prove a timeout
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypes.CommitPacketV2(packet)
	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
	}

	// verify packet receipt absence
	path := host.PacketReceiptKey(host.SentinelV2PortID, packet.DestinationId, packet.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	if err := k.ClientKeeper.VerifyNonMembership(
		ctx,
		packet.SourceId,
		proofHeight,
		0, 0,
		proof,
		merklePath,
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet receipt absence verification for client (%s)", packet.SourceId)
	}

	// delete packet commitment to prevent replay
	k.ChannelKeeper.DeletePacketCommitment(ctx, host.SentinelV2PortID, packet.SourceId, packet.Sequence)

	k.Logger(ctx).Info("packet timed out", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_port", packet.Data[0].SourcePort, "src_channel", packet.SourceId, "dst_port", packet.Data[0].DestinationPort, "dst_channel", packet.DestinationId)

	// TODO: pending decision on event structure for V2.
	// channelkeeper.EmitTimeoutPacketEvent(ctx, packet, nil)

	return nil
}
