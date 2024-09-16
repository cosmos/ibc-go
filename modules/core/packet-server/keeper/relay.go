package keeper

import (
	"bytes"
	"strconv"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channelkeeper "github.com/cosmos/ibc-go/v9/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

// SendPacket implements the packet sending logic required by a packet handler.
// It will generate a packet and store the commitment hash if all arguments provided are valid.
// The destination channel will be filled in using the counterparty information.
// The next sequence send will be initialized if this is the first packet sent for the given client.
func (k Keeper) SendPacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	sourceChannel string,
	sourcePort string,
	destPort string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	version string,
	data []byte,
) (uint64, error) {
	// Lookup counterparty associated with our source channel to retrieve the destination channel
	counterparty, ok := k.GetCounterparty(ctx, sourceChannel)
	if !ok {
		return 0, errorsmod.Wrap(types.ErrCounterpartyNotFound, sourceChannel)
	}
	destChannel := counterparty.ClientId

	// retrieve the sequence send for this channel
	// if no packets have been sent yet, initialize the sequence to 1.
	sequence, found := k.ChannelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		sequence = 1
	}

	// construct packet from given fields and channel state
	// TODO: packet only being used in event emission.
	packet := channeltypes.NewPacketWithVersion(data, sequence, sourcePort, sourceChannel,
		destPort, destChannel, timeoutHeight, timeoutTimestamp, version)

	// TODO: replace with a direct creation of a PacketV2
	packetV2, err := channeltypes.ConvertPacketV1toV2(packet)
	if err != nil {
		return 0, err
	}

	if err := packetV2.ValidateBasic(); err != nil {
		return 0, errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "constructed packet failed basic validation: %v", err)
	}

	// check that the client of counterparty chain is still active
	if status := k.ClientKeeper.GetClientStatus(ctx, sourceChannel); status != exported.Active {
		return 0, errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client (%s) status is %s", sourceChannel, status)
	}

	// retrieve latest height and timestamp of the client of counterparty chain
	latestHeight := k.ClientKeeper.GetClientLatestHeight(ctx, sourceChannel)
	if latestHeight.IsZero() {
		return 0, errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "cannot send packet using client (%s) with zero height", sourceChannel)
	}

	latestTimestamp, err := k.ClientKeeper.GetClientTimestampAtHeight(ctx, sourceChannel, latestHeight)
	if err != nil {
		return 0, err
	}

	// check if packet is timed out on the receiving chain
	timeout := channeltypes.NewTimeoutWithTimestamp(packetV2.GetTimeoutTimestamp())
	if timeout.Elapsed(latestHeight, latestTimestamp) {
		return 0, errorsmod.Wrap(timeout.ErrTimeoutElapsed(latestHeight, latestTimestamp), "invalid packet timeout")
	}

	commitment := channeltypes.CommitPacketV2(packetV2)

	// bump the sequence and set the packet commitment so it is provable by the counterparty
	k.ChannelKeeper.SetNextSequenceSend(ctx, sourcePort, sourceChannel, sequence+1)
	k.ChannelKeeper.SetPacketCommitment(ctx, sourcePort, sourceChannel, packetV2.GetSequence(), commitment)

	k.Logger(ctx).Info("packet sent", "sequence", strconv.FormatUint(packetV2.Sequence, 10), "src_port", packetV2.Data[0].SourcePort, "src_channel", packetV2.SourceId, "dst_port", packetV2.Data[0].DestinationPort, "dst_channel", packetV2.DestinationId)

	channelkeeper.EmitSendPacketEvent(ctx, packet, nil, timeoutHeight)

	return sequence, nil
}

