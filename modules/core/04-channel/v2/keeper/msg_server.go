package keeper

import (
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	internalerrors "github.com/cosmos/ibc-go/v9/modules/core/internal/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/internal/v2/telemetry"
)

var _ types.MsgServer = &Keeper{}

// CreateChannel defines a rpc handler method for MsgCreateChannel.
func (k *Keeper) CreateChannel(goCtx context.Context, msg *types.MsgCreateChannel) (*types.MsgCreateChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	channelID := k.channelKeeperV1.GenerateChannelIdentifier(ctx)

	// Initialize channel with empty counterparty channel identifier.
	channel := types.NewChannel(msg.ClientId, "", msg.MerklePathPrefix)
	k.SetChannel(ctx, channelID, channel)
	k.SetCreator(ctx, channelID, msg.Signer)
	k.SetNextSequenceSend(ctx, channelID, 1)

	k.emitCreateChannelEvent(goCtx, channelID, msg.ClientId)

	return &types.MsgCreateChannelResponse{ChannelId: channelID}, nil
}

// RegisterCounterparty defines a rpc handler method for MsgRegisterCounterparty.
func (k *Keeper) RegisterCounterparty(goCtx context.Context, msg *types.MsgRegisterCounterparty) (*types.MsgRegisterCounterpartyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	channel, ok := k.GetChannel(ctx, msg.ChannelId)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrChannelNotFound, "channel must exist for channel id %s", msg.ChannelId)
	}

	creator, found := k.GetCreator(ctx, msg.ChannelId)
	if !found {
		return nil, errorsmod.Wrap(ibcerrors.ErrUnauthorized, "channel creator must be set")
	}

	if creator != msg.Signer {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "channel creator (%s) must match signer (%s)", creator, msg.Signer)
	}

	channel.CounterpartyChannelId = msg.CounterpartyChannelId
	k.SetChannel(ctx, msg.ChannelId, channel)
	// Delete client creator from state as it is not needed after this point.
	k.DeleteCreator(ctx, msg.ChannelId)

	k.emitRegisterCounterpartyEvent(goCtx, msg.ChannelId, channel)

	return &types.MsgRegisterCounterpartyResponse{}, nil
}

// SendPacket implements the PacketMsgServer SendPacket method.
func (k *Keeper) SendPacket(ctx context.Context, msg *types.MsgSendPacket) (*types.MsgSendPacketResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Note, the validate basic function in sendPacket does the timeoutTimestamp != 0 check and other stateless checks on the packet.
	// timeoutTimestamp must be greater than current block time
	timeout := time.Unix(int64(msg.TimeoutTimestamp), 0)
	if timeout.Before(sdkCtx.BlockTime()) {
		return nil, errorsmod.Wrap(types.ErrTimeoutElapsed, "timeout is less than the current block timestamp")
	}

	// timeoutTimestamp must be less than current block time + MaxTimeoutDelta
	if timeout.After(sdkCtx.BlockTime().Add(types.MaxTimeoutDelta)) {
		return nil, errorsmod.Wrap(types.ErrInvalidTimeout, "timeout exceeds the maximum expected value")
	}

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		sdkCtx.Logger().Error("send packet failed", "error", errorsmod.Wrap(err, "invalid address for msg Signer"))
		return nil, errorsmod.Wrap(err, "invalid address for msg Signer")
	}

	sequence, destChannel, err := k.sendPacket(ctx, msg.SourceChannel, msg.TimeoutTimestamp, msg.Payloads)
	if err != nil {
		sdkCtx.Logger().Error("send packet failed", "source-channel", msg.SourceChannel, "error", errorsmod.Wrap(err, "send packet failed"))
		return nil, errorsmod.Wrapf(err, "send packet failed for source id: %s", msg.SourceChannel)
	}

	for _, pd := range msg.Payloads {
		cbs := k.Router.Route(pd.SourcePort)
		err := cbs.OnSendPacket(ctx, msg.SourceChannel, destChannel, sequence, pd, signer)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgSendPacketResponse{Sequence: sequence}, nil
}

