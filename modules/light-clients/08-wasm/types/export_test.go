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

// GetClientID is a wrapper around getClientID to allow the function to be directly called in tests.
func GetClientID(clientStore storetypes.KVStore) (string, error) {
	return getClientID(clientStore)
}

// NewUpdateProposalWrappedStore is a wrapper around newUpdateProposalWrappedStore to allow the function to be directly called in tests.
//
//nolint:revive // Returning unexported type for testing purposes.
func NewUpdateProposalWrappedStore(subjectStore, substituteStore storetypes.KVStore, subjectPrefix, substitutePrefix []byte) updateProposalWrappedStore {
	return newUpdateProposalWrappedStore(subjectStore, substituteStore, subjectPrefix, substitutePrefix)
}

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
