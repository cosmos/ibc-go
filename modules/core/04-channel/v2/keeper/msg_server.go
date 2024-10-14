package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var _ channeltypesv2.MsgServer = &Keeper{}

// SendPacket implements the PacketMsgServer SendPacket method.
func (k *Keeper) SendPacket(ctx context.Context, msg *channeltypesv2.MsgSendPacket) (*channeltypesv2.MsgSendPacketResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sequence, destChannel, err := k.sendPacket(ctx, msg.SourceChannel, msg.TimeoutTimestamp, msg.PacketData)
	if err != nil {
		sdkCtx.Logger().Error("send packet failed", "source-channel", msg.SourceChannel, "error", errorsmod.Wrap(err, "send packet failed"))
		return nil, errorsmod.Wrapf(err, "send packet failed for source id: %s", msg.SourceChannel)
	}

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		sdkCtx.Logger().Error("send packet failed", "error", errorsmod.Wrap(err, "invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "invalid address for msg Signer")
	}

	for _, pd := range msg.PacketData {
		cbs := k.Router.Route(pd.SourcePort)
		err := cbs.OnSendPacket(ctx, msg.SourceChannel, destChannel, sequence, pd, signer)
		if err != nil {
			return nil, err
		}
	}

	return &channeltypesv2.MsgSendPacketResponse{Sequence: sequence}, nil
}

func (k *Keeper) Acknowledgement(ctx context.Context, msg *channeltypesv2.MsgAcknowledgement) (*channeltypesv2.MsgAcknowledgementResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		sdkCtx.Logger().Error("acknowledgement failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	}

	cacheCtx, writeFn := sdkCtx.CacheContext()
	err = k.acknowledgePacket(cacheCtx, msg.Packet, msg.Acknowledgement, msg.ProofAcked, msg.ProofHeight)

	switch err {
	case nil:
		writeFn()
	case channeltypesv1.ErrNoOpMsg:
		// no-ops do not need event emission as they will be ignored
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-channel", msg.Packet.SourceChannel)
		return &channeltypesv2.MsgAcknowledgementResponse{Result: channeltypesv1.NOOP}, nil
	default:
		sdkCtx.Logger().Error("acknowledgement failed", "source-channel", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "acknowledge packet verification failed"))
		return nil, errorsmod.Wrap(err, "acknowledge packet verification failed")
	}

	_ = relayer

	// TODO: implement once app router is wired up.
	// https://github.com/cosmos/ibc-go/issues/7384
	// for _, pd := range msg.PacketData {
	//	cbs := k.PortKeeper.AppRouter.Route(pd.SourcePort)
	//	err := cbs.OnSendPacket(ctx, msg.SourceId, sequence, msg.TimeoutTimestamp, pd, signer)
	//	if err != nil {
	//		return nil, err
	//	}
	// }

	return nil, nil
}

// RecvPacket implements the PacketMsgServer RecvPacket method.
func (k *Keeper) RecvPacket(ctx context.Context, msg *channeltypesv2.MsgRecvPacket) (*channeltypesv2.MsgRecvPacketResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err := k.recvPacket(ctx, msg.Packet, msg.ProofCommitment, msg.ProofHeight)
	if err != nil {
		sdkCtx.Logger().Error("receive packet failed", "source-channel", msg.Packet.SourceChannel, "dest-channel", msg.Packet.DestinationChannel, "error", errorsmod.Wrap(err, "send packet failed"))
		return nil, errorsmod.Wrapf(err, "receive packet failed for source id: %s and destination id: %s", msg.Packet.SourceChannel, msg.Packet.DestinationChannel)
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

	return &channeltypesv2.MsgRecvPacketResponse{Result: channeltypesv1.SUCCESS}, nil
}

// Timeout implements the PacketMsgServer Timeout method.
func (k *Keeper) Timeout(ctx context.Context, timeout *channeltypesv2.MsgTimeout) (*channeltypesv2.MsgTimeoutResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err := k.timeoutPacket(ctx, timeout.Packet, timeout.ProofUnreceived, timeout.ProofHeight); err != nil {
		sdkCtx.Logger().Error("Timeout packet failed", "source-channel", timeout.Packet.SourceChannel, "destination-channel", timeout.Packet.DestinationChannel, "error", errorsmod.Wrap(err, "timeout packet failed"))
		return nil, errorsmod.Wrapf(err, "send packet failed for source id: %s and destination id: %s", timeout.Packet.SourceChannel, timeout.Packet.DestinationChannel)
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
	// 	err := cbs.OnTimeoutPacket(timeout.Packet.SourceChannel, timeout.Packet.TimeoutTimestamp, signer)
	// 	if err != nil {
	// 		return err, err
	// 	}
	// }

	return &channeltypesv2.MsgTimeoutResponse{Result: channeltypesv1.SUCCESS}, nil
}

// CreateChannel defines a rpc handler method for MsgCreateChannel
func (k *Keeper) CreateChannel(goCtx context.Context, msg *channeltypesv2.MsgCreateChannel) (*channeltypesv2.MsgCreateChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	channelID := k.channelKeeperV1.GenerateChannelIdentifier(ctx)

	// Initialize channel with empty counterparty channel identifier.
	channel := channeltypesv2.NewChannel(msg.ClientId, "", msg.MerklePathPrefix)
	k.SetChannel(ctx, channelID, channel)
	k.SetCreator(ctx, channelID, msg.Signer)
	k.SetNextSequenceSend(ctx, channelID, 1)

	k.EmitCreateChannelEvent(goCtx, channelID)

	return &channeltypesv2.MsgCreateChannelResponse{ChannelId: channelID}, nil
}

// ProvideCounterparty defines a rpc handler method for MsgProvideCounterparty.
func (k *Keeper) ProvideCounterparty(goCtx context.Context, msg *channeltypesv2.MsgProvideCounterparty) (*channeltypesv2.MsgProvideCounterpartyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	creator, found := k.GetCreator(ctx, msg.ChannelId)
	if !found {
		return nil, errorsmod.Wrap(ibcerrors.ErrUnauthorized, "channel creator must be set")
	}

	if creator != msg.Signer {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "channel creator (%s) must match signer (%s)", creator, msg.Signer)
	}

	channel, ok := k.GetChannel(ctx, msg.ChannelId)
	if !ok {
		return nil, errorsmod.Wrapf(channeltypesv2.ErrInvalidChannel, "channel must exist for channel id %s", msg.ChannelId)
	}

	channel.CounterpartyChannelId = msg.CounterpartyChannelId
	k.SetChannel(ctx, msg.ChannelId, channel)
	// Delete client creator from state as it is not needed after this point.
	k.DeleteCreator(ctx, msg.ChannelId)

	return &channeltypesv2.MsgProvideCounterpartyResponse{}, nil
}
