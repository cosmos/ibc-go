package types

import (
	storetypes "cosmossdk.io/store/types"
)

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

// MaxWasmSize is the maximum size of a wasm code in bytes.
const MaxWasmSize = maxWasmSize

// NewMigrateProposalWrappedStore is a wrapper around newMigrateProposalWrappedStore to allow the function to be directly called in tests.
//
//nolint:revive // Returning unexported type for testing purposes.
func NewMigrateProposalWrappedStore(subjectStore, substituteStore storetypes.KVStore) MigrateClientWrappedStore {
	return NewMigrateClientWrappedStore(subjectStore, substituteStore)
}

// GetStore is a wrapper around getStore to allow the function to be directly called in tests.
func (ws MigrateClientWrappedStore) GetStore(key []byte) (storetypes.KVStore, bool) {
	return ws.getStore(key)
}

// SplitPrefix is a wrapper around splitKey to allow the function to be directly called in tests.
func SplitPrefix(key []byte) ([]byte, []byte) {
	return splitPrefix(key)
}
