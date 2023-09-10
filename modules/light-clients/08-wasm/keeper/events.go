package keeper

import (
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
)

// emitCreateClientEvent emits a create client event
func emitStoreWasmCodeEvent(ctx sdk.Context, codeHash []byte) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			clienttypes.EventTypeStoreWasmCode,
			sdk.NewAttribute(clienttypes.AttributeKeyWasmCodeHash, hex.EncodeToString(codeHash)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, clienttypes.AttributeValueCategory),
		),
	})
}
