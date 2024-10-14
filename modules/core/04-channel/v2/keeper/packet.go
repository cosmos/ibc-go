package keeper

import (
	"bytes"
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// sendPacket constructs a packet from the input arguments, writes a packet commitment to state
// in order for the packet to be sent to the counterparty.
func (k *Keeper) sendPacket(
	ctx context.Context,
	sourceChannel string,
	timeoutTimestamp uint64,
	data []channeltypesv2.PacketData,
) (uint64, string, error) {
	// Lookup channel associated with our source channel to retrieve the destination channel
	channel, ok := k.GetChannel(ctx, sourceChannel)
	if !ok {
		// TODO: figure out how aliasing will work when more than one packet data is sent.
		channel, ok = k.convertV1Channel(ctx, data[0].SourcePort, sourceChannel)
		if !ok {
			return 0, "", errorsmod.Wrap(channeltypesv2.ErrChannelNotFound, sourceChannel)
		}
	}

	destChannel := channel.CounterpartyChannelId
	clientID := channel.ClientId

	sequence, found := k.GetNextSequenceSend(ctx, sourceChannel)
	if !found {
		return 0, "", errorsmod.Wrapf(
			channeltypesv2.ErrSequenceSendNotFound,
			"source channel: %s", sourceChannel,
		)
	}

	// construct packet from given fields and channel state
	packet := channeltypesv2.NewPacket(sequence, sourceChannel, destChannel, timeoutTimestamp, data...)

	if err := packet.ValidateBasic(); err != nil {
		return 0, "", errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "constructed packet failed basic validation: %v", err)
	}

	// check that the client of counterparty chain is still active
	if status := k.ClientKeeper.GetClientStatus(ctx, clientID); status != exported.Active {
		return 0, "", errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", clientID, status)
	}

	// retrieve latest height and timestamp of the client of counterparty chain
	latestHeight := k.ClientKeeper.GetClientLatestHeight(ctx, clientID)
	if latestHeight.IsZero() {
		return 0, "", errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "cannot send packet using client (%s) with zero height", clientID)
	}

	latestTimestamp, err := k.ClientKeeper.GetClientTimestampAtHeight(ctx, clientID, latestHeight)
	if err != nil {
		return 0, "", err
	}
	// check if packet is timed out on the receiving chain
	timeout := channeltypes.NewTimeoutWithTimestamp(timeoutTimestamp)
	if timeout.TimestampElapsed(latestTimestamp) {
		return 0, "", errorsmod.Wrap(timeout.ErrTimeoutElapsed(latestHeight, latestTimestamp), "invalid packet timeout")
	}

	commitment := channeltypesv2.CommitPacket(packet)

	// bump the sequence and set the packet commitment, so it is provable by the counterparty
	k.SetNextSequenceSend(ctx, sourceChannel, sequence+1)
	k.SetPacketCommitment(ctx, sourceChannel, packet.GetSequence(), commitment)

	k.Logger(ctx).Info("packet sent", "sequence", strconv.FormatUint(packet.Sequence, 10), "dest_channel_id", packet.DestinationChannel, "src_channel_id", packet.SourceChannel)

	EmitSendPacketEvents(ctx, packet)

	return sequence, destChannel, nil
}

// recvPacket implements the packet receiving logic required by a packet handler.￼
// The packet is checked for correctness including asserting that the packet was
// sent and received on clients which are counterparties for one another.
// If the packet has already been received a no-op error is returned.
// The packet handler will verify that the packet has not timed out and that the
// counterparty stored a packet commitment. If successful, a packet receipt is stored
// to indicate to the counterparty successful delivery.
func (k *Keeper) recvPacket(
	ctx context.Context,
	packet channeltypesv2.Packet,
	proof []byte,
	proofHeight exported.Height,
) error {
	// Lookup channel associated with destination channel ID and ensure
	// that the packet was indeed sent by our counterparty by verifying
	// packet sender is our channel's counterparty channel id.
	channel, ok := k.GetChannel(ctx, packet.DestinationChannel)
	if !ok {
		// TODO: figure out how aliasing will work when more than one packet data is sent.
		channel, ok = k.convertV1Channel(ctx, packet.Data[0].DestinationPort, packet.DestinationChannel)
		if !ok {
			return errorsmod.Wrap(channeltypesv2.ErrChannelNotFound, packet.DestinationChannel)
		}
	}
	if channel.CounterpartyChannelId != packet.SourceChannel {
		return channeltypes.ErrInvalidChannelIdentifier
	}
	clientID := channel.ClientId

	// check if packet timed out by comparing it with the latest height of the chain
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	selfHeight, selfTimestamp := clienttypes.GetSelfHeight(ctx), uint64(sdkCtx.BlockTime().UnixNano())
	timeout := channeltypes.NewTimeoutWithTimestamp(packet.GetTimeoutTimestamp())
	if timeout.Elapsed(selfHeight, selfTimestamp) {
		return errorsmod.Wrap(timeout.ErrTimeoutElapsed(selfHeight, selfTimestamp), "packet timeout elapsed")
	}

	// REPLAY PROTECTION: Packet receipts will indicate that a packet has already been received
	// on unordered channels. Packet receipts must not be pruned, unless it has been marked stale
	// by the increase of the recvStartSequence.
	if k.HasPacketReceipt(ctx, packet.DestinationChannel, packet.Sequence) {
		EmitRecvPacketEvents(ctx, packet)
		// This error indicates that the packet has already been relayed. Core IBC will
		// treat this error as a no-op in order to prevent an entire relay transaction
		// from failing and consuming unnecessary fees.
		return channeltypes.ErrNoOpMsg
	}

	path := hostv2.PacketCommitmentKey(packet.SourceChannel, packet.Sequence)
	merklePath := channeltypesv2.BuildMerklePath(channel.MerklePathPrefix, path)

	commitment := channeltypesv2.CommitPacket(packet)

	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		clientID,
		proofHeight,
		0, 0,
		proof,
		merklePath,
		commitment,
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet commitment verification for client (%s)", packet.DestinationChannel)
	}

	// Set Packet Receipt to prevent timeout from occurring on counterparty
	k.SetPacketReceipt(ctx, packet.DestinationChannel, packet.Sequence)

	k.Logger(ctx).Info("packet received", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_id", packet.SourceChannel, "dst_id", packet.DestinationChannel)

	EmitRecvPacketEvents(ctx, packet)

	return nil
}

