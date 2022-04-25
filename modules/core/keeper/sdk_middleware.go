package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/tx"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

var _ tx.Handler = ibcTxHandler{}

type ibcTxHandler struct {
	k    *Keeper
	next tx.Handler
}

// IBCTxMiddleware implements ibc tx handling middleware
func IBCTxMiddleware(IBCKeeper *Keeper) tx.Middleware {
	return func(txh tx.Handler) tx.Handler {
		return ibcTxHandler{
			k:    IBCKeeper,
			next: txh,
		}
	}
}

// perform redundancy checks on IBC relays. If a transaction contains a relay message (`RecvPacket`, `AcknowledgePacket`, `TimeoutPacket/OnClose`)
// and all the relay messages are redundant, it will be dropped from the mempool. If any non relay message is contained, with the exception of `UpdateClient`,
// it will be processed as normal. A transaction with only `UpdateClient` message will be processed as normal.
func (itxh ibcTxHandler) checkRedundancy(ctx context.Context, req tx.Request) error {
	// keep track of total packet messages and number of redundancies across `RecvPacket`, `AcknowledgePacket`, and `TimeoutPacket/OnClose`
	redundancies := 0
	packetMsgs := 0

	for _, m := range req.Tx.GetMsgs() {
		switch msg := m.(type) {
		case *channeltypes.MsgRecvPacket:
			response, err := itxh.k.RecvPacket(ctx, msg)
			if err != nil {
				return err
			}

			if response.Result == channeltypes.NOOP {
				redundancies += 1
			}
			packetMsgs += 1

		case *channeltypes.MsgAcknowledgement:
			response, err := itxh.k.Acknowledgement(ctx, msg)
			if err != nil {
				return err
			}

			if response.Result == channeltypes.NOOP {
				redundancies += 1
			}
			packetMsgs += 1

		case *channeltypes.MsgTimeout:
			response, err := itxh.k.Timeout(ctx, msg)
			if err != nil {
				return err
			}

			if response.Result == channeltypes.NOOP {
				redundancies += 1
			}
			packetMsgs += 1

		case *channeltypes.MsgTimeoutOnClose:
			response, err := itxh.k.TimeoutOnClose(ctx, msg)
			if err != nil {
				return err

			}

			if response.Result == channeltypes.NOOP {
				redundancies += 1
			}
			packetMsgs += 1

		case *clienttypes.MsgUpdateClient:
			if _, err := itxh.k.UpdateClient(ctx, msg); err != nil {
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

	return nil
}

// CheckTx implements tx.Handler.CheckTx. Run redundancy checks on CheckTx to filter any redundant
// relays in the mempool.
func (itxh ibcTxHandler) CheckTx(ctx context.Context, req tx.Request, checkReq tx.RequestCheckTx) (tx.Response, tx.ResponseCheckTx, error) {
	if err := itxh.checkRedundancy(ctx, req); err != nil {
		return tx.Response{}, tx.ResponseCheckTx{}, err
	}

	return itxh.next.CheckTx(ctx, req, checkReq)
}

// DeliverTx implements tx.Handler.DeliverTx. Redundancy checks are not run on DeliverTx since
// the transaction has already been included in a block.
func (itxh ibcTxHandler) DeliverTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	return itxh.next.DeliverTx(ctx, req)
}

// SimulateTx implements tx.Handler.SimulateTx. Run redundancy checks on SimulateTx to filter any redundant
// relays in the mempool.
func (itxh ibcTxHandler) SimulateTx(ctx context.Context, req tx.Request) (tx.Response, error) {
	if err := itxh.checkRedundancy(ctx, req); err != nil {
		return tx.Response{}, err
	}

	return itxh.next.SimulateTx(ctx, req)
}
