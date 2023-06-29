package keeper

import (
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// emitCreateClientEvent emits a create client event
func emitStoreWasmCodeEvent(ctx sdk.Context, codeID []byte) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			clienttypes.EventTypeStoreWasmCode,
			sdk.NewAttribute(clienttypes.AttributeKeyWasmCodeID, hex.EncodeToString(codeID)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, clienttypes.AttributeValueCategory),
		),
	})
}
