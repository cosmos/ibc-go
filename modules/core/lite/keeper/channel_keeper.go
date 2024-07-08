package keeper

import (
	"bytes"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channelkeeper "github.com/cosmos/ibc-go/v8/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	"github.com/cosmos/ibc-go/v8/modules/core/lite/types"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	channelKeeper types.ChannelKeeper
	clientKeeper  types.ClientKeeper
	clientRouter  types.ClientRouter
}

func NewKeeper(cdc codec.BinaryCodec, channelKeeper types.ChannelKeeper, clientKeeper types.ClientKeeper, clientRouter types.ClientRouter) *Keeper {
	return &Keeper{
		cdc:           cdc,
		channelKeeper: channelKeeper,
		clientKeeper:  clientKeeper,
		clientRouter:  clientRouter,
	}
}

// Keeper will implement the expected interface of ChannelKeeper
// for keeper.Keeper so that the standard message server can implement
// IBC Lite without major changes to its structure

func (k Keeper) SendPacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	destPort string,
	destChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	// Get LightClientModule associated with the destination channel
	// Note: This can be implemented by the current clientRouter
	lightClientModule, ok := k.clientRouter.GetRoute(sourceChannel)
	if !ok {
		return 0, clienttypes.ErrClientNotFound
	}

	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current keeper
	// TODO: Use context instead of sdk.Context eventually
	counterparty, ok := k.clientKeeper.GetCounterparty(ctx, sourceChannel)
	if !ok {
		return 0, channeltypes.ErrChannelNotFound
	}
	if counterparty.ClientId == "" {
		return 0, channeltypes.ErrChannelNotFound
	}

	if counterparty.ClientId != destChannel {
		return 0, channeltypes.ErrInvalidChannelIdentifier
	}

	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		sequence = 1
	}

	// construct packet from given fields and channel state
	packet := channeltypes.NewPacket(data, sequence, sourcePort, sourceChannel,
		destPort, destChannel, timeoutHeight, timeoutTimestamp)

	if err := packet.ValidateBasic(); err != nil {
		return 0, errorsmod.Wrap(err, "constructed packet failed basic validation")
	}

	latestHeight, ok := lightClientModule.LatestHeight(ctx, sourceChannel).(clienttypes.Height)
	if !ok {
		return 0, errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "latest height of client (%s) is not a %T", sourceChannel, (*clienttypes.Height)(nil))
	}
	if latestHeight.IsZero() {
		return 0, errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "cannot send packet using client (%s) with zero height", sourceChannel)
	}

	latestTimestamp, err := lightClientModule.TimestampAtHeight(ctx, sourceChannel, latestHeight)
	if err != nil {
		return 0, err
	}

	// check if packet is timed out on the receiving chain
	timeout := channeltypes.NewTimeout(packet.GetTimeoutHeight().(clienttypes.Height), packet.GetTimeoutTimestamp())
	if timeout.Elapsed(latestHeight, latestTimestamp) {
		return 0, errorsmod.Wrap(timeout.ErrTimeoutElapsed(latestHeight, latestTimestamp), "invalid packet timeout")
	}

	commitment := channeltypes.CommitLitePacket(k.cdc, packet)

	k.channelKeeper.SetNextSequenceSend(ctx, sourcePort, sourceChannel, sequence+1)
	k.channelKeeper.SetPacketCommitment(ctx, sourcePort, sourceChannel, packet.GetSequence(), commitment)

	// TODO: Abstract away to not require SDK event system
	channelkeeper.EmitSendPacketEvent(ctx, packet, nil)

	return packet.Sequence, nil
}

