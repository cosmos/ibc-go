package keeper

import (
	"context"
	"slices"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

var (
	_ channeltypesv2.PacketMsgServer = (*Keeper)(nil)
)

func (k *Keeper) SendPacketV2(ctx context.Context, msg *channeltypesv2.MsgSendPacket) (*channeltypesv2.MsgSendPacketResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sequence, err := k.PacketServerKeeper.SendPacketV2(ctx, msg.SourceId, msg.TimeoutTimestamp, msg.PacketData)
	if err != nil {
		sdkCtx.Logger().Error("send packet failed", "source-id", msg.SourceId, "error", errorsmod.Wrap(err, "send packet failed"))
		return nil, errorsmod.Wrapf(err, "send packet failed for source id: %s", msg.SourceId)
	}

	for _, pd := range msg.PacketData {
		cbs := k.PortKeeper.AppRouter.Route(pd.SourcePort)
		err := cbs.OnSendPacketV2(ctx, msg.SourceId, sequence, msg.TimeoutTimestamp, pd.Payload, sdk.AccAddress(msg.Signer))
		if err != nil {
			return nil, err
		}
	}

	return &channeltypesv2.MsgSendPacketResponse{Sequence: sequence}, nil
}

func (k *Keeper) RecvPacketV2(ctx context.Context, msg *channeltypesv2.MsgRecvPacket) (*channeltypesv2.MsgRecvPacketResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		sdkCtx.Logger().Error("receive packet failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	}

	// Perform TAO verification
	//
	// If the packet was already received, perform a no-op
	// Use a cached context to prevent accidental state changes
	cacheCtx, writeFn := sdkCtx.CacheContext()
	err = k.PacketServerKeeper.RecvPacketV2(cacheCtx, msg.Packet, msg.ProofCommitment, msg.ProofHeight)

	switch err {
	case nil:
		writeFn()
	case channeltypes.ErrNoOpMsg:
		// no-ops do not need event emission as they will be ignored
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-id", msg.Packet.SourceId)
		return &channeltypesv2.MsgRecvPacketResponse{Result: channeltypes.NOOP}, nil
	default:
		sdkCtx.Logger().Error("receive packet failed", "source-id", msg.Packet.SourceId, "error", errorsmod.Wrap(err, "receive packet verification failed"))
		return nil, errorsmod.Wrap(err, "receive packet verification failed")
	}

	// Perform application logic callback
	//
	// Cache context so that we may discard state changes from callback if the acknowledgement is unsuccessful.

	multiAck := channeltypes.MultiAcknowledgement{
		AcknowledgementResults: []channeltypes.AcknowledgementResult{},
	}

	for _, pd := range msg.Packet.Data {
		cacheCtx, writeFn = sdkCtx.CacheContext()
		cb := k.PortKeeper.AppRouter.Route(pd.DestinationPort)
		res := cb.OnRecvPacketV2(cacheCtx, msg.Packet, pd.Payload, relayer)

		if res.Status != channeltypes.PacketStatus_Failure {
			// write application state changes for asynchronous and successful acknowledgements
			writeFn()
		} else {
			// Modify events in cached context to reflect unsuccessful acknowledgement
			sdkCtx.EventManager().EmitEvents(convertToErrorEvents(cacheCtx.EventManager().Events()))
		}

		multiAck.AcknowledgementResults = append(multiAck.AcknowledgementResults, channeltypes.AcknowledgementResult{
			AppName:          pd.DestinationPort,
			RecvPacketResult: res,
		})
	}

	// Set packet acknowledgement only if the acknowledgement is not nil.
	// NOTE: IBC applications modules may call the WriteAcknowledgement asynchronously if the
	// acknowledgement is nil.

	isAsync := slices.ContainsFunc(multiAck.AcknowledgementResults, func(ackResult channeltypes.AcknowledgementResult) bool {
		return ackResult.RecvPacketResult.Status == channeltypes.PacketStatus_Async
	})

	if !isAsync {
		if err := k.PacketServerKeeper.WriteAcknowledgementV2(ctx, msg.Packet, multiAck); err != nil {
			return nil, err
		}
		// TODO: log line
		return &channeltypesv2.MsgRecvPacketResponse{Result: channeltypes.SUCCESS}, nil
	}

	k.ChannelKeeper.SetMultiAcknowledgement(ctx, host.SentinelV2PortID, msg.Packet.DestinationId, msg.Packet.Sequence, multiAck)

	// defer telemetry.ReportRecvPacket(msg.Packet)

	// ctx.Logger().Info("receive packet callback succeeded", "port-id", msg.Packet.SourcePort, "channel-id", msg.Packet.SourceChannel, "result", channeltypes.SUCCESS.String())

	return &channeltypesv2.MsgRecvPacketResponse{Result: channeltypes.SUCCESS}, nil
}

func (k *Keeper) TimeoutV2(ctx context.Context, msg *channeltypesv2.MsgTimeout) (*channeltypesv2.MsgTimeoutResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		sdkCtx.Logger().Error("timeout failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	}

	// Perform TAO verification
	//
	// If the timeout was already received, perform a no-op
	// Use a cached context to prevent accidental state changes
	cacheCtx, writeFn := sdkCtx.CacheContext()
	err = k.PacketServerKeeper.TimeoutPacketV2(cacheCtx, msg.Packet, msg.ProofUnreceived, msg.ProofHeight)

	switch err {
	case nil:
		writeFn()
	case channeltypes.ErrNoOpMsg:
		// no-ops do not need event emission as they will be ignored
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-id", msg.Packet.SourceId)
		return &channeltypesv2.MsgTimeoutResponse{Result: channeltypes.NOOP}, nil
	default:
		sdkCtx.Logger().Error("timeout failed", "source-id", msg.Packet.SourceId, "error", errorsmod.Wrap(err, "timeout packet verification failed"))
		return nil, errorsmod.Wrap(err, "timeout packet verification failed")
	}

	for _, pd := range msg.Packet.Data {
		cb := k.PortKeeper.AppRouter.Route(pd.SourcePort)
		err := cb.OnTimeoutPacketV2(ctx, msg.Packet, pd.Payload, relayer)
		if err != nil {
			return nil, err
		}
	}

	return &channeltypesv2.MsgTimeoutResponse{Result: channeltypes.SUCCESS}, nil
}

func (k *Keeper) AcknowledgementV2(ctx context.Context, msg *channeltypesv2.MsgAcknowledgement) (*channeltypesv2.MsgAcknowledgementResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	relayer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		sdkCtx.Logger().Error("acknowledgement failed", "error", errorsmod.Wrap(err, "Invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "Invalid address for msg Signer")
	}

	// Perform TAO verification
	//
	// If the acknowledgement was already received, perform a no-op
	// Use a cached context to prevent accidental state changes
	cacheCtx, writeFn := sdkCtx.CacheContext()
	err = k.PacketServerKeeper.AcknowledgePacketV2(cacheCtx, msg.Packet, msg.MultiAcknowledgement, msg.ProofAcked, msg.ProofHeight)

	switch err {
	case nil:
		writeFn()
	case channeltypes.ErrNoOpMsg:
		// no-ops do not need event emission as they will be ignored
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-id", msg.Packet.SourceId)
		return &channeltypesv2.MsgAcknowledgementResponse{Result: channeltypes.NOOP}, nil
	default:
		sdkCtx.Logger().Error("acknowledgement failed", "source-id", msg.Packet.SourceId, "error", errorsmod.Wrap(err, "acknowledge packet verification failed"))
		return nil, errorsmod.Wrap(err, "acknowledge packet verification failed")
	}

	// construct mapping of app name to recvPacketResult
	// TODO: helper fn to do this.
	recvResults := make(map[string]channeltypes.RecvPacketResult)
	for _, r := range msg.MultiAcknowledgement.AcknowledgementResults {
		recvResults[r.AppName] = r.RecvPacketResult
	}

	// Perform application logic callback
	for _, pd := range msg.Packet.Data {
		cb := k.PortKeeper.AppRouter.Route(pd.SourcePort)
		err = cb.OnAcknowledgementPacketV2(ctx, msg.Packet, pd.Payload, recvResults[pd.DestinationPort], relayer)
		if err != nil {
			sdkCtx.Logger().Error("acknowledgement failed", "src_id", msg.Packet.SourceId, "src_port", pd.SourcePort, "dst_port", pd.DestinationPort, "error", errorsmod.Wrap(err, "acknowledge packet callback failed"))
			return nil, errorsmod.Wrap(err, "acknowledge packet callback failed")
		}
	}

	// defer telemetry.ReportAcknowledgePacket(msg.PacketV2)

	sdkCtx.Logger().Info("acknowledgement succeeded", "src_id", msg.Packet.SourceId, "result", channeltypes.SUCCESS.String())

	return &channeltypesv2.MsgAcknowledgementResponse{Result: channeltypes.SUCCESS}, nil
}
