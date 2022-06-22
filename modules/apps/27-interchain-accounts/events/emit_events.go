package events

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/types"
)

// EmitAcknowledgementErrorEvent emits an acknowledgement error event.
func EmitAcknowledgementErrorEvent(ctx sdk.Context, err error) {
	if err != nil {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypePacket,
				sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
				sdk.NewAttribute(types.AttributeKeyAckError, err.Error()),
			),
		)
	}
}
