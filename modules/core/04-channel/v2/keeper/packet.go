package keeper

import (
	"bytes"
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// sendPacket constructs a packet from the input arguments, writes a packet commitment to state
// in order for the packet to be sent to the counterparty.
func (k *Keeper) sendPacket(
	ctx context.Context,
	sourceClient string,
	timeoutTimestamp uint64,
	payloads []types.Payload,
) (uint64, string, error) {
	// lookup counterparty and clientid from packet identifiers
	clientID, counterparty, err := k.resolveV2Identifiers(ctx, payloads[0].SourcePort, sourceClient)
	if err != nil {
		return 0, "", err
	}

	sequence, found := k.GetNextSequenceSend(ctx, sourceClient)
	if !found {
		// initialize sequnce to 1 if it does not exist
		sequence = 1
	}

	// construct packet from given fields and channel state
	packet := types.NewPacket(sequence, sourceClient, counterparty.ClientId, timeoutTimestamp, payloads...)

	if err := packet.ValidateBasic(); err != nil {
		return 0, "", errorsmod.Wrapf(types.ErrInvalidPacket, "constructed packet failed basic validation: %v", err)
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

	// client timestamps are in nanoseconds while packet timeouts are in seconds
	// thus to compare them, we convert the packet timeout to nanoseconds
	timeoutTimestamp = types.TimeoutTimestampToNanos(packet.TimeoutTimestamp)
	if latestTimestamp >= timeoutTimestamp {
		return 0, "", errorsmod.Wrapf(types.ErrTimeoutElapsed, "latest timestamp: %d, timeout timestamp: %d", latestTimestamp, timeoutTimestamp)
	}

	commitment := types.CommitPacket(packet)

	// bump the sequence and set the packet commitment, so it is provable by the counterparty
	k.SetNextSequenceSend(ctx, sourceClient, sequence+1)
	k.SetPacketCommitment(ctx, sourceClient, packet.GetSequence(), commitment)

	k.Logger(ctx).Info("packet sent", "sequence", strconv.FormatUint(packet.Sequence, 10), "dst_client_id", packet.DestinationClient, "src_client_id", packet.SourceClient)

	emitSendPacketEvents(ctx, packet)

	return sequence, counterparty.ClientId, nil
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
	packet types.Packet,
	proof []byte,
	proofHeight exported.Height,
) error {
	// lookup counterparty and clientid from packet identifiers
	clientID, counterparty, err := k.resolveV2Identifiers(ctx, packet.Payloads[0].DestinationPort, packet.DestinationClient)
	if err != nil {
		return err
	}

	if counterparty.ClientId != packet.SourceClient {
		return errorsmod.Wrapf(types.ErrInvalidChannelIdentifier, "counterparty id (%s) does not match packet source id (%s)", counterparty.ClientId, packet.SourceClient)
	}

	// check if packet timed out by comparing it with the latest height of the chain
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentTimestamp := uint64(sdkCtx.BlockTime().Unix())
	if currentTimestamp >= packet.TimeoutTimestamp {
		return errorsmod.Wrapf(types.ErrTimeoutElapsed, "current timestamp: %d, timeout timestamp: %d", currentTimestamp, packet.TimeoutTimestamp)
	}

	// REPLAY PROTECTION: Packet receipts will indicate that a packet has already been received
	// on unordered channels. Packet receipts must not be pruned, unless it has been marked stale
	// by the increase of the recvStartSequence.
	if k.HasPacketReceipt(ctx, packet.DestinationClient, packet.Sequence) {
		// This error indicates that the packet has already been relayed. Core IBC will
		// treat this error as a no-op in order to prevent an entire relay transaction
		// from failing and consuming unnecessary fees.
		return types.ErrNoOpMsg
	}

	path := hostv2.PacketCommitmentKey(packet.SourceClient, packet.Sequence)
	merklePrefix := commitmenttypes.NewMerklePath(counterparty.MerklePrefix...)
	merklePath := types.BuildMerklePath(merklePrefix, path)

	commitment := types.CommitPacket(packet)

	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		clientID,
		proofHeight,
		0, 0,
		proof,
		merklePath,
		commitment,
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet commitment verification for client (%s)", clientID)
	}

	// Set Packet Receipt to prevent timeout from occurring on counterparty
	k.SetPacketReceipt(ctx, packet.DestinationClient, packet.Sequence)

	k.Logger(ctx).Info("packet received", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_client_id", packet.SourceClient, "dst_client_id", packet.DestinationClient)

	emitRecvPacketEvents(ctx, packet)

	return nil
}

// WriteAcknowledgement writes the acknowledgement to the store.
// TODO: change this function to accept destPort, destChannel, sequence, ack
func (k Keeper) WriteAcknowledgement(
	ctx context.Context,
	packet types.Packet,
	ack types.Acknowledgement,
) error {
	// lookup counterparty and clientid from packet identifiers
	_, counterparty, err := k.resolveV2Identifiers(ctx, packet.Payloads[0].DestinationPort, packet.DestinationClient)
	if err != nil {
		return err
	}

	if counterparty.ClientId != packet.SourceClient {
		return errorsmod.Wrapf(types.ErrInvalidChannelIdentifier, "counterparty id (%s) does not match packet source id (%s)", counterparty.ClientId, packet.SourceClient)
	}

	// NOTE: IBC app modules might have written the acknowledgement synchronously on
	// the OnRecvPacket callback so we need to check if the acknowledgement is already
	// set on the store and return an error if so.
	if k.HasPacketAcknowledgement(ctx, packet.DestinationClient, packet.Sequence) {
		return errorsmod.Wrapf(types.ErrAcknowledgementExists, "acknowledgement for id %s, sequence %d already exists", packet.DestinationClient, packet.Sequence)
	}

	if _, found := k.GetPacketReceipt(ctx, packet.DestinationClient, packet.Sequence); !found {
		return errorsmod.Wrap(types.ErrInvalidPacket, "receipt not found for packet")
	}

	// set the acknowledgement so that it can be verified on the other side
	k.SetPacketAcknowledgement(
		ctx, packet.DestinationClient, packet.Sequence,
		types.CommitAcknowledgement(ack),
	)

	k.Logger(ctx).Info("acknowledgement written", "sequence", strconv.FormatUint(packet.Sequence, 10), "dst_client_id", packet.DestinationClient)

	emitWriteAcknowledgementEvents(ctx, packet, ack)

	// TODO: delete the packet that has been stored in ibc-core.

	return nil
}

func (k *Keeper) acknowledgePacket(ctx context.Context, packet types.Packet, acknowledgement types.Acknowledgement, proof []byte, proofHeight exported.Height) error {
	// lookup counterparty and clientid from packet identifiers
	clientID, counterparty, err := k.resolveV2Identifiers(ctx, packet.Payloads[0].SourcePort, packet.SourceClient)
	if err != nil {
		return err
	}

	if counterparty.ClientId != packet.DestinationClient {
		return errorsmod.Wrapf(types.ErrInvalidChannelIdentifier, "counterparty id (%s) does not match packet destination id (%s)", counterparty.ClientId, packet.DestinationClient)
	}

	commitment := k.GetPacketCommitment(ctx, packet.SourceClient, packet.Sequence)
	if len(commitment) == 0 {
		// This error indicates that the acknowledgement has already been relayed
		// or there is a misconfigured relayer attempting to prove an acknowledgement
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return types.ErrNoOpMsg
	}

	packetCommitment := types.CommitPacket(packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return errorsmod.Wrapf(types.ErrInvalidPacket, "commitment bytes are not equal: got (%v), expected (%v)", packetCommitment, commitment)
	}

	path := hostv2.PacketAcknowledgementKey(packet.DestinationClient, packet.Sequence)
	merklePrefix := commitmenttypes.NewMerklePath(counterparty.MerklePrefix...)
	merklePath := types.BuildMerklePath(merklePrefix, path)

	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		clientID,
		proofHeight,
		0, 0,
		proof,
		merklePath,
		types.CommitAcknowledgement(acknowledgement),
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet acknowledgement verification for client (%s)", clientID)
	}

	k.DeletePacketCommitment(ctx, packet.SourceClient, packet.Sequence)

	k.Logger(ctx).Info("packet acknowledged", "sequence", strconv.FormatUint(packet.GetSequence(), 10), "src_client_id", packet.GetSourceClient(), "dst_client_id", packet.GetDestinationClient())

	emitAcknowledgePacketEvents(ctx, packet)

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
	packet types.Packet,
	proof []byte,
	proofHeight exported.Height,
) error {
	// lookup counterparty and clientid from packet identifiers
	clientID, counterparty, err := k.resolveV2Identifiers(ctx, packet.Payloads[0].SourcePort, packet.SourceClient)
	if err != nil {
		return err
	}

	if counterparty.ClientId != packet.DestinationClient {
		return errorsmod.Wrapf(types.ErrInvalidChannelIdentifier, "counterparty id (%s) does not match packet destination id (%s)", counterparty.ClientId, packet.DestinationClient)
	}

	// check that timeout height or timeout timestamp has passed on the other end
	proofTimestamp, err := k.ClientKeeper.GetClientTimestampAtHeight(ctx, clientID, proofHeight)
	if err != nil {
		return err
	}

	timeoutTimestamp := types.TimeoutTimestampToNanos(packet.TimeoutTimestamp)
	if proofTimestamp < timeoutTimestamp {
		return errorsmod.Wrapf(types.ErrTimeoutNotReached, "proof timestamp: %d, timeout timestamp: %d", proofTimestamp, timeoutTimestamp)
	}

	// check that the commitment has not been cleared and that it matches the packet sent by relayer
	commitment := k.GetPacketCommitment(ctx, packet.SourceClient, packet.Sequence)
	if len(commitment) == 0 {
		// This error indicates that the timeout has already been relayed
		// or there is a misconfigured relayer attempting to prove a timeout
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return types.ErrNoOpMsg
	}

	packetCommitment := types.CommitPacket(packet)
	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return errorsmod.Wrapf(types.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
	}

	// verify packet receipt absence
	path := hostv2.PacketReceiptKey(packet.DestinationClient, packet.Sequence)
	merklePrefix := commitmenttypes.NewMerklePath(counterparty.MerklePrefix...)
	merklePath := types.BuildMerklePath(merklePrefix, path)

	if err := k.ClientKeeper.VerifyNonMembership(
		ctx,
		clientID,
		proofHeight,
		0, 0,
		proof,
		merklePath,
	); err != nil {
		return errorsmod.Wrapf(err, "failed packet receipt absence verification for client (%s)", clientID)
	}

	// delete packet commitment to prevent replay
	k.DeletePacketCommitment(ctx, packet.SourceClient, packet.Sequence)

	k.Logger(ctx).Info("packet timed out", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_client_id", packet.SourceClient, "dst_client_id", packet.DestinationClient)

	emitTimeoutPacketEvents(ctx, packet)

	return nil
}
