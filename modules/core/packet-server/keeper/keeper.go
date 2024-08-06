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
		return 0, errorsmod.Wrap(err, "constructed packet failed basic validation")
	}

	// check that the client of receiver chain is still active
	if status := k.ClientKeeper.GetClientStatus(ctx, sourceChannel); status != exported.Active {
		return 0, errorsmod.Wrapf(clienttypes.ErrClientNotActive, "client state is not active: %s", status)
	}

	// retrieve latest height and timestamp of the client of receiver chain
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

	// create sentinel channel for events
	channel := channeltypes.Channel{
		Ordering:       channeltypes.ORDERED,
		ConnectionHops: []string{sourceChannel},
	}
	channelkeeper.EmitSendPacketEvent(ctx, packet, channel, timeoutHeight)

	// return the sequence
	return sequence, nil
}

func (k Keeper) RecvPacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packet channeltypes.Packet,
	proof []byte,
	proofHeight exported.Height,
) error {
	if packet.ProtocolVersion != channeltypes.IBC_VERSION_2 {
		return channeltypes.ErrInvalidPacket
	}

	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current keeper
	counterparty, ok := k.ClientKeeper.GetCounterparty(ctx, packet.DestinationChannel)
	if !ok {
		return channeltypes.ErrChannelNotFound
	}
	if counterparty.ClientId != packet.SourceChannel {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	// check if packet timed out by comparing it with the latest height of the chain
	selfHeight, selfTimestamp := clienttypes.GetSelfHeight(ctx), uint64(ctx.BlockTime().UnixNano())
	timeout := channeltypes.NewTimeout(packet.GetTimeoutHeight().(clienttypes.Height), packet.GetTimeoutTimestamp())
	if timeout.Elapsed(selfHeight, selfTimestamp) {
		return errorsmod.Wrap(timeout.ErrTimeoutElapsed(selfHeight, selfTimestamp), "packet timeout elapsed")
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
		return channeltypes.ErrNoOpMsg
	}

	// create key/value pair for proof verification by appending the ICS24 path to the last element of the counterparty merklepath
	// TODO: allow for custom prefix
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
		return err
	}

	// Set Packet Receipt to prevent timeout from occurring on counterparty
	k.ChannelKeeper.SetPacketReceipt(ctx, packet.DestinationPort, packet.DestinationChannel, packet.Sequence)

	// log that a packet has been received & executed
	k.Logger(ctx).Info("packet received", "sequence", strconv.FormatUint(packet.Sequence, 10), "src_port", packet.SourcePort, "src_channel", packet.SourceChannel, "dst_port", packet.DestinationPort, "dst_channel", packet.DestinationChannel)

	// emit the same events as receive packet without channel fields
	channelkeeper.EmitRecvPacketEvent(ctx, packet, sentinelChannel(packet.DestinationChannel))

	return nil
}

func (k Keeper) TimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	proof []byte,
	proofHeight exported.Height,
	nextSequenceRecv uint64,
) error {
	// Lookup counterparty associated with our channel and ensure that destination channel
	// is the expected counterparty
	counterparty, ok := k.ClientKeeper.GetCounterparty(ctx, packet.SourceChannel)
	if !ok {
		return channeltypes.ErrChannelNotFound
	}

	if counterparty.ClientId != packet.DestinationChannel {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	// check that timeout height or timeout timestamp has passed on the other end
	proofTimestamp, err := k.ClientKeeper.GetClientTimestampAtHeight(ctx, packet.SourceChannel, proofHeight)
	if err != nil {
		return err
	}

	timeout := channeltypes.NewTimeout(packet.GetTimeoutHeight().(clienttypes.Height), packet.GetTimeoutTimestamp())
	if !timeout.Elapsed(proofHeight.(clienttypes.Height), proofTimestamp) {
		return errorsmod.Wrap(timeout.ErrTimeoutNotReached(proofHeight.(clienttypes.Height), proofTimestamp), "packet timeout not reached")
	}

	// check that the commitment has not been cleared and that it matches the packet sent by relayer
	commitment := k.ChannelKeeper.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	if len(commitment) == 0 {
		channelkeeper.EmitTimeoutPacketEvent(ctx, packet, sentinelChannel(packet.SourceChannel))
		// This error indicates that the timeout has already been relayed
		// or there is a misconfigured relayer attempting to prove a timeout
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypes.CommitPacket(packet)
	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
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
		return err
	}

	// delete packet commitment to prevent replay
	k.ChannelKeeper.DeletePacketCommitment(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)

	// emit timeout events
	channelkeeper.EmitTimeoutPacketEvent(ctx, packet, sentinelChannel(packet.SourceChannel))

	return nil
}

// sentinelChannel creates a sentinel channel for use in events for Eureka protocol handlers.
func sentinelChannel(clientID string) channeltypes.Channel {
	return channeltypes.Channel{Ordering: channeltypes.UNORDERED, ConnectionHops: []string{clientID}}
}
