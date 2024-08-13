package keeper

import (
	"bytes"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channelkeeper "github.com/cosmos/ibc-go/v9/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	ChannelKeeper types.ChannelKeeper
	ClientKeeper  types.ClientKeeper
}

func NewKeeper(cdc codec.BinaryCodec, channelKeeper types.ChannelKeeper, clientKeeper types.ClientKeeper) *Keeper {
	return &Keeper{
		cdc:           cdc,
		ChannelKeeper: channelKeeper,
		ClientKeeper:  clientKeeper,
	}
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx sdk.Context) log.Logger {
	// TODO: prefix some submodule identifier?
	return ctx.Logger().With("module", "x/"+exported.ModuleName)
}

// TODO: add godoc
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
	counterparty, ok := k.ClientKeeper.GetCounterparty(ctx, sourceChannel)
	if !ok {
		return 0, channeltypes.ErrChannelNotFound
	}
	destChannel := counterparty.ClientId

	// retrieve the sequence send for this channel
	// if no packets have been sent yet, initialize the sequence to 1.
	sequence, found := k.ChannelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		sequence = 1
	}

	// construct packet from given fields and channel state
	packet := channeltypes.NewPacketWithVersion(data, sequence, sourcePort, sourceChannel,
		destPort, destChannel, timeoutHeight, timeoutTimestamp, version)

	if err := packet.ValidateBasic(); err != nil {
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
	timeout := channeltypes.NewTimeout(packet.GetTimeoutHeight().(clienttypes.Height), packet.GetTimeoutTimestamp())
	if timeout.Elapsed(latestHeight, latestTimestamp) {
		return 0, errorsmod.Wrap(timeout.ErrTimeoutElapsed(latestHeight, latestTimestamp), "invalid packet timeout")
	}

	commitment := channeltypes.CommitPacket(packet)

	// bump the sequence and set the packet commitment so it is provable by the counterparty
	k.ChannelKeeper.SetNextSequenceSend(ctx, sourcePort, sourceChannel, sequence+1)
	k.ChannelKeeper.SetPacketCommitment(ctx, sourcePort, sourceChannel, packet.GetSequence(), commitment)

	// log that a packet has been sent
	k.Logger(ctx).Info("packet sent", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_port", packet.SourcePort, "src_channel", packet.SourceChannel, "dst_port", packet.DestinationPort, "dst_channel", packet.DestinationChannel)

	channelkeeper.EmitSendPacketEvent(ctx, packet, sentinelChannel(sourceChannel), timeoutHeight)

	// return the sequence
	return sequence, nil
}

// TODO: add godoc
func (k Keeper) RecvPacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packet channeltypes.Packet,
	proof []byte,
	proofHeight exported.Height,
) (string, error) {
	if packet.ProtocolVersion != channeltypes.IBC_VERSION_2 {
		return "", channeltypes.ErrInvalidPacket
	}

	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current keeper
	counterparty, ok := k.ClientKeeper.GetCounterparty(ctx, packet.DestinationChannel)
	if !ok {
		return "", channeltypes.ErrChannelNotFound
	}
	if counterparty.ClientId != packet.SourceChannel {
		return "", channeltypes.ErrInvalidChannelIdentifier
	}

	// check if packet timed out by comparing it with the latest height of the chain
	selfHeight, selfTimestamp := clienttypes.GetSelfHeight(ctx), uint64(ctx.BlockTime().UnixNano())
	timeout := channeltypes.NewTimeout(packet.GetTimeoutHeight().(clienttypes.Height), packet.GetTimeoutTimestamp())
	if timeout.Elapsed(selfHeight, selfTimestamp) {
		return "", errorsmod.Wrap(timeout.ErrTimeoutElapsed(selfHeight, selfTimestamp), "packet timeout elapsed")
	}

	// REPLAY PROTECTION: Packet receipts will indicate that a packet has already been received
	// on unordered channels. Packet receipts must not be pruned, unless it has been marked stale
	// by the increase of the recvStartSequence.
	_, found := k.ChannelKeeper.GetPacketReceipt(ctx, packet.DestinationPort, packet.DestinationChannel, packet.Sequence)
	if found {
		channelkeeper.EmitRecvPacketEvent(ctx, packet, sentinelChannel(packet.DestinationChannel))
		// This error indicates that the packet has already been relayed. Core IBC will
		// treat this error as a no-op in order to prevent an entire relay transaction
		// from failing and consuming unnecessary fees.
		return "", channeltypes.ErrNoOpMsg
	}

	path := host.PacketCommitmentKey(packet.SourcePort, packet.SourceChannel, packet.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	commitment := channeltypes.CommitPacket(packet)

	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		packet.DestinationChannel,
		proofHeight,
		0, 0,
		proof,
		merklePath,
		commitment,
	); err != nil {
		return "", errorsmod.Wrapf(err, "failed packet commitment verification for client (%s)", packet.DestinationChannel)
	}

	// Set Packet Receipt to prevent timeout from occurring on counterparty
	k.ChannelKeeper.SetPacketReceipt(ctx, packet.DestinationPort, packet.DestinationChannel, packet.Sequence)

	// log that a packet has been received & executed
	k.Logger(ctx).Info("packet received", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_port", packet.SourcePort, "src_channel", packet.SourceChannel, "dst_port", packet.DestinationPort, "dst_channel", packet.DestinationChannel)

	// emit the same events as receive packet without channel fields
	channelkeeper.EmitRecvPacketEvent(ctx, packet, sentinelChannel(packet.DestinationChannel))

	return packet.AppVersion, nil
}

// TODO: add godoc
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

	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	counterparty, ok := k.ClientKeeper.GetCounterparty(ctx, packet.DestinationChannel)
	if !ok {
		return channeltypes.ErrChannelNotFound
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

	// log that a packet acknowledgement has been written
	k.Logger(ctx).Info("acknowledgement written", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_port", packet.SourcePort, "src_channel", packet.SourceChannel, "dst_port", packet.DestinationPort, "dst_channel", packet.DestinationChannel)

	// emit the same events as write acknowledgement without channel fields
	channelkeeper.EmitWriteAcknowledgementEvent(ctx, packet, sentinelChannel(packet.DestinationChannel), bz)

	return nil
}

// TODO: add godoc
func (k Keeper) AcknowledgePacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packet channeltypes.Packet,
	acknowledgement []byte,
	proofAcked []byte,
	proofHeight exported.Height,
) (string, error) {
	if packet.ProtocolVersion != channeltypes.IBC_VERSION_2 {
		return "", channeltypes.ErrInvalidPacket
	}

	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	counterparty, ok := k.ClientKeeper.GetCounterparty(ctx, packet.SourceChannel)
	if !ok {
		return "", channeltypes.ErrChannelNotFound
	}

	if counterparty.ClientId != packet.DestinationChannel {
		return "", channeltypes.ErrInvalidChannelIdentifier
	}

	commitment := k.ChannelKeeper.GetPacketCommitment(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)
	if len(commitment) == 0 {
		channelkeeper.EmitAcknowledgePacketEvent(ctx, packet, sentinelChannel(packet.SourceChannel))

		// This error indicates that the acknowledgement has already been relayed
		// or there is a misconfigured relayer attempting to prove an acknowledgement
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return "", channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypes.CommitPacket(packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return "", errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "commitment bytes are not equal: got (%v), expected (%v)", packetCommitment, commitment)
	}

	path := host.PacketAcknowledgementKey(packet.DestinationPort, packet.DestinationChannel, packet.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	if err := k.ClientKeeper.VerifyMembership(
		ctx,
		packet.SourceChannel,
		proofHeight,
		0, 0,
		proofAcked,
		merklePath,
		channeltypes.CommitAcknowledgement(acknowledgement),
	); err != nil {
		return "", errorsmod.Wrapf(err, "failed packet acknowledgement verification for client (%s)", packet.SourceChannel)
	}

	k.ChannelKeeper.DeletePacketCommitment(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)

	// log that a packet has been acknowledged
	k.Logger(ctx).Info("packet acknowledged", "sequence", strconv.FormatUint(packet.GetSequence(), 10), "src_port", packet.GetSourcePort(), "src_channel", packet.GetSourceChannel(), "dst_port", packet.GetDestPort(), "dst_channel", packet.GetDestChannel())

	// emit the same events as acknowledge packet without channel fields
	channelkeeper.EmitAcknowledgePacketEvent(ctx, packet, sentinelChannel(packet.SourceChannel))

	return packet.AppVersion, nil
}

// TODO: add godoc
func (k Keeper) TimeoutPacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packet channeltypes.Packet,
	proof []byte,
	proofHeight exported.Height,
	_ uint64,
) (string, error) {
	if packet.ProtocolVersion != channeltypes.IBC_VERSION_2 {
		return "", channeltypes.ErrInvalidPacket
	}
	// Lookup counterparty associated with our channel and ensure that destination channel
	// is the expected counterparty
	counterparty, ok := k.ClientKeeper.GetCounterparty(ctx, packet.SourceChannel)
	if !ok {
		return "", channeltypes.ErrChannelNotFound
	}

	if counterparty.ClientId != packet.DestinationChannel {
		return "", channeltypes.ErrInvalidChannelIdentifier
	}

	// check that timeout height or timeout timestamp has passed on the other end
	proofTimestamp, err := k.ClientKeeper.GetClientTimestampAtHeight(ctx, packet.SourceChannel, proofHeight)
	if err != nil {
		return "", err
	}

	timeout := channeltypes.NewTimeout(packet.GetTimeoutHeight().(clienttypes.Height), packet.GetTimeoutTimestamp())
	if !timeout.Elapsed(proofHeight.(clienttypes.Height), proofTimestamp) {
		return "", errorsmod.Wrap(timeout.ErrTimeoutNotReached(proofHeight.(clienttypes.Height), proofTimestamp), "packet timeout not reached")
	}

	// check that the commitment has not been cleared and that it matches the packet sent by relayer
	commitment := k.ChannelKeeper.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	if len(commitment) == 0 {
		channelkeeper.EmitTimeoutPacketEvent(ctx, packet, sentinelChannel(packet.SourceChannel))
		// This error indicates that the timeout has already been relayed
		// or there is a misconfigured relayer attempting to prove a timeout
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return "", channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypes.CommitPacket(packet)
	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return "", errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
	}

	// verify packet receipt absence
	path := host.PacketReceiptKey(packet.DestinationPort, packet.DestinationChannel, packet.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	if err := k.ClientKeeper.VerifyNonMembership(
		ctx,
		packet.SourceChannel,
		proofHeight,
		0, 0,
		proof,
		merklePath,
	); err != nil {
		return "", errorsmod.Wrapf(err, "failed packet receipt absence verification for client (%s)", packet.SourceChannel)
	}

	// delete packet commitment to prevent replay
	k.ChannelKeeper.DeletePacketCommitment(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)

	// log that a packet has been timed out
	k.Logger(ctx).Info("packet timed out", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_port", packet.SourcePort, "src_channel", packet.SourceChannel, "dst_port", packet.DestinationPort, "dst_channel", packet.DestinationChannel)

	// emit timeout events
	channelkeeper.EmitTimeoutPacketEvent(ctx, packet, sentinelChannel(packet.SourceChannel))

	return packet.AppVersion, nil
}

// sentinelChannel creates a sentinel channel for use in events for Eureka protocol handlers.
func sentinelChannel(clientID string) channeltypes.Channel {
	return channeltypes.Channel{Ordering: channeltypes.UNORDERED, ConnectionHops: []string{clientID}}
}
