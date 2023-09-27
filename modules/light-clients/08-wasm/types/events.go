package types

import (
	"fmt"

	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// IBC 08-wasm events
const (
	EventTypeStoreWasmCode = "store_wasm_code"

	AttributeKeyWasmCodeHash = "wasm_code_hash"
)

var AttributeValueCategory = fmt.Sprintf("%s_%s", ibcexported.ModuleName, ModuleName)
