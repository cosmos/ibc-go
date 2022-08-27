package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"
)

// EmitCreateClientEvent emits a create client event
func EmitQueryEvent(ctx sdk.Context, query *types.MsgSubmitCrossChainQuery) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventSendQuery,
			sdk.NewAttribute(types.AttributeKeyTimeoutHeight, fmt.Sprintf("%d", query.GetTimeoutHeight())),
			sdk.NewAttribute(types.AttributeKeyTimeoutTimestamp, fmt.Sprintf("%d", query.GetTimeoutTimestamp())),
			sdk.NewAttribute(types.AttributeKeyQueryHeight, fmt.Sprintf("%d",query.GetQueryHeight())),
			sdk.NewAttribute(types.AttributeKeyQueryID, string(query.GetQueryId())),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
		),
	})
}