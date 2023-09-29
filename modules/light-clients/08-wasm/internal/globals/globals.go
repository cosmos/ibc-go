package globals

import (
	wasmvm "github.com/CosmWasm/wasmvm"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
)

var (
	WasmVM *wasmvm.VM
	// Store key for 08-wasm module, required as a global so that the KV store can be retrieved
	// in the ClientState Initialize function which doesn't have access to the keeper.
	// The storeKey is used to check the code hash of the contract and determine if the light client
	// is allowed to be instantiated.
	WasmStoreKey storetypes.StoreKey
)
