package keeper

import (
	"bytes"

	errorsmod "cosmossdk.io/errors"

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

func (k Keeper) TimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	proof []byte,
	proofHeight exported.Height,
	nextSequenceRecv uint64,
) error {
	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
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

	commitment := k.ChannelKeeper.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	// create sentinel channel
	channel := channeltypes.Channel{Ordering: channeltypes.UNORDERED, ConnectionHops: []string{packet.SourceChannel}}

	if len(commitment) == 0 {
		channelkeeper.EmitTimeoutPacketEvent(ctx, packet, channel)
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
	channelkeeper.EmitTimeoutPacketEvent(ctx, packet, channel)

	return nil
}
