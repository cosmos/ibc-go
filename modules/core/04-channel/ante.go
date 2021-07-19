package channel

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/core/04-channel/keeper"
	"github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

type ChannelAnteDecorator struct {
	k keeper.Keeper
}

func NewChannelAnteDecorator(k keeper.Keeper) ChannelAnteDecorator {
	return ChannelAnteDecorator{k: k}
}

// ChannelAnteDecorator returns an error if a multiMsg tx only contains packet messages (Recv, Ack, Timeout) and additional update messages and all packet messages
// are redundant. If the transaction is just a single UpdateClient message, or the multimsg transaction contains some other message type, then the antedecorator returns no error
// and continues processing to ensure these transactions are included.
// This will ensure that relayers do not waste fees on multiMsg transactions when another relayer has already submitted all packets, by rejecting the tx at the mempool layer.
func (cad ChannelAnteDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// do not run redundancy check on DeliverTx or simulate
	if (ctx.IsCheckTx() || ctx.IsReCheckTx()) && !simulate {
		// keep track of total packet messages and number of redundancies across `RecvPacket`, `AcknowledgePacket`, and `TimeoutPacket/OnClose`
		redundancies := 0
		packetMsgs := 0
		for _, m := range tx.GetMsgs() {
			switch msg := m.(type) {
			case *types.MsgRecvPacket:
				if _, found := cad.k.GetPacketReceipt(ctx, msg.Packet.GetDestPort(), msg.Packet.GetDestChannel(), msg.Packet.GetSequence()); found {
					redundancies += 1
				}
				packetMsgs += 1

			case *types.MsgAcknowledgement:
				if commitment := cad.k.GetPacketCommitment(ctx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies += 1
				}
				packetMsgs += 1

			case *types.MsgTimeout:
				if commitment := cad.k.GetPacketCommitment(ctx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies += 1
				}
				packetMsgs += 1

			case *types.MsgTimeoutOnClose:
				if commitment := cad.k.GetPacketCommitment(ctx, msg.Packet.GetSourcePort(), msg.Packet.GetSourceChannel(), msg.Packet.GetSequence()); len(commitment) == 0 {
					redundancies += 1
				}
				packetMsgs += 1

			case *clienttypes.MsgUpdateClient:
				// do nothing here, as we want to avoid updating clients if it is batched with only redundant messages

			default:
				// if the multiMsg tx has a msg that is not a packet msg or update msg, then we will not return error
				// regardless of if all packet messages are redundant. This ensures that non-packet messages get processed
				// even if they get batched with redundant packet messages.
				return next(ctx, tx, simulate)
			}

		}

		// only return error if all packet messages are redundant
		if redundancies == packetMsgs && packetMsgs > 0 {
			return ctx, types.ErrRedundantTx
		}
	}
	return next(ctx, tx, simulate)
}