// RecvPacket implements the packet receiving logic required by a packet handler.
// The packet is checked for correctness including asserting that the packet was
// sent and received on clients which are counterparties for one another.
// If the packet has already been received a no-op error is returned.
// The packet handler will verify that the packet has not timed out and that the
// counterparty stored a packet commitment. If successful, a packet receipt is stored
// to indicate to the counterparty successful delivery.
func (k Keeper) RecvPacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packet channeltypes.Packet,
	proof []byte,
	proofHeight exported.Height,
) (string, error) {
	packetV2, err := channeltypes.ConvertPacketV1toV2(packet)
	if err != nil {
		return "", err
	}

	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	counterparty, ok := k.GetCounterparty(ctx, packetV2.DestinationId)
	if !ok {
		return "", errorsmod.Wrap(types.ErrCounterpartyNotFound, packetV2.DestinationId)
	}

	if counterparty.ClientId != packetV2.SourceId {
		return "", channeltypes.ErrInvalidChannelIdentifier
	}

	// check if packet timed out by comparing it with the latest height of the chain
	selfHeight, selfTimestamp := clienttypes.GetSelfHeight(ctx), uint64(ctx.BlockTime().UnixNano())
	timeout := channeltypes.NewTimeoutWithTimestamp(packetV2.GetTimeoutTimestamp())
	if timeout.Elapsed(selfHeight, selfTimestamp) {
		return "", errorsmod.Wrap(timeout.ErrTimeoutElapsed(selfHeight, selfTimestamp), "packet timeout elapsed")
	}

	// REPLAY PROTECTION: Packet receipts will indicate that a packet has already been received
	// on unordered channels. Packet receipts must not be pruned, unless it has been marked stale
	// by the increase of the recvStartSequence.
	_, found := k.ChannelKeeper.GetPacketReceipt(ctx, packetV2.Data[0].DestinationPort, packetV2.DestinationId, packetV2.Sequence)
	if found {
		// TODO: explicitly using packet(V1) here, as the event structure will remain the same until PacketV2 API is being used.
		channelkeeper.EmitRecvPacketEvent(ctx, packet, nil)
		// This error indicates that the packet has already been relayed. Core IBC will
		// treat this error as a no-op in order to prevent an entire relay transaction
		// from failing and consuming unnecessary fees.
		return "", channeltypes.ErrNoOpMsg
	}

	path := host.PacketCommitmentKey(packetV2.Data[0].SourcePort, packetV2.SourceId, packetV2.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	commitment := channeltypes.CommitPacketV2(packetV2)

	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		packetV2.DestinationId,
		proofHeight,
		0, 0,
		proof,
		merklePath,
		commitment,
	); err != nil {
		return "", errorsmod.Wrapf(err, "failed packet commitment verification for client (%s)", packet.DestinationChannel)
	}

	// Set Packet Receipt to prevent timeout from occurring on counterparty
	k.ChannelKeeper.SetPacketReceipt(ctx, packetV2.Data[0].DestinationPort, packetV2.DestinationId, packetV2.Sequence)

	k.Logger(ctx).Info("packet received", "sequence", strconv.FormatUint(packetV2.Sequence, 10), "src_port", packetV2.Data[0].SourcePort, "src_channel", packetV2.SourceId, "dst_port", packetV2.Data[0].DestinationPort, "dst_channel", packetV2.DestinationId)

	// TODO: explicitly using packet(V1) here, as the event structure will remain the same until PacketV2 API is being used.
	channelkeeper.EmitRecvPacketEvent(ctx, packet, nil)

	return packetV2.Data[0].Payload.Version, nil
}