// RecvPacket implements the PacketMsgServer RecvPacket method.
func (k *Keeper) RecvPacket(ctx context.Context, msg *types.MsgRecvPacket) (*types.MsgRecvPacketResponse, error) {
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
	case types.ErrNoOpMsg:
		// no-ops do not need event emission as they will be ignored
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-channel", msg.Packet.SourceChannel)
		return &types.MsgRecvPacketResponse{Result: types.NOOP}, nil
	default:
		sdkCtx.Logger().Error("receive packet failed", "source-channel", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "receive packet verification failed"))
		return nil, errorsmod.Wrap(err, "receive packet verification failed")
	}

	// build up the recv results for each application callback.
	ack := types.Acknowledgement{
		AppAcknowledgements: [][]byte{},
	}

	var isAsync bool
	for _, pd := range msg.Packet.Payloads {
		// Cache context so that we may discard state changes from callback if the acknowledgement is unsuccessful.
		cacheCtx, writeFn = sdkCtx.CacheContext()
		cb := k.Router.Route(pd.DestinationPort)
		res := cb.OnRecvPacket(cacheCtx, msg.Packet.SourceChannel, msg.Packet.DestinationChannel, msg.Packet.Sequence, pd, signer)

		if res.Status != types.PacketStatus_Failure {
			// write application state changes for asynchronous and successful acknowledgements
			writeFn()
		} else {
			// Modify events in cached context to reflect unsuccessful acknowledgement
			sdkCtx.EventManager().EmitEvents(internalerrors.ConvertToErrorEvents(cacheCtx.EventManager().Events()))
		}

		if res.Status == types.PacketStatus_Async {
			// Set packet acknowledgement to async if any of the acknowledgements are async.
			isAsync = true
			// Return error if there is more than 1 payload
			// TODO: Handle case where there are multiple payloads
			if len(msg.Packet.Payloads) > 1 {
				return nil, errorsmod.Wrapf(types.ErrInvalidPacket, "packet with multiple payloads cannot have async acknowledgement")
			}
		}

		// append app acknowledgement to the overall acknowledgement
		ack.AppAcknowledgements = append(ack.AppAcknowledgements, res.Acknowledgement)
	}

	if len(ack.AppAcknowledgements) != len(msg.Packet.Payloads) {
		return nil, errorsmod.Wrapf(types.ErrInvalidAcknowledgement, "length of app acknowledgement %d does not match length of app payload %d", len(ack.AppAcknowledgements), len(msg.Packet.Payloads))
	}

	// note this should never happen as the payload would have had to be empty.
	if len(ack.AppAcknowledgements) == 0 {
		sdkCtx.Logger().Error("receive packet failed", "source-channel", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "invalid acknowledgement results"))
		return &types.MsgRecvPacketResponse{Result: types.FAILURE}, errorsmod.Wrapf(err, "receive packet failed source-channel %s invalid acknowledgement results", msg.Packet.SourceChannel)
	}

	if !isAsync {
		// Validate ack before forwarding to WriteAcknowledgement.
		if err := ack.Validate(); err != nil {
			return nil, err
		}
		// Set packet acknowledgement only if the acknowledgement is not async.
		// NOTE: IBC applications modules may call the WriteAcknowledgement asynchronously if the
		// acknowledgement is async.
		if err := k.WriteAcknowledgement(ctx, msg.Packet, ack); err != nil {
			return nil, err
		}
	}

	// TODO: store the packet for async applications to access if required.
	defer telemetry.ReportRecvPacket(msg.Packet)

	sdkCtx.Logger().Info("receive packet callback succeeded", "source-channel", msg.Packet.SourceChannel, "dest-channel", msg.Packet.DestinationChannel, "result", types.SUCCESS.String())
	return &types.MsgRecvPacketResponse{Result: types.SUCCESS}, nil
}

// Acknowledgement defines an rpc handler method for MsgAcknowledgement.
func (k *Keeper) Acknowledgement(ctx context.Context, msg *types.MsgAcknowledgement) (*types.MsgAcknowledgementResponse, error) {
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
	case types.ErrNoOpMsg:
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-channel", msg.Packet.SourceChannel)
		return &types.MsgAcknowledgementResponse{Result: types.NOOP}, nil
	default:
		sdkCtx.Logger().Error("acknowledgement failed", "source-channel", msg.Packet.SourceChannel, "error", errorsmod.Wrap(err, "acknowledge packet verification failed"))
		return nil, errorsmod.Wrap(err, "acknowledge packet verification failed")
	}

	for i, pd := range msg.Packet.Payloads {
		cbs := k.Router.Route(pd.SourcePort)
		ack := msg.Acknowledgement.AppAcknowledgements[i]
		err := cbs.OnAcknowledgementPacket(ctx, msg.Packet.SourceChannel, msg.Packet.DestinationChannel, msg.Packet.Sequence, ack, pd, relayer)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed OnAcknowledgementPacket for source port %s, source channel %s, destination channel %s", pd.SourcePort, msg.Packet.SourceChannel, msg.Packet.DestinationChannel)
		}
	}

	return &types.MsgAcknowledgementResponse{Result: types.SUCCESS}, nil
}

// Timeout implements the PacketMsgServer Timeout method.
func (k *Keeper) Timeout(ctx context.Context, timeout *types.MsgTimeout) (*types.MsgTimeoutResponse, error) {
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
	case types.ErrNoOpMsg:
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-channel", timeout.Packet.SourceChannel)
		return &types.MsgTimeoutResponse{Result: types.NOOP}, nil
	default:
		sdkCtx.Logger().Error("timeout failed", "source-channel", timeout.Packet.SourceChannel, "error", errorsmod.Wrap(err, "timeout packet verification failed"))
		return nil, errorsmod.Wrap(err, "timeout packet verification failed")
	}

	for _, pd := range timeout.Packet.Payloads {
		cbs := k.Router.Route(pd.SourcePort)
		err := cbs.OnTimeoutPacket(ctx, timeout.Packet.SourceChannel, timeout.Packet.DestinationChannel, timeout.Packet.Sequence, pd, signer)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed OnTimeoutPacket for source port %s, source channel %s, destination channel %s", pd.SourcePort, timeout.Packet.SourceChannel, timeout.Packet.DestinationChannel)
		}
	}

	return &types.MsgTimeoutResponse{Result: types.SUCCESS}, nil
}
