package ibcwasm

import (
	storetypes "cosmossdk.io/core/store"
)

var (
	vm WasmEngine
	// wasmStoreService stores the key-value storage service for the 08-wasm module.
	// Using a global storage service is required since the client state interface functions
	// do not have access to the keeper.
	wasmStoreService storetypes.KVStoreService
)

// SetVM sets the wasm VM for the 08-wasm module.
func SetVM(wasmVM WasmEngine) {
	vm = wasmVM
}

// GetVM returns the wasm VM for the 08-wasm module.
func GetVM() WasmEngine {
	return vm
}

// SetWasmStoreKey sets the store key for the 08-wasm module.
func SetWasmStoreService(storeService storetypes.KVStoreService) {
	wasmStoreService = storeService
}

// GetWasmStoreKey returns the store key for the 08-wasm module.
func GetWasmStoreService() storetypes.KVStoreService {
	return wasmStoreService
}
