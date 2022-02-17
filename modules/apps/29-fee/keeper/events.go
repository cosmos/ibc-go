package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

// EmitIncentivizedPacket emits an event so that relayers know an incentivized packet is ready to be relayed
func EmitIncentivizedPacket(ctx sdk.Context, identifiedFee types.IdentifiedPacketFee) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeIncentivizedPacket,
			sdk.NewAttribute(channeltypes.AttributeKeyPortID, identifiedFee.PacketId.PortId),
			sdk.NewAttribute(channeltypes.AttributeKeyChannelID, identifiedFee.PacketId.ChannelId),
			sdk.NewAttribute(channeltypes.AttributeKeySequence, fmt.Sprint(identifiedFee.PacketId.Sequence)),
			sdk.NewAttribute(types.AttributeKeyRecvFee, fmt.Sprint(identifiedFee.Fee.RecvFee)),
			sdk.NewAttribute(types.AttributeKeyAckFee, fmt.Sprint(identifiedFee.Fee.AckFee)),
			sdk.NewAttribute(types.AttributeKeyTimeoutFee, fmt.Sprint(identifiedFee.Fee.TimeoutFee)),
		),
	)
}
