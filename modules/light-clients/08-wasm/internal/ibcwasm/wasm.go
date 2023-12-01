package ibcwasm

import (
	"errors"

<<<<<<< HEAD
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
=======
	wasmvm "github.com/CosmWasm/wasmvm"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
>>>>>>> 7016a94e (feat: add custom queries to wasm module (#5261))
)

var (
	vm WasmEngine

<<<<<<< HEAD
	// wasmStoreKey stores the storeKey for the 08-wasm module. Using a global storeKey is required since
	// the client state interface functions do not have access to the keeper.
	wasmStoreKey storetypes.StoreKey
=======
	querier wasmvm.Querier

	// state management
	Schema    collections.Schema
	Checksums collections.KeySet[[]byte]

	// ChecksumsKey is the key under which all checksums are stored
	ChecksumsKey = collections.NewPrefix(0)
>>>>>>> 7016a94e (feat: add custom queries to wasm module (#5261))
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
=======
// SetQuerier sets the custom wasm query handle for the 08-wasm module.
// If wasmQuerier is nil a default querier is used that return always an error for any query.
func SetQuerier(wasmQuerier wasmvm.Querier) {
	if wasmQuerier == nil {
		querier = &defaultQuerier{}
	} else {
		querier = wasmQuerier
	}
}

// GetQuerier returns the custom wasm query handler for the 08-wasm module.
func GetQuerier() wasmvm.Querier {
	return querier
}

// SetupWasmStoreService sets up the 08-wasm module's collections.
func SetupWasmStoreService(storeService storetypes.KVStoreService) {
	sb := collections.NewSchemaBuilder(storeService)

	Checksums = collections.NewKeySet(sb, ChecksumsKey, "checksums", collections.BytesKey)

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	Schema = schema
>>>>>>> 7016a94e (feat: add custom queries to wasm module (#5261))
}
