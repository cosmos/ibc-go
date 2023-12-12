package ibcwasm

import (
	"errors"

<<<<<<< HEAD
	wasmvm "github.com/CosmWasm/wasmvm"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
)

var (
	vm      WasmEngine
	querier wasmvm.Querier
=======
	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
)

var (
	vm WasmEngine

	queryRouter  QueryRouter
	queryPlugins QueryPluginsI
>>>>>>> e2bcb775 (feat(08-wasm): querier plugins implemented (#5345))

	// wasmStoreKey stores the storeKey for the 08-wasm module. Using a global storeKey is required since
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

<<<<<<< HEAD
// SetWasmStoreKey sets the store key for the 08-wasm module.
func SetWasmStoreKey(storeKey storetypes.StoreKey) {
	wasmStoreKey = storeKey
}

// GetWasmStoreKey returns the store key for the 08-wasm module.
func GetWasmStoreKey() storetypes.StoreKey {
	return wasmStoreKey
}

// SetQuerier sets the custom wasm query handle for the 08-wasm module.
// If wasmQuerier is nil a default querier is used that return always an error for any query.
func SetQuerier(wasmQuerier wasmvm.Querier) {
	if wasmQuerier == nil {
		querier = &defaultQuerier{}
	} else {
		querier = wasmQuerier
=======
// SetQueryRouter sets the custom wasm query router for the 08-wasm module.
// Panics if the queryRouter is nil.
func SetQueryRouter(router QueryRouter) {
	if router == nil {
		panic(errors.New("query router must be not nil"))
>>>>>>> e2bcb775 (feat(08-wasm): querier plugins implemented (#5345))
	}
	queryRouter = router
}

// GetQueryRouter returns the custom wasm query router for the 08-wasm module.
func GetQueryRouter() QueryRouter {
	return queryRouter
}

// SetQueryPlugins sets the current query plugins
func SetQueryPlugins(plugins QueryPluginsI) {
	if plugins == nil {
		panic(errors.New("query plugins must be not nil"))
	}
	queryPlugins = plugins
}

// GetQueryPlugins returns the current query plugins
func GetQueryPlugins() QueryPluginsI {
	return queryPlugins
}
