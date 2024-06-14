package lite

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v8/modules/core/lite/keeper"
	"github.com/cosmos/ibc-go/v8/modules/core/lite/types"
)

var _ channeltypes.PacketMsgServer = (*Handler)(nil)

type Handler struct {
	cdc       codec.BinaryCodec
	keeper    keeper.Keeper
	appRouter types.AppRouter
}

func NewHandler(cdc codec.BinaryCodec, channelKeeper types.ChannelKeeper, clientKeeper types.ClientKeeper, appRouter types.AppRouter, clientRouter types.ClientRouter) *Handler {
	k := keeper.NewKeeper(cdc, channelKeeper, clientKeeper, clientRouter)
	return &Handler{
		cdc:       cdc,
		keeper:    *k,
		appRouter: appRouter,
	}
}

// SendPacket implements the MsgServer interface. It creates a new packet
// with the given source port and source channel and sends it over the channel
// end with the given destination port and channel identifiers.
func (h Handler) SendPacket(goCtx context.Context, msg *channeltypes.MsgSendPacket) (*channeltypes.MsgSendPacketResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sequence, err := h.keeper.SendPacket(ctx, nil, msg.SourcePort, msg.SourceChannel, msg.DestPort, msg.DestChannel,
		*msg.TimeoutHeight, msg.TimeoutTimestamp, msg.Data)
	if err != nil {
		return nil, err
	}

	// IBC Lite routes to the application to do specific sendpacket logic rather than assuming the caller is the application module.
	// IMPORTANT: This changes the ordering of core and application execution for SendPacket
	// Port should directly correspond to the application module route
	// No need for capabilities and mapping from portID to ModuleName
	appModule, ok := h.appRouter.GetRoute(msg.SourcePort)
	if !ok {
		return nil, porttypes.ErrInvalidPort
	}

	// Perform application logic callback
	err = appModule.OnSendPacket(ctx, msg.SourcePort, msg.SourceChannel, sequence, "", msg.Data, msg.Signer)
	if err != nil {
		ctx.Logger().Error("send packet failed", "port-id", msg.SourcePort, "channel-id", msg.SourceChannel, "error", errorsmod.Wrap(err, "send packet callback failed"))
		return nil, errorsmod.Wrap(err, "send packet callback failed")
	}

	return &channeltypes.MsgSendPacketResponse{Sequence: sequence}, nil
}

// ReceivePacket implements the MsgServer interface. It receives an incoming
// packet, which was sent over a channel end with the given port and channel
// identifiers, performs all necessary application logic, and then
// acknowledges the packet.
func (h Handler) RecvPacket(goCtx context.Context, msg *channeltypes.MsgRecvPacket) (*channeltypes.MsgRecvPacketResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	packet := msg.Packet

	if err := h.keeper.RecvPacket(ctx, nil, packet, msg.ProofCommitment, msg.ProofHeight); err != nil {
		return nil, err
	}

	// Port should directly correspond to the application module route
	// No need for capabilities and mapping from portID to ModuleName
	appModule, ok := h.appRouter.GetRoute(packet.DestinationPort)
	if !ok {
		return nil, porttypes.ErrInvalidPort
	}

	// TODO: Figure out how to do caching generically without using SDK
	// Perform application logic callback
	//
	// Cache context so that we may discard state changes from callback if the acknowledgement is unsuccessful.
	cacheCtx, writeFn := ctx.CacheContext()
	// TODO: Use signer as string rather than sdk.AccAddress
	// relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	// if err != nil {
	// 	ctx.Logger().Error("acknowledgement failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
	// 	return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	// }

	ack := appModule.OnRecvPacket(cacheCtx, "", packet, msg.Signer)
	if ack == nil || ack.Success() {
		// write application state changes for asynchronous and successful acknowledgements
		writeFn()
	} else { //nolint
		// Modify events in cached context to reflect unsuccessful acknowledgement
		// TODO: How do we create interface for this that isn't too SDK specific?
		// ctx.EventManager().EmitEvents(convertToErrorEvents(cacheCtx.EventManager().Events()))
	}

	// Write acknowledgement to store
	if ack != nil {
		if err := h.keeper.WriteAcknowledgement(ctx, nil, packet, ack); err != nil {
			return nil, err
		}
	}

	return &channeltypes.MsgRecvPacketResponse{Result: channeltypes.SUCCESS}, nil
}

// Acknowledgement implements the MsgServer interface. It processes the acknowledgement
// of a packet previously sent by the calling chain once the packet has been received and acknowledged
// by the counterparty chain.
func (h Handler) Acknowledgement(goCtx context.Context, msg *channeltypes.MsgAcknowledgement) (*channeltypes.MsgAcknowledgementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	packet := msg.Packet

	if err := h.keeper.AcknowledgePacket(ctx, nil, packet, msg.Acknowledgement, msg.ProofAcked, msg.ProofHeight); err != nil {
		return nil, err
	}

	// TODO: emit events
	// emitAcknowledgePacketEvent(ctx, packet, channel)

	// Port should directly correspond to the application module route
	// No need for capabilities and mapping from portID to ModuleName
	appModule, ok := h.appRouter.GetRoute(packet.SourcePort)
	if !ok {
		return nil, porttypes.ErrInvalidPort
	}

	// relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	// if err != nil {
	// 	ctx.Logger().Error("acknowledgement failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
	// 	return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	// }
	// TODO: Use context instead of sdk.Context eventually
	err := appModule.OnAcknowledgementPacket(ctx, "", packet, msg.Acknowledgement, msg.Signer)
	if err != nil {
		ctx.Logger().Error("acknowledgement failed", "port-id", packet.SourcePort, "channel-id", packet.SourceChannel, "error", errorsmod.Wrap(err, "acknowledge packet callback failed"))
		return nil, errorsmod.Wrap(err, "acknowledge packet callback failed")
	}

	return &channeltypes.MsgAcknowledgementResponse{Result: channeltypes.SUCCESS}, nil
}

// Timeout implements the MsgServer interface. It processes a timeout
// for a packet previously sent by the calling chain.
func (h Handler) Timeout(goCtx context.Context, msg *channeltypes.MsgTimeout) (*channeltypes.MsgTimeoutResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	packet := msg.Packet

	if err := h.keeper.TimeoutPacket(ctx, packet, msg.ProofUnreceived, msg.ProofHeight, 0); err != nil {
		return nil, err
	}

	// Port should directly correspond to the application module route
	// No need for capabilities and mapping from portID to ModuleName
	appModule, ok := h.appRouter.GetRoute(packet.SourcePort)
	if !ok {
		return nil, porttypes.ErrInvalidPort
	}
	// relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	// if err != nil {
	// 	ctx.Logger().Error("acknowledgement failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
	// 	return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	// }
	// Perform application logic callback
	// TODO: Use context instead of sdk.Context eventually
	err := appModule.OnTimeoutPacket(ctx, "", packet, msg.Signer)
	if err != nil {
		ctx.Logger().Error("timeout failed", "port-id", packet.SourcePort, "channel-id", packet.SourceChannel, "error", errorsmod.Wrap(err, "timeout packet callback failed"))
		return nil, errorsmod.Wrap(err, "timeout packet callback failed")
	}

	return &channeltypes.MsgTimeoutResponse{Result: channeltypes.SUCCESS}, nil
}
