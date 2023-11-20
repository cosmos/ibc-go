package ibcwasm

import (
	"errors"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
)

var (
	vm WasmEngine

	// storeKeyMap stores the storeKey for the 08-wasm module. Using a global storeKey is required since
	// the client state interface functions do not have access to the keeper.
	wasmStoreKey storetypes.StoreKey
)

// SetVM sets the wasm VM for the 08-wasm module.
// It panics if the wasm VM is nil.
func SetVM(wasmVM WasmEngine) {
	if wasmVM == nil {
		panic(errors.New("wasm VM must be not nil"))
	}
	vm = wasmVM
}

// GetVM returns the wasm VM for the 08-wasm module.
func GetVM() WasmEngine {
	return vm
}

// SetWasmStoreKey sets the store key for the 08-wasm module.
func SetWasmStoreKey(storeKey storetypes.StoreKey) {
	wasmStoreKey = storeKey
}

// GetWasmStoreKey returns the store key for the 08-wasm module.
func GetWasmStoreKey() storetypes.StoreKey {
	return wasmStoreKey
}
