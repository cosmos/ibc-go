package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/keeper"
)

type RedundantRelayDecorator struct {
	k *keeper.Keeper
}

func NewRedundantRelayDecorator(k *keeper.Keeper) RedundantRelayDecorator {
	return RedundantRelayDecorator{k: k}
}

// RedundantRelayDecorator returns an error if a multiMsg tx only contains packet messages (Recv, Ack, Timeout) and additional update messages
// and all packet messages are redundant. If the transaction is just a single UpdateClient message, or the multimsg transaction
// contains some other message type, then the antedecorator returns no error and continues processing to ensure these transactions
// are included. This will ensure that relayers do not waste fees on multiMsg transactions when another relayer has already submitted
// all packets, by rejecting the tx at the mempool layer.
func (rrd RedundantRelayDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// do not run redundancy check on DeliverTx or simulate
	if (ctx.IsCheckTx() || ctx.IsReCheckTx()) && !simulate {
		// keep track of total packet messages and number of redundancies across `RecvPacket`, `AcknowledgePacket`, and `TimeoutPacket/OnClose`
		redundancies := 0
		packetMsgs := 0
		for _, m := range tx.GetMsgs() {
			switch msg := m.(type) {
			case *channeltypes.MsgRecvPacket:
				var (
					response *channeltypes.MsgRecvPacketResponse
					err      error
				)
				// when we are in ReCheckTx mode, ctx.IsCheckTx() will also return true
				// therefore we must start the if statement on ctx.IsReCheckTx() to correctly
				// determine which mode we are in
				if ctx.IsReCheckTx() {
					response, err = rrd.recvPacketReCheckTx(ctx, msg)
				} else {
					response, err = rrd.recvPacketCheckTx(ctx, msg)
				}
				if err != nil {
					return ctx, err
				}

				if response.Result == channeltypes.NOOP {
					redundancies++
				}
				packetMsgs++

			case *channeltypes.MsgAcknowledgement:
				response, err := rrd.k.Acknowledgement(sdk.WrapSDKContext(ctx), msg)
				if err != nil {
					return ctx, err
				}
				if response.Result == channeltypes.NOOP {
					redundancies++
				}
				packetMsgs++

			case *channeltypes.MsgTimeout:
				response, err := rrd.k.Timeout(sdk.WrapSDKContext(ctx), msg)
				if err != nil {
					return ctx, err
				}
				if response.Result == channeltypes.NOOP {
					redundancies++
				}
				packetMsgs++

			case *channeltypes.MsgTimeoutOnClose:
				response, err := rrd.k.TimeoutOnClose(sdk.WrapSDKContext(ctx), msg)
				if err != nil {
					return ctx, err
				}
				if response.Result == channeltypes.NOOP {
					redundancies++
				}
				packetMsgs++

			case *clienttypes.MsgUpdateClient:
				_, err := rrd.k.UpdateClient(sdk.WrapSDKContext(ctx), msg)
				if err != nil {
					return ctx, err
				}

			default:
				// if the multiMsg tx has a msg that is not a packet msg or update msg, then we will not return error
				// regardless of if all packet messages are redundant. This ensures that non-packet messages get processed
				// even if they get batched with redundant packet messages.
				return next(ctx, tx, simulate)
			}
		}

		// only return error if all packet messages are redundant
		if redundancies == packetMsgs && packetMsgs > 0 {
			return ctx, channeltypes.ErrRedundantTx
		}
	}
	return next(ctx, tx, simulate)
}

// recvPacketCheckTx runs a subset of ibc recv packet logic to be used specifically within the RedundantRelayDecorator AnteHandler.
// It only performs core IBC receiving logic and skips any application logic.
func (rrd RedundantRelayDecorator) recvPacketCheckTx(ctx sdk.Context, msg *channeltypes.MsgRecvPacket) (*channeltypes.MsgRecvPacketResponse, error) {
	// grab channel capability
	_, capability, err := rrd.k.ChannelKeeper.LookupModuleByChannel(ctx, msg.Packet.DestinationPort, msg.Packet.DestinationChannel)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "could not retrieve module from port-id")
	}

	// If the packet was already received, perform a no-op
	// Use a cached context to prevent accidental state changes
	cacheCtx, writeFn := ctx.CacheContext()
	err = rrd.k.ChannelKeeper.RecvPacket(cacheCtx, capability, msg.Packet, msg.ProofCommitment, msg.ProofHeight)

	switch err {
	case nil:
		writeFn()
	case channeltypes.ErrNoOpMsg:
		return &channeltypes.MsgRecvPacketResponse{Result: channeltypes.NOOP}, nil
	default:
		return nil, sdkerrors.Wrap(err, "receive packet verification failed")
	}

	return &channeltypes.MsgRecvPacketResponse{Result: channeltypes.SUCCESS}, nil
}

// recvPacketReCheckTx runs a subset of ibc recv packet logic to be used specifically within the RedundantRelayDecorator AnteHandler.
// It only performs core IBC receiving logic and skips any application logic.
func (rrd RedundantRelayDecorator) recvPacketReCheckTx(ctx sdk.Context, msg *channeltypes.MsgRecvPacket) (*channeltypes.MsgRecvPacketResponse, error) {
	// If the packet was already received, perform a no-op
	// Use a cached context to prevent accidental state changes
	cacheCtx, writeFn := ctx.CacheContext()
	err := rrd.k.ChannelKeeper.RecvPacketReCheckTx(cacheCtx, msg.Packet)

	switch err {
	case nil:
		writeFn()
	case channeltypes.ErrNoOpMsg:
		return &channeltypes.MsgRecvPacketResponse{Result: channeltypes.NOOP}, nil
	default:
		return nil, sdkerrors.Wrap(err, "receive packet verification failed")
	}

	return &channeltypes.MsgRecvPacketResponse{Result: channeltypes.SUCCESS}, nil
}
