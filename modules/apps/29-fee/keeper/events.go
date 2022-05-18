package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

// EmitIncentivizedPacketEvent emits an event containing information on the total amount of fees incentivizing
// a specific packet. It should be emitted on every fee escrowed for the given packetID.
func EmitIncentivizedPacketEvent(ctx sdk.Context, packetID channeltypes.PacketId, packetFees types.PacketFees) {
	var (
		totalRecvFees    sdk.Coins
		totalAckFees     sdk.Coins
		totalTimeoutFees sdk.Coins
	)

	for _, fee := range packetFees.PacketFees {
		// only emit total fees for packet fees which allow any relayer to relay
		if fee.Relayers == nil {
			totalRecvFees = totalRecvFees.Add(fee.Fee.RecvFee...)
			totalAckFees = totalAckFees.Add(fee.Fee.AckFee...)
			totalTimeoutFees = totalTimeoutFees.Add(fee.Fee.TimeoutFee...)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeIncentivizedPacket,
			sdk.NewAttribute(channeltypes.AttributeKeyPortID, packetID.PortId),
			sdk.NewAttribute(channeltypes.AttributeKeyChannelID, packetID.ChannelId),
			sdk.NewAttribute(channeltypes.AttributeKeySequence, fmt.Sprint(packetID.Sequence)),
			sdk.NewAttribute(types.AttributeKeyRecvFee, totalRecvFees.String()),
			sdk.NewAttribute(types.AttributeKeyAckFee, totalAckFees.String()),
			sdk.NewAttribute(types.AttributeKeyTimeoutFee, totalTimeoutFees.String()),
		),
	)
}
