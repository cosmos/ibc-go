package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

// EmitIncentivizedPacket emits an event so that relayers know an incentivized packet is ready to be relayed
func EmitIncentivizedPacket(ctx sdk.Context, packetID channeltypes.PacketId, packetFee types.PacketFee) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeIncentivizedPacket,
			sdk.NewAttribute(channeltypes.AttributeKeyPortID, packetID.PortId),
			sdk.NewAttribute(channeltypes.AttributeKeyChannelID, packetID.ChannelId),
			sdk.NewAttribute(channeltypes.AttributeKeySequence, fmt.Sprint(packetID.Sequence)),
			sdk.NewAttribute(types.AttributeKeyRecvFee, packetFee.Fee.RecvFee.String()),
			sdk.NewAttribute(types.AttributeKeyAckFee, packetFee.Fee.AckFee.String()),
			sdk.NewAttribute(types.AttributeKeyTimeoutFee, packetFee.Fee.TimeoutFee.String()),
		),
	)
}