// WriteAcknowledgement implements the async acknowledgement writing logic required by a packet handler.
// The packet is checked for correctness including asserting that the packet was
// sent and received on clients which are counterparties for one another.
// If no acknowledgement exists for the given packet, then a commitment of the acknowledgement
// is written into state.
func (k Keeper) WriteAcknowledgement(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packetI exported.PacketI,
	ack exported.Acknowledgement,
) error {
	packet, ok := packetI.(channeltypes.Packet)
	if !ok {
		return errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "expected type %T, got %T", &channeltypes.Packet{}, packetI)
	}
	if packet.ProtocolVersion != channeltypes.IBC_VERSION_2 {
		return channeltypes.ErrInvalidPacket
	}

	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	counterparty, ok := k.GetCounterparty(ctx, packet.DestinationChannel)
	if !ok {
		return errorsmod.Wrap(types.ErrCounterpartyNotFound, packet.DestinationChannel)
	}
	if counterparty.ClientId != packet.SourceChannel {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	// NOTE: IBC app modules might have written the acknowledgement synchronously on
	// the OnRecvPacket callback so we need to check if the acknowledgement is already
	// set on the store and return an error if so.
	if k.ChannelKeeper.HasPacketAcknowledgement(ctx, packet.DestinationPort, packet.DestinationChannel, packet.Sequence) {
		return channeltypes.ErrAcknowledgementExists
	}

	if _, found := k.ChannelKeeper.GetPacketReceipt(ctx, packet.DestinationPort, packet.DestinationChannel, packet.Sequence); !found {
		return errorsmod.Wrap(channeltypes.ErrInvalidPacket, "receipt not found for packet")
	}

	if ack == nil {
		return errorsmod.Wrap(channeltypes.ErrInvalidAcknowledgement, "acknowledgement cannot be nil")
	}

	bz := ack.Acknowledgement()
	if len(bz) == 0 {
		return errorsmod.Wrap(channeltypes.ErrInvalidAcknowledgement, "acknowledgement cannot be empty")
	}

	k.ChannelKeeper.SetPacketAcknowledgement(ctx, packet.DestinationPort, packet.DestinationChannel, packet.Sequence, channeltypes.CommitAcknowledgement(bz))

	k.Logger(ctx).Info("acknowledgement written", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_port", packet.SourcePort, "src_channel", packet.SourceChannel, "dst_port", packet.DestinationPort, "dst_channel", packet.DestinationChannel)

	channelkeeper.EmitWriteAcknowledgementEvent(ctx, packet, nil, bz)

	return nil
}

// AcknowledgePacket implements the acknowledgement processing logic required by a packet handler.
// The packet is checked for correctness including asserting that the packet was
// sent and received on clients which are counterparties for one another.
// If no packet commitment exists, a no-op error is returned, otherwise
// the acknowledgement provided is verified to have been stored by the counterparty.
// If successful, the packet commitment is deleted and the packet has completed its lifecycle.
func (k Keeper) AcknowledgePacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packet channeltypes.Packet,
	acknowledgement []byte,
	proofAcked []byte,
	proofHeight exported.Height,
) (string, error) {
	packetV2, err := channeltypes.ConvertPacketV1toV2(packet)
	if err != nil {
		return "", err
	}

	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	counterparty, ok := k.GetCounterparty(ctx, packetV2.SourceId)
	if !ok {
		return "", errorsmod.Wrap(types.ErrCounterpartyNotFound, packetV2.SourceId)
	}

	if counterparty.ClientId != packetV2.DestinationId {
		return "", channeltypes.ErrInvalidChannelIdentifier
	}

	commitment := k.ChannelKeeper.GetPacketCommitment(ctx, packetV2.Data[0].SourcePort, packetV2.SourceId, packetV2.Sequence)
	if len(commitment) == 0 {
		// TODO: explicitly using packet(V1) here, as the event structure will remain the same until PacketV2 API is being used.
		channelkeeper.EmitAcknowledgePacketEvent(ctx, packet, nil)

		// This error indicates that the acknowledgement has already been relayed
		// or there is a misconfigured relayer attempting to prove an acknowledgement
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return "", channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypes.CommitPacketV2(packetV2)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return "", errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "commitment bytes are not equal: got (%v), expected (%v)", packetCommitment, commitment)
	}

	path := host.PacketAcknowledgementKey(packetV2.Data[0].DestinationPort, packetV2.DestinationId, packetV2.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		packetV2.SourceId,
		proofHeight,
		0, 0,
		proofAcked,
		merklePath,
		channeltypes.CommitAcknowledgement(acknowledgement),
	); err != nil {
		return "", errorsmod.Wrapf(err, "failed packet acknowledgement verification for client (%s)", packetV2.SourceId)
	}

	k.ChannelKeeper.DeletePacketCommitment(ctx, packetV2.Data[0].SourcePort, packetV2.SourceId, packetV2.Sequence)

	k.Logger(ctx).Info("packet acknowledged", "sequence", strconv.FormatUint(packetV2.Sequence, 10), "src_port", packetV2.Data[0].SourcePort, "src_channel", packetV2.SourceId, "dst_port", packetV2.Data[0].DestinationPort, "dst_channel", packetV2.DestinationId)

	// TODO: explicitly using packet(V1) here, as the event structure will remain the same until PacketV2 API is being used.
	channelkeeper.EmitAcknowledgePacketEvent(ctx, packet, nil)

	return packetV2.Data[0].Payload.Version, nil
}

