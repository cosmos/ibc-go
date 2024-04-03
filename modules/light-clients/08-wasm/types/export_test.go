package types

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

// MaxWasmSize is the maximum size of a wasm code in bytes.
const MaxWasmSize = maxWasmSize

var (
	SubjectPrefix    = subjectPrefix
	SubstitutePrefix = substitutePrefix
)

// NewMigrateProposalWrappedStore is a wrapper around newMigrateProposalWrappedStore to allow the function to be directly called in tests.
//
//nolint:revive // Returning unexported type for testing purposes.
func NewMigrateProposalWrappedStore(subjectStore, substituteStore storetypes.KVStore) migrateClientWrappedStore {
	return NewMigrateClientWrappedStore(subjectStore, substituteStore)
}

// GetStore is a wrapper around getStore to allow the function to be directly called in tests.
func (ws migrateClientWrappedStore) GetStore(key []byte) (storetypes.KVStore, bool) {
	return ws.getStore(key)
}

// SplitPrefix is a wrapper around splitKey to allow the function to be directly called in tests.
func SplitPrefix(key []byte) ([]byte, []byte) {
	return splitPrefix(key)
}

// WasmMigrate wraps wasmMigrate and is used solely for testing.
func WasmMigrate(ctx sdk.Context, vm ibcwasm.WasmEngine, cdc codec.BinaryCodec, clientStore storetypes.KVStore, cs *ClientState, clientID string, payload []byte) error {
	return wasmMigrate(ctx, vm, cdc, clientStore, cs, clientID, payload)
}