func (k Keeper) RecvPacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packet channeltypes.Packet,
	proof []byte,
	proofHeight exported.Height,
) error {
	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current keeper
	// TODO: Use context instead of sdk.Context eventually
	counterparty, ok := k.clientKeeper.GetCounterparty(ctx, packet.DestinationChannel)
	if !ok {
		return channeltypes.ErrChannelNotFound
	}
	if counterparty.ClientId == "" {
		return channeltypes.ErrChannelNotFound
	}

	if counterparty.ClientId != packet.SourceChannel {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	// create key/value pair for proof verification by appending the ICS24 path to the last element of the counterparty merklepath

	// TODO: allow for custom prefix
	path := host.PacketCommitmentPath(packet.SourcePort, packet.SourceChannel, packet.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	commitment := channeltypes.CommitLitePacket(k.cdc, packet)

	// Get LightClientModule associated with the destination channel
	// Note: This can be implemented by the current clientRouter
	lightClientModule, ok := k.clientRouter.GetRoute(packet.DestinationChannel)
	if !ok {
		return clienttypes.ErrClientNotFound
	}

	// TODO: Use context instead of sdk.Context eventually
	if err := lightClientModule.VerifyMembership(
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
	k.channelKeeper.SetPacketReceipt(ctx, packet.DestinationPort, packet.DestinationChannel, packet.Sequence)

	// emit the same events as receive packet without channel fields
	channelkeeper.EmitRecvPacketEvent(ctx, packet, nil)

	return nil
}

func (k Keeper) WriteAcknowledgement(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packet exported.PacketI,
	ack exported.Acknowledgement,
) error {
	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current keeper
	// Can be implemented by current keeper with change in sdk.Context to context.Context
	ackHash := channeltypes.CommitAcknowledgement(ack.Acknowledgement())
	k.channelKeeper.SetPacketAcknowledgement(ctx, packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(), ackHash)

	bz := ack.Acknowledgement()
	if len(bz) == 0 {
		return errorsmod.Wrap(channeltypes.ErrInvalidAcknowledgement, "acknowledgement cannot be empty")
	}

	// emit the same events as write acknowledgement without channel fields
	channelkeeper.EmitWriteAcknowledgementEvent(ctx, packet.(channeltypes.Packet), nil, bz)

	return nil
}

func (k Keeper) AcknowledgePacket(
	ctx sdk.Context,
	_ *capabilitytypes.Capability,
	packet channeltypes.Packet,
	acknowledgement []byte,
	proofAcked []byte,
	proofHeight exported.Height,
) error {
	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current keeper
	// TODO: Use context instead of sdk.Context eventually
	counterparty, ok := k.clientKeeper.GetCounterparty(ctx, packet.SourceChannel)
	if !ok {
		return channeltypes.ErrChannelNotFound
	}
	if counterparty.ClientId == "" {
		return channeltypes.ErrChannelNotFound
	}

	if counterparty.ClientId != packet.DestinationChannel {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	commitment := k.channelKeeper.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	if len(commitment) == 0 {
		// TODO: events
		// emitAcknowledgePacketEvent(ctx, packet, channel)

		// This error indicates that the acknowledgement has already been relayed
		// or there is a misconfigured relayer attempting to prove an acknowledgement
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypes.CommitLitePacket(k.cdc, packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "commitment bytes are not equal: got (%v), expected (%v)", packetCommitment, commitment)
	}

	path := host.PacketAcknowledgementPath(packet.DestinationPort, packet.DestinationChannel, packet.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	// Get LightClientModule associated with the destination channel
	// Note: This can be implemented by the current clientRouter
	lightClientModule, ok := k.clientRouter.GetRoute(packet.SourceChannel)
	if !ok {
		return clienttypes.ErrClientNotFound
	}
	// TODO: Use context instead of sdk.Context eventually
	if err := lightClientModule.VerifyMembership(
		ctx,
		packet.SourceChannel,
		proofHeight,
		0, 0,
		proofAcked,
		merklePath,
		channeltypes.CommitAcknowledgement(acknowledgement),
	); err != nil {
		return err
	}

	k.channelKeeper.DeletePacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	// emit the same events as acknowledge packet without channel fields
	channelkeeper.EmitAcknowledgePacketEvent(ctx, packet, nil)
	return nil
}

func (k Keeper) TimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	proofTimeout []byte,
	proofHeight exported.Height,
	_ uint64,
) error {
	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current keeper
	// TODO: Use context instead of sdk.Context eventually
	counterparty, ok := k.clientKeeper.GetCounterparty(ctx, packet.SourceChannel)
	if !ok {
		return channeltypes.ErrChannelNotFound
	}
	if counterparty.ClientId == "" {
		return channeltypes.ErrChannelNotFound
	}

	if counterparty.ClientId != packet.DestinationChannel {
		return channeltypes.ErrInvalidChannelIdentifier
	}

	// TODO: Use context instead of sdk.Context eventually
	commitment := k.channelKeeper.GetPacketCommitment(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)
	if len(commitment) == 0 {
		// TODO: events
		// emitTimeoutPacketEvent(ctx, packet, channel)

		// This error indicates that the timeout has already been relayed
		// or there is a misconfigured relayer attempting to prove a timeout
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypes.CommitLitePacket(k.cdc, packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
	}

	path := host.PacketReceiptPath(packet.DestinationPort, packet.DestinationChannel, packet.Sequence)
	merklePath := types.BuildMerklePath(counterparty.MerklePathPrefix, path)

	// Get LightClientModule associated with the destination channel
	// Note: This can be implemented by the current clientRouter
	lightClientModule, ok := k.clientRouter.GetRoute(packet.SourceChannel)
	if !ok {
		return clienttypes.ErrClientNotFound
	}

	timestamp, err := lightClientModule.TimestampAtHeight(ctx, packet.SourceChannel, proofHeight)
	if err != nil {
		return err
	}
	timeout := channeltypes.NewTimeout(packet.GetTimeoutHeight().(clienttypes.Height), packet.GetTimeoutTimestamp())
	if !timeout.Elapsed(proofHeight.(clienttypes.Height), timestamp) {
		return errorsmod.Wrap(timeout.ErrTimeoutNotReached(proofHeight.(clienttypes.Height), timestamp), "packet timeout not reached")
	}

	// TODO: Use context instead of sdk.Context eventually
	if err := lightClientModule.VerifyNonMembership(
		ctx,
		packet.DestinationChannel,
		proofHeight,
		0, 0,
		proofTimeout,
		merklePath,
	); err != nil {
		return err
	}

	// TODO: Use context instead of sdk.Context eventually
	k.channelKeeper.DeletePacketCommitment(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)

	// emit the same events as timeout packet without channel fields
	channelkeeper.EmitTimeoutPacketEvent(ctx, packet, nil)
	return nil
}
