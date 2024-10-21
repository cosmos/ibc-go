package keeper

import (
	"context"
	"slices"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypesv1 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	internalerrors "github.com/cosmos/ibc-go/v9/modules/core/internal/errors"
	telemetryv2 "github.com/cosmos/ibc-go/v9/modules/core/internal/v2/telemetry"
)

var _ channeltypesv2.MsgServer = &Keeper{}

// SendPacket implements the PacketMsgServer SendPacket method.
func (k *Keeper) SendPacket(ctx context.Context, msg *channeltypesv2.MsgSendPacket) (*channeltypesv2.MsgSendPacketResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sequence, destChannel, err := k.sendPacket(ctx, msg.SourceChannel, msg.TimeoutTimestamp, msg.Payload)
	if err != nil {
		sdkCtx.Logger().Error("send packet failed", "source-channel", msg.SourceChannel, "error", errorsmod.Wrap(err, "send packet failed"))
		return nil, errorsmod.Wrapf(err, "send packet failed for source id: %s", msg.SourceChannel)
	}

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		sdkCtx.Logger().Error("send packet failed", "error", errorsmod.Wrap(err, "invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "invalid address for msg Signer")
	}

	for _, pd := range msg.Payload {
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

	recvResults := make(map[string]channeltypesv2.RecvPacketResult)
	for _, r := range msg.Acknowledgement.AcknowledgementResults {
		recvResults[r.AppName] = r.RecvPacketResult
	}

	for _, pd := range msg.Packet.Data {
		cbs := k.Router.Route(pd.SourcePort)
		err := cbs.OnAcknowledgementPacket(ctx, msg.Packet.SourceChannel, msg.Packet.DestinationChannel, pd, recvResults[pd.DestinationPort].Acknowledgement, relayer)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed OnAcknowledgementPacket for source port %s, source channel %s, destination channel %s", pd.SourcePort, msg.Packet.SourceChannel, msg.Packet.DestinationChannel)
		}
	}

	return &channeltypesv2.MsgAcknowledgementResponse{Result: channeltypesv1.SUCCESS}, nil
}

// RecvPacket implements the PacketMsgServer RecvPacket method.
func (k *Keeper) RecvPacket(ctx context.Context, msg *channeltypesv2.MsgRecvPacket) (*channeltypesv2.MsgRecvPacketResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		sdkCtx.Logger().Error("receive packet failed", "error", errorsmod.Wrap(err, "invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "invalid address for msg Signer")
	}

	// Perform TAO verification
	//
	// If the packet was already received, perform a no-op
	// Use a cached context to prevent accidental state changes
	cacheCtx, writeFn := sdkCtx.CacheContext()
	err = k.recvPacket(cacheCtx, msg.Packet, msg.ProofCommitment, msg.ProofHeight)

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
			sdkCtx.EventManager().EmitEvents(internalerrors.ConvertToErrorEvents(cacheCtx.EventManager().Events()))
		}

		ack.AcknowledgementResults = append(ack.AcknowledgementResults, channeltypesv2.AcknowledgementResult{
			AppName:          pd.DestinationPort,
			RecvPacketResult: res,
		})
	}

	// note this should never happen as the packet data would have had to be empty.
	if len(ack.AcknowledgementResults) == 0 {
		sdkCtx.Logger().Error("receive packet failed", "source-channel", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "invalid acknowledgement results"))
		return &channeltypesv2.MsgRecvPacketResponse{Result: channeltypesv1.FAILURE}, errorsmod.Wrapf(err, "receive packet failed source-channel %s invalid acknowledgement results", msg.Packet.SourceChannel)
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

	signer, err := sdk.AccAddressFromBech32(timeout.Signer)
	if err != nil {
		sdkCtx.Logger().Error("timeout packet failed", "error", errorsmod.Wrap(err, "invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "invalid address for msg Signer")
	}

	cacheCtx, writeFn := sdkCtx.CacheContext()
	if err := k.timeoutPacket(cacheCtx, timeout.Packet, timeout.ProofUnreceived, timeout.ProofHeight); err != nil {
		sdkCtx.Logger().Error("Timeout packet failed", "source-channel", timeout.Packet.SourceChannel, "destination-channel", timeout.Packet.DestinationChannel, "error", errorsmod.Wrap(err, "timeout packet failed"))
		return nil, errorsmod.Wrapf(err, "timeout packet failed for source id: %s and destination id: %s", timeout.Packet.SourceChannel, timeout.Packet.DestinationChannel)
	}

	switch err {
	case nil:
		writeFn()
	case channeltypesv1.ErrNoOpMsg:
		// no-ops do not need event emission as they will be ignored
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-channel", timeout.Packet.SourceChannel)
		return &channeltypesv2.MsgTimeoutResponse{Result: channeltypesv1.NOOP}, nil
	default:
		sdkCtx.Logger().Error("timeout failed", "source-channel", timeout.Packet.SourceChannel, "error", errorsmod.Wrap(err, "timeout packet verification failed"))
		return nil, errorsmod.Wrap(err, "timeout packet verification failed")
	}

	for _, pd := range timeout.Packet.Data {
		cbs := k.Router.Route(pd.SourcePort)
		err := cbs.OnTimeoutPacket(ctx, timeout.Packet.SourceChannel, timeout.Packet.DestinationChannel, pd, signer)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed OnTimeoutPacket for source port %s, source channel %s, destination channel %s", pd.SourcePort, timeout.Packet.SourceChannel, timeout.Packet.DestinationChannel)
		}
	}

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
