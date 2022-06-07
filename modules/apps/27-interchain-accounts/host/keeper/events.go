package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
<<<<<<< HEAD
	
=======
	icatypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/types"
>>>>>>> b2ca193 (Emit an event to indicate a successful acknowledgement in the ICA module (#1466))
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	icatypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/types"
)

// EmitAcknowledgementEvent emits an event signalling a successful or failed acknowledgement and including the error
// details if any.
func EmitAcknowledgementEvent(ctx sdk.Context, packet exported.PacketI, ack exported.Acknowledgement, err error) {
	var errorMsg string
	if err != nil {
		errorMsg = err.Error()
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			icatypes.EventTypePacket,
			sdk.NewAttribute(sdk.AttributeKeyModule, icatypes.ModuleName),
			sdk.NewAttribute(icatypes.AttributeKeyAckError, errorMsg),
			sdk.NewAttribute(icatypes.AttributeKeyHostChannelID, packet.GetDestChannel()),
			sdk.NewAttribute(icatypes.AttributeKeyAckSuccess, fmt.Sprintf("%t", ack.Success())),
		),
	)
}
