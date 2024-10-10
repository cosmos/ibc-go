package keeper

import (
	"context"
	"slices"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	telemetryv2 "github.com/cosmos/ibc-go/v9/modules/core/internal/v2/telemetry"
	coretypes "github.com/cosmos/ibc-go/v9/modules/core/types"
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

	// Perform TAO verification
	//
	// If the packet was already received, perform a no-op
	// Use a cached context to prevent accidental state changes
	cacheCtx, writeFn := sdkCtx.CacheContext()
	err := k.recvPacket(cacheCtx, msg.Packet, msg.ProofCommitment, msg.ProofHeight)
	if err != nil {
		sdkCtx.Logger().Error("receive packet failed", "source-channel", msg.Packet.SourceChannel, "dest-channel", msg.Packet.DestinationChannel, "error", errorsmod.Wrap(err, "send packet failed"))
		return nil, errorsmod.Wrapf(err, "receive packet failed for source id: %s and destination id: %s", msg.Packet.SourceChannel, msg.Packet.DestinationChannel)
	}

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		sdkCtx.Logger().Error("receive packet failed", "error", errorsmod.Wrap(err, "invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "invalid address for msg Signer")
	}

	switch err {
	case nil:
		writeFn()
	case channeltypesv1.ErrNoOpMsg:
		// no-ops do not need event emission as they will be ignored
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-channel", msg.Packet.SourceChannel)
		return &channeltypesv2.MsgRecvPacketResponse{Result: channeltypesv1.NOOP}, nil
	default:
		sdkCtx.Logger().Error("receive packet failed", "source-channel", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "receive packet verification failed"))
		return nil, errorsmod.Wrap(err, "receive packet verification failed")
	}

	// build up the recv results for each application callback.
	ack := channeltypesv2.Acknowledgement{
		AcknowledgementResults: []channeltypesv2.AcknowledgementResult{},
	}

	for _, pd := range msg.Packet.Data {
		// Cache context so that we may discard state changes from callback if the acknowledgement is unsuccessful.
		cacheCtx, writeFn = sdkCtx.CacheContext()
		cb := k.Router.Route(pd.DestinationPort)
		res := cb.OnRecvPacket(cacheCtx, msg.Packet.SourceChannel, msg.Packet.DestinationChannel, pd, signer)

		if res.Status != channeltypesv2.PacketStatus_Failure {
			// write application state changes for asynchronous and successful acknowledgements
			writeFn()
		} else {
			// Modify events in cached context to reflect unsuccessful acknowledgement
			sdkCtx.EventManager().EmitEvents(convertToErrorEvents(cacheCtx.EventManager().Events()))
		}

		ack.AcknowledgementResults = append(ack.AcknowledgementResults, channeltypesv2.AcknowledgementResult{
			AppName:          pd.DestinationPort,
			RecvPacketResult: res,
		})
	}

	// note this should never happen as the packet data would have had to be empty.
	if len(ack.AcknowledgementResults) == 0 {
		sdkCtx.Logger().Error("receive packet failed", "source-channel", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "invalid acknowledgement results"))
		return &channeltypesv2.MsgRecvPacketResponse{Result: channeltypesv1.FAILURE}, nil
	}

	// NOTE: TBD how we will handle async acknowledgements with more than one packet data.
	isAsync := slices.ContainsFunc(ack.AcknowledgementResults, func(ackResult channeltypesv2.AcknowledgementResult) bool {
		return ackResult.RecvPacketResult.Status == channeltypesv2.PacketStatus_Async
	})

	if !isAsync {
		// Set packet acknowledgement only if the acknowledgement is not async.
		// NOTE: IBC applications modules may call the WriteAcknowledgement asynchronously if the
		// acknowledgement is async.
		if err := k.WriteAcknowledgement(ctx, msg.Packet, ack); err != nil {
			return nil, err
		}
	}

	defer telemetryv2.ReportRecvPacket(msg.Packet)

	sdkCtx.Logger().Info("receive packet callback succeeded", "source-channel", msg.Packet.SourceChannel, "dest-channel", msg.Packet.DestinationChannel, "result", channeltypesv1.SUCCESS.String())
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

// convertToErrorEvents converts all events to error events by appending the
// error attribute prefix to each event's attribute key.
// TODO: duplication, create issue for this
func convertToErrorEvents(events sdk.Events) sdk.Events {
	if events == nil {
		return nil
	}

	newEvents := make(sdk.Events, len(events))
	for i, event := range events {
		newAttributes := make([]sdk.Attribute, len(event.Attributes))
		for j, attribute := range event.Attributes {
			newAttributes[j] = sdk.NewAttribute(coretypes.ErrorAttributeKeyPrefix+attribute.Key, attribute.Value)
		}

		newEvents[i] = sdk.NewEvent(coretypes.ErrorAttributeKeyPrefix+event.Type, newAttributes...)
	}

	return newEvents
}
