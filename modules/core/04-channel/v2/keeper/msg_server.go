package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
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

func (*Keeper) Acknowledgement(ctx context.Context, acknowledgement *channeltypesv2.MsgAcknowledgement) (*channeltypesv2.MsgAcknowledgementResponse, error) {
	panic("implement me")
}

// RecvPacket implements the PacketMsgServer RecvPacket method.
func (k *Keeper) RecvPacket(ctx context.Context, msg *channeltypesv2.MsgRecvPacket) (*channeltypesv2.MsgRecvPacketResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := k.recvPacket(ctx, msg.Packet, msg.ProofCommitment, msg.ProofHeight)
	if err != nil {
		sdkCtx.Logger().Error("receive packet failed", "source-id", msg.Packet.SourceId, "dest-id", msg.Packet.DestinationId, "error", errorsmod.Wrap(err, "send packet failed"))
		return nil, errorsmod.Wrapf(err, "receive packet failed for source id: %s and destination id: %s", msg.Packet.SourceId, msg.Packet.DestinationId)
	}

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		sdkCtx.Logger().Error("receive packet failed", "error", errorsmod.Wrap(err, "invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "invalid address for msg Signer")
	}

	_ = signer

	// TODO: implement once app router is wired up.
	// https://github.com/cosmos/ibc-go/issues/7384
	// for _, pd := range packet.PacketData {
	//	cbs := k.PortKeeper.AppRouter.Route(pd.SourcePort)
	//	err := cbs.OnRecvPacket(ctx, packet, msg.ProofCommitment, msg.ProofHeight, signer)
	//	if err != nil {
	//		return nil, err
	//	}
	// }

	return &channeltypesv2.MsgRecvPacketResponse{Result: types.SUCCESS}, nil
}

// Timeout implements the PacketMsgServer Timeout method.
func (*Keeper) Timeout(ctx context.Context, timeout *channeltypesv2.MsgTimeout) (*channeltypesv2.MsgTimeoutResponse, error) {
	panic("implement me")
}
