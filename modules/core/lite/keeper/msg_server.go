package keeper

import (
	"bytes"
	"context"

	"cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/lite/types"
)

var _ channeltypes.PacketMsgServer = (*Keeper)(nil)

type Keeper struct {
	cdc                 codec.BinaryCodec
	channelStoreService store.KVStoreService
	clientStoreService  store.KVStoreService
	channelKeeper       types.ChannelKeeper
	clientRouter        types.ClientRouter
	appRouter           porttypes.Router
}

func NewKeeper(cdc codec.BinaryCodec) *Keeper {
	return &Keeper{
		cdc: cdc,
	}
}

// SendPacket implements the MsgServer interface. It creates a new packet
// with the given source port and source channel and sends it over the channel
// end with the given destination port and channel identifiers.
func (k Keeper) SendPacket(goCtx context.Context, msg *channeltypes.MsgSendPacket) (*channeltypes.MsgSendPacketResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get LightClientModule associated with the destination channel
	// Note: This can be implemented by the current clientRouter
	lightClientModule := k.clientRouter.Route(msg.SourceChannel)
	if lightClientModule == nil {
		return nil, errorsmod.Wrapf(channeltypes.ErrInvalidChannel, "source channel %s not associated with a light client module", msg.SourceChannel)
	}

	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current channelKeeper
	// TODO: Use context instead of sdk.Context eventually
	_, counterpartyChannelId, found := k.channelKeeper.GetCounterparty(ctx, "", msg.SourceChannel)
	if !found {
		return nil, channeltypes.ErrChannelNotFound
	}

	if counterpartyChannelId != msg.DestChannel {
		return nil, channeltypes.ErrInvalidChannelIdentifier
	}

	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, msg.SourcePort, msg.SourceChannel)
	if !found {
		sequence = 1
	}

	// construct packet from given fields and channel state
	packet := channeltypes.NewPacket(msg.Data, sequence, msg.SourcePort, msg.SourceChannel,
		msg.DestPort, msg.DestChannel, *msg.TimeoutHeight, msg.TimeoutTimestamp)

	if err := packet.ValidateBasic(); err != nil {
		return nil, errorsmod.Wrap(err, "constructed packet failed basic validation")
	}

	latestHeight := lightClientModule.LatestHeight(ctx, msg.SourceChannel).(clienttypes.Height)
	if latestHeight.IsZero() {
		return nil, errorsmod.Wrapf(clienttypes.ErrInvalidHeight, "cannot send packet using client (%s) with zero height", msg.SourceChannel)
	}

	latestTimestamp, err := lightClientModule.TimestampAtHeight(ctx, msg.SourceChannel, latestHeight)
	if err != nil {
		return nil, err
	}

	// check if packet is timed out on the receiving chain
	timeout := channeltypes.NewTimeout(packet.GetTimeoutHeight().(clienttypes.Height), packet.GetTimeoutTimestamp())
	if timeout.Elapsed(latestHeight, latestTimestamp) {
		return nil, errorsmod.Wrap(timeout.ErrTimeoutElapsed(latestHeight, latestTimestamp), "invalid packet timeout")
	}

	commitment := channeltypes.CommitLitePacket(k.cdc, packet)

	k.channelKeeper.SetNextSequenceSend(ctx, msg.SourcePort, msg.SourceChannel, sequence+1)
	k.channelKeeper.SetPacketCommitment(ctx, msg.SourcePort, msg.SourceChannel, packet.GetSequence(), commitment)

	// IBC Lite routes to the application to do specific sendpacket logic rather than assuming the caller is the application module.
	// IMPORTANT: This changes the ordering of core and application execution for SendPacket
	// TODO: Add SendPacket callback to IBCModule interface

	return &channeltypes.MsgSendPacketResponse{Sequence: packet.GetSequence()}, nil
}

