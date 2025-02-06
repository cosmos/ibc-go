package keeper

import (
	"bytes"
	"context"
	"time"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	internalerrors "github.com/cosmos/ibc-go/v9/modules/core/internal/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/internal/v2/telemetry"
)

var _ types.MsgServer = &Keeper{}

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

	sequence, destChannel, err := k.sendPacket(ctx, msg.SourceClient, msg.TimeoutTimestamp, msg.Payloads)
	if err != nil {
		sdkCtx.Logger().Error("send packet failed", "source-client", msg.SourceClient, "error", errorsmod.Wrap(err, "send packet failed"))
		return nil, errorsmod.Wrapf(err, "send packet failed for source id: %s", msg.SourceClient)
	}

	for _, pd := range msg.Payloads {
		cbs := k.Router.Route(pd.SourcePort)
		err := cbs.OnSendPacket(ctx, msg.SourceClient, destChannel, sequence, pd, signer)
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
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-client", msg.Packet.SourceClient)
		return &types.MsgRecvPacketResponse{Result: types.NOOP}, nil
	default:
		sdkCtx.Logger().Error("receive packet failed", "source-client", msg.Packet.SourceClient, "error", errorsmod.Wrap(err, "receive packet verification failed"))
		return nil, errorsmod.Wrap(err, "receive packet verification failed")
	}

	// build up the recv results for each application callback.
	ack := types.Acknowledgement{
		AppAcknowledgements: [][]byte{},
	}

	var isAsync bool
	isSuccess := true
	for _, pd := range msg.Packet.Payloads {
		// Cache context so that we may discard state changes from callback if the acknowledgement is unsuccessful.
		cacheCtx, writeFn = sdkCtx.CacheContext()
		cb := k.Router.Route(pd.DestinationPort)
		res := cb.OnRecvPacket(cacheCtx, msg.Packet.SourceClient, msg.Packet.DestinationClient, msg.Packet.Sequence, pd, signer)

		if res.Status != types.PacketStatus_Failure {
			// successful app acknowledgement cannot equal sentinel error acknowledgement
			if bytes.Equal(res.GetAcknowledgement(), types.ErrorAcknowledgement[:]) {
				return nil, errorsmod.Wrapf(types.ErrInvalidAcknowledgement, "application acknowledgement cannot be sentinel error acknowledgement")
			}
			// write application state changes for asynchronous and successful acknowledgements
			writeFn()
			// append app acknowledgement to the overall acknowledgement
			ack.AppAcknowledgements = append(ack.AppAcknowledgements, res.Acknowledgement)
		} else {
			isSuccess = false
			// construct acknowledgement with single app acknowledgement that is the sentinel error acknowledgement
			ack = types.Acknowledgement{
				AppAcknowledgements: [][]byte{types.ErrorAcknowledgement[:]},
			}
			// Modify events in cached context to reflect unsuccessful acknowledgement
			sdkCtx.EventManager().EmitEvents(internalerrors.ConvertToErrorEvents(cacheCtx.EventManager().Events()))
			break
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
	}

	if !isAsync {
		// If the application callback was successful, the acknowledgement must have the same number of app acknowledgements as the packet payloads.
		if isSuccess {
			if len(ack.AppAcknowledgements) != len(msg.Packet.Payloads) {
				return nil, errorsmod.Wrapf(types.ErrInvalidAcknowledgement, "length of app acknowledgement %d does not match length of app payload %d", len(ack.AppAcknowledgements), len(msg.Packet.Payloads))
			}
		}

		// Validate ack before forwarding to WriteAcknowledgement.
		if err := ack.Validate(); err != nil {
			return nil, err
		}
		// Set packet acknowledgement only if the acknowledgement is not async.
		// NOTE: IBC applications modules may call the WriteAcknowledgement asynchronously if the
		// acknowledgement is async.
		if err := k.writeAcknowledgement(ctx, msg.Packet, ack); err != nil {
			return nil, err
		}
	} else {
		// store the packet temporarily until the application returns an acknowledgement
		k.SetAsyncPacket(ctx, msg.Packet.DestinationClient, msg.Packet.Sequence, msg.Packet)
	}

	// TODO: store the packet for async applications to access if required.
	defer telemetry.ReportRecvPacket(msg.Packet)

	sdkCtx.Logger().Info("receive packet callback succeeded", "source-client", msg.Packet.SourceClient, "dest-client", msg.Packet.DestinationClient, "result", types.SUCCESS.String())
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
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-client", msg.Packet.SourceClient)
		return &types.MsgAcknowledgementResponse{Result: types.NOOP}, nil
	default:
		sdkCtx.Logger().Error("acknowledgement failed", "source-client", msg.Packet.SourceClient, "error", errorsmod.Wrap(err, "acknowledge packet verification failed"))
		return nil, errorsmod.Wrap(err, "acknowledge packet verification failed")
	}

	recvSuccess := !bytes.Equal(msg.Acknowledgement.AppAcknowledgements[0], types.ErrorAcknowledgement[:])
	for i, pd := range msg.Packet.Payloads {
		cbs := k.Router.Route(pd.SourcePort)
		var ack []byte
		// if recv was successful, each payload should have its own acknowledgement so we send each individual acknowledgment to the application
		// otherwise, the acknowledgement only contains the sentinel error acknowledgement which we send to the application. The application is responsible
		// for knowing that this is an error acknowledgement and executing the appropriate logic.
		if recvSuccess {
			ack = msg.Acknowledgement.AppAcknowledgements[i]
		} else {
			ack = types.ErrorAcknowledgement[:]
		}
		err := cbs.OnAcknowledgementPacket(ctx, msg.Packet.SourceClient, msg.Packet.DestinationClient, msg.Packet.Sequence, ack, pd, relayer)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed OnAcknowledgementPacket for source port %s, source client %s, destination client %s", pd.SourcePort, msg.Packet.SourceClient, msg.Packet.DestinationClient)
		}
	}

	defer telemetry.ReportAcknowledgePacket(msg.Packet)

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
	err = k.timeoutPacket(cacheCtx, timeout.Packet, timeout.ProofUnreceived, timeout.ProofHeight)

	switch err {
	case nil:
		writeFn()
	case types.ErrNoOpMsg:
		sdkCtx.Logger().Debug("no-op on redundant relay", "source-client", timeout.Packet.SourceClient)
		return &types.MsgTimeoutResponse{Result: types.NOOP}, nil
	default:
		sdkCtx.Logger().Error("timeout failed", "source-client", timeout.Packet.SourceClient, "error", errorsmod.Wrap(err, "timeout packet verification failed"))
		return nil, errorsmod.Wrap(err, "timeout packet verification failed")
	}

	for _, pd := range timeout.Packet.Payloads {
		cbs := k.Router.Route(pd.SourcePort)
		err := cbs.OnTimeoutPacket(ctx, timeout.Packet.SourceClient, timeout.Packet.DestinationClient, timeout.Packet.Sequence, pd, signer)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed OnTimeoutPacket for source port %s, source client %s, destination client %s", pd.SourcePort, timeout.Packet.SourceClient, timeout.Packet.DestinationClient)
		}
	}

	defer telemetry.ReportTimeoutPacket(timeout.Packet)

	return &types.MsgTimeoutResponse{Result: types.SUCCESS}, nil
}
