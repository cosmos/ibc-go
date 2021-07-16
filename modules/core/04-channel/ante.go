package channel

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/core/04-channel/keeper"
	"github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

type ChannelAnteDecorator struct {
	k      keeper.Keeper
	strict bool
}

func NewChannelAnteDecorator(k keeper.Keeper, strict bool) ChannelAnteDecorator {
	return ChannelAnteDecorator{k: k, strict: strict}
}

func (cad ChannelAnteDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// do not run redundancy check on DeliverTx or simulate
	if (ctx.IsCheckTx() || ctx.IsReCheckTx()) && !simulate {
		// keep track of total packet messages and number of redundancies across `RecvPacket`, `AcknowledgePacket`, and `TimeoutPacket/OnClose`
		msgs := 0
		redundancies := 0
		for _, m := range tx.GetMsgs() {
			switch msg := m.(type) {
			case *types.MsgRecvPacket:
				if _, found := cad.k.GetPacketReceipt(ctx, msg.Packet.GetDestPort(), msg.Packet.GetDestChannel(), msg.Packet.GetSequence()); found {
					redundancies += 1
				}

			case *types.MsgAcknowledgement:
				if commitment := cad.k.GetPacketCommitment(ctx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies += 1
				}

			case *types.MsgTimeout:
				if commitment := cad.k.GetPacketCommitment(ctx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies += 1
				}

			case *types.MsgTimeoutOnClose:
				if commitment := cad.k.GetPacketCommitment(ctx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies += 1
				}
			default:
				continue
			}

			msgs += 1

		}

		if cad.strict && redundancies > 1 {
			// if strict, return error on single redundancy
			return ctx, types.ErrRedundantTx
		} else if redundancies == msgs && msgs > 0 {
			// if not strict, only return error if all packet messages are redundant
			return ctx, types.ErrRedundantTx
		}
	}
	return next(ctx, tx, simulate)
}
