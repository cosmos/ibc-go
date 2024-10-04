package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

var _ channeltypesv2.PacketMsgServer = &Keeper{}

// SendPacket implements the PacketMsgServer SendPacket method.
func (k *Keeper) SendPacket(ctx context.Context, msg *channeltypesv2.MsgSendPacket) (*channeltypesv2.MsgSendPacketResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sequence, err := k.sendPacket(ctx, msg.SourceId, msg.TimeoutTimestamp, msg.PacketData)
	if err != nil {
		sdkCtx.Logger().Error("send packet failed", "source-id", msg.SourceId, "error", errorsmod.Wrap(err, "send packet failed"))
		return nil, errorsmod.Wrapf(err, "send packet failed for source id: %s", msg.SourceId)
	}

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		sdkCtx.Logger().Error("send packet failed", "error", errorsmod.Wrap(err, "invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "invalid address for msg Signer")
	}

	_ = signer

	// TODO: implement once app router is wired up.
	// https://github.com/cosmos/ibc-go/issues/7384
	// for _, pd := range msg.PacketData {
	//	cbs := k.PortKeeper.AppRouter.Route(pd.SourcePort)
	//	err := cbs.OnSendPacket(ctx, msg.SourceId, sequence, msg.TimeoutTimestamp, pd, signer)
	//	if err != nil {
	//		return nil, err
	//	}
	// }

	return &channeltypesv2.MsgSendPacketResponse{Sequence: sequence}, nil
}

func (k Keeper) Acknowledgement(ctx context.Context, acknowledgement *channeltypesv2.MsgAcknowledgement) (*channeltypesv2.MsgAcknowledgementResponse, error) {
	panic("implement me")
}

// RecvPacket implements the PacketMsgServer RecvPacket method.
func (k *Keeper) RecvPacket(ctx context.Context, packet *channeltypesv2.MsgRecvPacket) (*channeltypesv2.MsgRecvPacketResponse, error) {
	panic("implement me")
}

// Timeout implements the PacketMsgServer Timeout method.
func (k *Keeper) Timeout(ctx context.Context, timeout *channeltypesv2.MsgTimeout) (*channeltypesv2.MsgTimeoutResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err := k.TimeoutPacket(ctx, timeout.Packet, timeout.ProofUnreceived, timeout.ProofHeight); err != nil {
		sdkCtx.Logger().Error("Timeout packet failed", "source-id", timeout.Packet.SourceId, "error", errorsmod.Wrap(err, "timeout packet failed"))
		return nil, errorsmod.Wrapf(err, "send packet failed for source id: %s", timeout.Packet.SourceId)
	}

	signer, err := sdk.AccAddressFromBech32(timeout.Signer)
	if err != nil {
		sdkCtx.Logger().Error("timeout packet failed", "error", errorsmod.Wrap(err, "invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "invalid address for msg Signer")
	}

	_ = signer

	// TODO: implement once app router is wired up.
	// https://github.com/cosmos/ibc-go/issues/7384
	// for _, pd := range timeout.Packet.Data {
	// 	cbs := k.PortKeeper.AppRouter.Route(pd.SourcePort)
	// 	err := cbs.OnTimeoutPacket(timeout.Packet.SourceId, timeout.Packet.TimeoutTimestamp, signer)
	// 	if err != nil {
	// 		return err, err
	// 	}
	// }

	return &channeltypesv2.MsgTimeoutResponse{Result: channeltypesv1.SUCCESS}, nil
}
