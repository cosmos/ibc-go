package types

import (
	"bytes"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MigrateContract migrates the contract bytecode.
func (cs ClientState) MigrateContract(
	ctx sdk.Context, cdc codec.BinaryCodec, clientID string,
	clientStore storetypes.KVStore, newCodeHash []byte, migrateMsg []byte,
) error {
	if !HasCodeHash(ctx, cdc, newCodeHash) {
		return ErrWasmCodeHashNotFound
	}

	if bytes.Equal(cs.CodeHash, newCodeHash) {
		return nil
	}

	_, err := wasmMigrate[EmptyResult](ctx, clientID, clientStore, &cs, migrateMsg)
	if err != nil {
		return err
	}

	return nil
}
