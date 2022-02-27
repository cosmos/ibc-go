package middleware

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channelkeeper "github.com/cosmos/ibc-go/v3/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

var _ tx.Handler = ibcTxHandler{}

type ibcTxHandler struct {
	k    channelkeeper.Keeper
	next tx.Handler
}

// IbcTxMiddleware implements ibc tx handling middleware
func IbcTxMiddleware(channelkeeper channelkeeper.Keeper) tx.Middleware {
	return func(txh tx.Handler) tx.Handler {
		return ibcTxHandler{
			k:    channelkeeper,
			next: txh,
		}
	}
}

func (itxh ibcTxHandler) checkRedundancy(ctx context.Context, req tx.Request, simulate bool) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// do not run redundancy check on DeliverTx or simulate
	if (sdkCtx.IsCheckTx() || sdkCtx.IsReCheckTx()) && !simulate {
		redundancies := 0
		packetMsgs := 0
		for _, m := range req.Tx.GetMsgs() {
			switch msg := m.(type) {
			case *channeltypes.MsgRecvPacket:
				if _, found := itxh.k.GetPacketReceipt(sdkCtx, msg.Packet.GetDestPort(), msg.Packet.GetDestChannel(), msg.Packet.GetSequence()); found {
					redundancies++
				}
				packetMsgs++

			case *channeltypes.MsgAcknowledgement:
				if commitment := itxh.k.GetPacketCommitment(sdkCtx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies++
				}
				packetMsgs++

			case *channeltypes.MsgTimeout:
				if commitment := itxh.k.GetPacketCommitment(sdkCtx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies++
				}
				packetMsgs++

			case *channeltypes.MsgTimeoutOnClose:
				if commitment := itxh.k.GetPacketCommitment(sdkCtx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies++
				}
				packetMsgs++

			case *clienttypes.MsgUpdateClient:
				// do nothing here, as we want to avoid updating clients if it is batched with only redundant messages

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