func (k *Keeper) acknowledgePacket(ctx context.Context, packet channeltypesv2.Packet, acknowledgement channeltypesv2.Acknowledgement, proof []byte, proofHeight exported.Height) error {
	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	channel, ok := k.GetChannel(ctx, packet.SourceChannel)
	if !ok {
		return errorsmod.Wrap(channeltypesv2.ErrChannelNotFound, packet.SourceChannel)
	}

	if channel.CounterpartyChannelId != packet.DestinationChannel {
		return channeltypes.ErrInvalidChannelIdentifier
	}
	clientID := channel.ClientId

	commitment := k.GetPacketCommitment(ctx, packet.SourceChannel, packet.Sequence)
	if len(commitment) == 0 {
		// TODO: signal noop in events?
		EmitAcknowledgePacketEvents(ctx, packet)

		// This error indicates that the acknowledgement has already been relayed
		// or there is a misconfigured relayer attempting to prove an acknowledgement
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypesv2.CommitPacket(packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "commitment bytes are not equal: got (%v), expected (%v)", packetCommitment, commitment)
	}

	path := hostv2.PacketAcknowledgementKey(packet.DestinationChannel, packet.Sequence)
	merklePath := channeltypesv2.BuildMerklePath(channel.MerklePathPrefix, path)

	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		clientID,
		proofHeight,
		0, 0,
		proof,
		merklePath,
		channeltypesv2.CommitAcknowledgement(acknowledgement),
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet acknowledgement verification for client (%s)", clientID)
	}

	k.DeletePacketCommitment(ctx, packet.SourceChannel, packet.Sequence)

	k.Logger(ctx).Info("packet acknowledged", "sequence", strconv.FormatUint(packet.GetSequence(), 10), "source_channel_id", packet.GetSourceChannel(), "destination_channel_id", packet.GetDestinationChannel())

	EmitAcknowledgePacketEvents(ctx, packet)

	return nil
}

// timeoutPacket implements the timeout logic required by a packet handler.
// The packet is checked for correctness including asserting that the packet was
// sent and received on clients which are counterparties for one another.
// If no packet commitment exists, a no-op error is returned, otherwise
// an absence proof of the packet receipt is performed to ensure that the packet
// was never delivered to the counterparty. If successful, the packet commitment
// is deleted and the packet has completed its lifecycle.
func (k *Keeper) timeoutPacket(
	ctx context.Context,
	packet channeltypesv2.Packet,
	proof []byte,
	proofHeight exported.Height,
) error {
	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	channel, ok := k.GetChannel(ctx, packet.SourceChannel)
	if !ok {
		// TODO: figure out how aliasing will work when more than one packet data is sent.
		channel, ok = k.convertV1Channel(ctx, packet.Data[0].SourcePort, packet.SourceChannel)
		if !ok {
			return errorsmod.Wrap(channeltypesv2.ErrChannelNotFound, packet.DestinationChannel)
		}
	}
	if channel.CounterpartyChannelId != packet.DestinationChannel {
		return channeltypes.ErrInvalidChannelIdentifier
	}
	clientID := channel.ClientId

	// check that timeout height or timeout timestamp has passed on the other end
	proofTimestamp, err := k.ClientKeeper.GetClientTimestampAtHeight(ctx, clientID, proofHeight)
	if err != nil {
		return err
	}

	timeout := channeltypes.NewTimeoutWithTimestamp(packet.GetTimeoutTimestamp())
	if !timeout.Elapsed(clienttypes.ZeroHeight(), proofTimestamp) {
		return errorsmod.Wrap(timeout.ErrTimeoutNotReached(proofHeight.(clienttypes.Height), proofTimestamp), "packet timeout not reached")
	}

	// check that the commitment has not been cleared and that it matches the packet sent by relayer
	commitment := k.GetPacketCommitment(ctx, packet.SourceChannel, packet.Sequence)
	if len(commitment) == 0 {
		EmitTimeoutPacketEvents(ctx, packet)
		// This error indicates that the timeout has already been relayed
		// or there is a misconfigured relayer attempting to prove a timeout
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypesv2.CommitPacket(packet)
	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
	}

	// verify packet receipt absence
	path := hostv2.PacketReceiptKey(packet.SourceChannel, packet.Sequence)
	merklePath := channeltypesv2.BuildMerklePath(channel.MerklePathPrefix, path)

	if err := k.ClientKeeper.VerifyNonMembership(
		ctx,
		clientID,
		proofHeight,
		0, 0,
		proof,
		merklePath,
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet receipt absence verification for client (%s)", packet.SourceChannel)
	}

	// delete packet commitment to prevent replay
	k.DeletePacketCommitment(ctx, packet.SourceChannel, packet.Sequence)

	k.Logger(ctx).Info("packet timed out", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_channel_id", packet.SourceChannel, "dst_channel_id", packet.DestinationChannel)

	EmitTimeoutPacketEvents(ctx, packet)

	return nil
}
