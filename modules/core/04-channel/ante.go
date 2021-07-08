package channel

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/core/04-channel/keeper"
	"github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

type ChannelAnteDecorator struct {
	k keeper.Keeper
}

func NewChannelAnteDecorator(k keeper.Keeper) ChannelAnteDecorator {
	return ChannelAnteDecorator{k: k}
}

func (cad ChannelAnteDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// do not run redundancy check on DeliverTx or simulate
	if (ctx.IsCheckTx() || ctx.IsReCheckTx()) && !simulate {
		// keep track of total packet messages and number of redundancies across `RecvPacket`, `AcknowledgePacket`, and `TimeoutPacket/OnClose`
		msgs := 0
		redundancies := 0
		for _, m := range tx.GetMsgs() {
			if msg, ok := m.(*types.MsgRecvPacket); ok {
				if _, found := cad.k.GetPacketReceipt(ctx, msg.Packet.GetDestPort(), msg.Packet.GetDestChannel(), msg.Packet.GetSequence()); found {
					redundancies += 1
				}
				msgs += 1
			}
			if msg, ok := m.(*types.MsgAcknowledgement); ok {
				if commitment := cad.k.GetPacketCommitment(ctx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies += 1
				}
				msgs += 1
			}
			if msg, ok := m.(*types.MsgTimeout); ok {
				if commitment := cad.k.GetPacketCommitment(ctx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies += 1
				}
				msgs += 1
			}
			if msg, ok := m.(*types.MsgTimeoutOnClose); ok {
				if commitment := cad.k.GetPacketCommitment(ctx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies += 1
				}
				msgs += 1
			}
		}
		// return error if all packet messages are redundant
		if redundancies == msgs && msgs > 0 {
			return ctx, types.ErrRedundantTx
		}
	}
	return next(ctx, tx, simulate)
}
