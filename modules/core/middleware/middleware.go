package middleware

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v3/modules/core/keeper"
)

var _ tx.Handler = ibcTxHandler{}

type ibcTxHandler struct {
	k    *keeper.Keeper
	next tx.Handler
}

// IBCTxMiddleware implements ibc tx handling middleware
func IBCTxMiddleware(IBCKeeper *keeper.Keeper) tx.Middleware {
	return func(txh tx.Handler) tx.Handler {
		return ibcTxHandler{
			k:    IBCKeeper,
			next: txh,
		}
	}
}

func (itxh ibcTxHandler) checkRedundancy(ctx context.Context, req tx.Request, simulate bool) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// do not run redundancy check on DeliverTx or simulate
	if (sdkCtx.IsCheckTx() || sdkCtx.IsReCheckTx()) && !simulate {
		// keep track of total packet messages and number of redundancies across `RecvPacket`, `AcknowledgePacket`, and `TimeoutPacket/OnClose`
		redundancies := 0
		packetMsgs := 0
		for _, m := range req.Tx.GetMsgs() {
			switch msg := m.(type) {
			case *channeltypes.MsgRecvPacket:
				response, err := itxh.k.RecvPacket(sdkCtx, msg)
				if err != nil {
					return err
				}
				if response.Result == channeltypes.NOOP {
					redundancies += 1
				}
				packetMsgs += 1

			case *channeltypes.MsgAcknowledgement:
				response, err := itxh.k.Acknowledgement(sdkCtx, msg)
				if err != nil {
					return err
				}
				if response.Result == channeltypes.NOOP {
					redundancies += 1
				}
				packetMsgs += 1

			case *channeltypes.MsgTimeout:
				response, err := itxh.k.Timeout(sdkCtx, msg)
				if err != nil {
					return err
				}
				if response.Result == channeltypes.NOOP {
					redundancies += 1
				}
				packetMsgs += 1

			case *channeltypes.MsgTimeoutOnClose:
				response, err := itxh.k.TimeoutOnClose(sdkCtx, msg)
				if err != nil {
					return err
				}
				if response.Result == channeltypes.NOOP {
					redundancies += 1
				}
				packetMsgs += 1

			case *clienttypes.MsgUpdateClient:
				_, err := itxh.k.UpdateClient(sdkCtx, msg)
				if err != nil {
					return err
				}

			default:
				// if the multiMsg tx has a msg that is not a packet msg or update msg, then we will not return error
				// regardless of if all packet messages are redundant. This ensures that non-packet messages get processed
				// even if they get batched with redundant packet messages.
				return nil
			}
		}

		// only return error if all packet messages are redundant
		if redundancies == packetMsgs && packetMsgs > 0 {
			return channeltypes.ErrRedundantTx
		}
	}
	return nil
}

// CheckTx implements tx.Handler.CheckTx.
func (itxh ibcTxHandler) CheckTx(ctx context.Context, req tx.Request, checkReq tx.RequestCheckTx) (tx.Response, tx.ResponseCheckTx, error) {
	err := itxh.checkRedundancy(ctx, req, false)
	if err != nil {
		return tx.Response{}, tx.ResponseCheckTx{}, err
	}

	return itxh.next.CheckTx(ctx, req, checkReq)
}

// DeliverTx implements tx.Handler.DeliverTx.
func (itxh ibcTxHandler) DeliverTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	err := itxh.checkRedundancy(ctx, req, false)
	if err != nil {
		return tx.Response{}, err
	}
	return itxh.next.DeliverTx(ctx, req)
}

// SimulateTx implements tx.Handler.SimulateTx.
func (itxh ibcTxHandler) SimulateTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	err := itxh.checkRedundancy(ctx, req, true)
	if err != nil {
		return tx.Response{}, err
	}
	return itxh.next.SimulateTx(ctx, req)
}