// TimeoutPacket implements the timeout logic required by a packet handler.
// The packet is checked for correctness including asserting that the packet was
// sent and received on clients which are counterparties for one another.
// If no packet commitment exists, a no-op error is returned, otherwise
// an absence proof of the packet receipt is performed to ensure that the packet
// was never delivered to the counterparty. If successful, the packet commitment
// is deleted and the packet has completed its lifecycle.
func (k Keeper) TimeoutPacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packet channeltypes.Packet,
	proof []byte,
	proofHeight exported.Height,
	_ uint64,
) (string, error) {
	packetV2, err := channeltypes.ConvertPacketV1toV2(packet)
	if err != nil {
		return "", err
	}

	// Lookup counterparty associated with our channel and ensure
	// that the packet was indeed sent by our counterparty.
	counterparty, ok := k.GetCounterparty(ctx, packetV2.SourceId)
	if !ok {
		return "", errorsmod.Wrap(types.ErrCounterpartyNotFound, packetV2.SourceId)
	}

	if counterparty.ClientId != packetV2.DestinationId {
		return "", channeltypes.ErrInvalidChannelIdentifier
	}

	// check that timeout height or timeout timestamp has passed on the other end
	proofTimestamp, err := k.ClientKeeper.GetClientTimestampAtHeight(ctx, packetV2.SourceId, proofHeight)
	if err != nil {
		return "", err
	}

	timeout := channeltypes.NewTimeoutWithTimestamp(packet.GetTimeoutTimestamp())
	if !timeout.Elapsed(clienttypes.ZeroHeight(), proofTimestamp) {
		return "", errorsmod.Wrap(timeout.ErrTimeoutNotReached(proofHeight.(clienttypes.Height), proofTimestamp), "packet timeout not reached")
	}

	// check that the commitment has not been cleared and that it matches the packet sent by relayer
	commitment := k.ChannelKeeper.GetPacketCommitment(ctx, packetV2.Data[0].SourcePort, packetV2.SourceId, packetV2.Sequence)

	if len(commitment) == 0 {
		// TODO: explicitly using packet(V1) here, as the event structure will remain the same until PacketV2 API is being used.
		channelkeeper.EmitTimeoutPacketEvent(ctx, packet, nil)
		// This error indicates that the timeout has already been relayed
		// or there is a misconfigured relayer attempting to prove a timeout
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return "", channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypes.CommitPacketV2(packetV2)
	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return "", errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
	}

	// verify packet receipt absence
	path := host.PacketReceiptKey(packetV2.Data[0].DestinationPort, packetV2.DestinationId, packetV2.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	if err := k.ClientKeeper.VerifyNonMembership(
		ctx,
		packetV2.SourceId,
		proofHeight,
		0, 0,
		proof,
		merklePath,
	); err != nil {
		return "", errorsmod.Wrapf(err, "failed packet receipt absence verification for client (%s)", packet.SourceChannel)
	}

	// delete packet commitment to prevent replay
	k.ChannelKeeper.DeletePacketCommitment(ctx, packetV2.Data[0].SourcePort, packetV2.SourceId, packetV2.Sequence)

	k.Logger(ctx).Info("packet timed out", "sequence", strconv.FormatUint(packetV2.Sequence, 10), "src_port", packetV2.Data[0].SourcePort, "src_channel", packetV2.SourceId, "dst_port", packetV2.Data[0].DestinationPort, "dst_channel", packetV2.DestinationId)

	// TODO: explicitly using packet(V1) here, as the event structure will remain the same until PacketV2 API is being used.
	channelkeeper.EmitTimeoutPacketEvent(ctx, packet, nil)

	return packet.AppVersion, nil
}
