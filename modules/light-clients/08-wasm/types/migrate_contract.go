package types

import (
	"bytes"
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MigrateContract calls the migrate entry point on the contract with the given
// migrateMsg. The contract must exist and the code hash must be found in the
// store. If the code hash is the same as the current code hash, return nil.
// This does not update the code hash in the client state.
func (cs ClientState) MigrateContract(
	ctx sdk.Context, cdc codec.BinaryCodec, clientStore storetypes.KVStore,
	clientID string, newCodeHash, migrateMsg []byte,
) error {
	if !HasCodeHash(ctx, newCodeHash) {
		return ErrWasmCodeHashNotFound
	}

	if bytes.Equal(cs.CodeHash, newCodeHash) {
		return errorsmod.Wrapf(ErrWasmCodeExists, "new code hash (%s) is the same as current code hash (%s)", hex.EncodeToString(newCodeHash), hex.EncodeToString(cs.CodeHash))
	}

	_, err := wasmMigrate[EmptyResult](ctx, clientID, clientStore, &cs, migrateMsg)
	if err != nil {
		return err
	}

	return nil
}
