package types

import (
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

// MaxWasmSize is the maximum size of a wasm code in bytes.
const MaxWasmSize = maxWasmSize

// these fields are exported aliases for the payload fields passed to the wasm vm.
// these are used to specify callback functions to handle specific queries in the mock vm.
type (
	// CallbackFn types
	QueryFn = queryFn
	SudoFn  = sudoFn
)

// WasmQuery wraps wasmQuery and is used solely for testing.
func WasmQuery[T ContractResult](ctx sdk.Context, clientStore storetypes.KVStore, cs *ClientState, payload QueryMsg) (T, error) {
	return wasmQuery[T](ctx, clientStore, cs, payload)
}

// WasmCall wraps wasmCall and is used solely for testing.
func WasmCall[T ContractResult](ctx sdk.Context, clientStore storetypes.KVStore, cs *ClientState, payload SudoMsg) (T, error) {
	return wasmCall[T](ctx, clientStore, cs, payload)
}

// WasmInit wraps wasmInit and is used solely for testing.
func WasmInit(ctx sdk.Context, clientStore storetypes.KVStore, cs *ClientState, payload InstantiateMessage) error {
	return wasmInit(ctx, clientStore, cs, payload)
}