// ReceivePacket implements the MsgServer interface. It receives an incoming
// packet, which was sent over a channel end with the given port and channel
// identifiers, performs all necessary application logic, and then
// acknowledges the packet.
func (k Keeper) RecvPacket(goCtx context.Context, msg *channeltypes.MsgRecvPacket) (*channeltypes.MsgRecvPacketResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	packet := msg.Packet

	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current channelKeeper
	// TODO: Use context instead of sdk.Context eventually
	_, counterpartyChannelId, found := k.channelKeeper.GetCounterparty(ctx, "", packet.DestinationChannel)
	if !found {
		return nil, channeltypes.ErrChannelNotFound
	}

	if counterpartyChannelId != packet.SourceChannel {
		return nil, channeltypes.ErrInvalidChannelIdentifier
	}

	// create key/value pair for proof verification
	key := host.PacketCommitmentPath(packet.SourcePort, packet.SourceChannel, packet.Sequence)
	commitment := channeltypes.CommitLitePacket(k.cdc, packet)

	// Get LightClientModule associated with the destination channel
	// Note: This can be implemented by the current clientRouter
	lightClientModule := k.clientRouter.Route(packet.DestinationChannel)

	// TODO: Use context instead of sdk.Context eventually
	if err := lightClientModule.VerifyMembership(
		ctx,
		packet.DestinationChannel,
		msg.ProofHeight,
		0, 0,
		msg.ProofCommitment,
		commitmenttypes.NewMerklePath(key),
		commitment,
	); err != nil {
		return nil, err
	}

	// Port should directly correspond to the application module route
	// No need for capabilities and mapping from portID to ModuleName
	appModule, ok := k.appRouter.GetRoute(packet.DestinationPort)
	if !ok {
		return nil, porttypes.ErrInvalidPort
	}

	// TODO: Figure out how to do caching generically without using SDK
	// Perform application logic callback
	//
	// Cache context so that we may discard state changes from callback if the acknowledgement is unsuccessful.
	cacheCtx, writeFn := ctx.CacheContext()
	// TODO: Use signer as string rather than sdk.AccAddress
	relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		ctx.Logger().Error("acknowledgement failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	}
	ack := appModule.OnRecvPacket(cacheCtx, packet, relayer)
	if ack == nil || ack.Success() {
		// write application state changes for asynchronous and successful acknowledgements
		writeFn()
	} else {
		// Modify events in cached context to reflect unsuccessful acknowledgement
		// TODO: How do we create interface for this that isn't too SDK specific?
		// ctx.EventManager().EmitEvents(convertToErrorEvents(cacheCtx.EventManager().Events()))
	}

	// Write acknowledgement to store
	if ack != nil {
		// Can be implemented by current channelKeeper with change in sdk.Context to context.Context
		k.channelKeeper.WriteAcknowledgement(ctx, packet.DestinationPort, packet.DestinationChannel, ack.Acknowledgement())
	}

	return &channeltypes.MsgRecvPacketResponse{Result: channeltypes.SUCCESS}, nil
}

// Acknowledgement implements the MsgServer interface. It processes the acknowledgement
// of a packet previously sent by the calling chain once the packet has been received and acknowledged
// by the counterparty chain.
func (k Keeper) Acknowledgement(goCtx context.Context, msg *channeltypes.MsgAcknowledgement) (*channeltypes.MsgAcknowledgementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	packet := msg.Packet

	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current channelKeeper
	// TODO: Use context instead of sdk.Context eventually
	_, counterpartyChannelId, found := k.channelKeeper.GetCounterparty(ctx, "", packet.SourceChannel)
	if !found {
		return nil, channeltypes.ErrChannelNotFound
	}

	if counterpartyChannelId != packet.DestinationChannel {
		return nil, channeltypes.ErrInvalidChannelIdentifier
	}

	commitment := k.channelKeeper.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	if len(commitment) == 0 {
		// TODO: events
		// emitAcknowledgePacketEvent(ctx, packet, channel)

		// This error indicates that the acknowledgement has already been relayed
		// or there is a misconfigured relayer attempting to prove an acknowledgement
		// for a packet never sent. Core IBC will treat this error as a no-op in order to
		// prevent an entire relay transaction from failing and consuming unnecessary fees.
		return nil, channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypes.CommitLitePacket(k.cdc, packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return nil, errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "commitment bytes are not equal: got (%v), expected (%v)", packetCommitment, commitment)
	}

	proofPath := commitmenttypes.NewMerklePath(host.PacketAcknowledgementPath(packet.DestinationPort, packet.DestinationChannel, packet.Sequence))

	// Get LightClientModule associated with the destination channel
	// Note: This can be implemented by the current clientRouter
	lightClientModule := k.clientRouter.Route(packet.SourceChannel)
	// TODO: Use context instead of sdk.Context eventually
	if err := lightClientModule.VerifyMembership(
		ctx,
		packet.SourceChannel,
		msg.ProofHeight,
		0, 0,
		msg.ProofAcked,
		proofPath,
		channeltypes.CommitAcknowledgement(msg.Acknowledgement),
	); err != nil {
		return nil, err
	}

	k.channelKeeper.DeletePacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

	// TODO: emit events
	// emitAcknowledgePacketEvent(ctx, packet, channel)

	// Port should directly correspond to the application module route
	// No need for capabilities and mapping from portID to ModuleName
	appModule, ok := k.appRouter.GetRoute(packet.SourcePort)
	if !ok {
		return nil, porttypes.ErrInvalidPort
	}

	relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		ctx.Logger().Error("acknowledgement failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	}
	// TODO: Use context instead of sdk.Context eventually
	err = appModule.OnAcknowledgementPacket(ctx, packet, msg.Acknowledgement, relayer)
	if err != nil {
		ctx.Logger().Error("acknowledgement failed", "port-id", packet.SourcePort, "channel-id", packet.SourceChannel, "error", errorsmod.Wrap(err, "acknowledge packet callback failed"))
		return nil, errorsmod.Wrap(err, "acknowledge packet callback failed")
	}

	return &channeltypes.MsgAcknowledgementResponse{Result: channeltypes.SUCCESS}, nil
}

// Timeout implements the MsgServer interface. It processes a timeout
// for a packet previously sent by the calling chain.
func (k Keeper) Timeout(goCtx context.Context, msg *channeltypes.MsgTimeout) (*channeltypes.MsgTimeoutResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	packet := msg.Packet

	// Lookup counterparty associated with our channel and ensure that it was packet was indeed
	// sent by our counterparty.
	// Note: This can be implemented by the current channelKeeper
	// TODO: Use context instead of sdk.Context eventually
	_, counterpartyChannelId, found := k.channelKeeper.GetCounterparty(ctx, "", packet.SourceChannel)
	if !found {
		return nil, channeltypes.ErrChannelNotFound
	}

	if counterpartyChannelId != packet.DestinationChannel {
		return nil, channeltypes.ErrInvalidChannelIdentifier
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
		return nil, channeltypes.ErrNoOpMsg
	}

	packetCommitment := channeltypes.CommitLitePacket(k.cdc, packet)

	// verify we sent the packet and haven't cleared it out yet
	if !bytes.Equal(commitment, packetCommitment) {
		return nil, errorsmod.Wrapf(channeltypes.ErrInvalidPacket, "packet commitment bytes are not equal: got (%v), expected (%v)", commitment, packetCommitment)
	}

	proofPath := commitmenttypes.NewMerklePath(host.PacketReceiptPath(packet.DestinationPort, packet.DestinationChannel, packet.Sequence))

	// Get LightClientModule associated with the destination channel
	// Note: This can be implemented by the current clientRouter
	lightClientModule := k.clientRouter.Route(packet.SourceChannel)
	// TODO: Use context instead of sdk.Context eventually
	if err := lightClientModule.VerifyNonMembership(
		ctx,
		packet.DestinationChannel,
		msg.ProofHeight,
		0, 0,
		msg.ProofUnreceived,
		proofPath,
	); err != nil {
		return nil, err
	}

	// TODO: Use context instead of sdk.Context eventually
	k.channelKeeper.DeletePacketCommitment(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)

	// TODO: emit an event marking that we have processed the timeout
	// emitTimeoutPacketEvent(ctx, packet, channel)

	// Port should directly correspond to the application module route
	// No need for capabilities and mapping from portID to ModuleName
	appModule, ok := k.appRouter.GetRoute(packet.SourcePort)
	if !ok {
		return nil, porttypes.ErrInvalidPort
	}
	relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		ctx.Logger().Error("acknowledgement failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	}
	// Perform application logic callback
	// TODO: Use context instead of sdk.Context eventually
	err = appModule.OnTimeoutPacket(ctx, packet, relayer)
	if err != nil {
		ctx.Logger().Error("timeout failed", "port-id", packet.SourcePort, "channel-id", packet.SourceChannel, "error", errorsmod.Wrap(err, "timeout packet callback failed"))
		return nil, errorsmod.Wrap(err, "timeout packet callback failed")
	}

	return &channeltypes.MsgTimeoutResponse{Result: channeltypes.SUCCESS}, nil
}
