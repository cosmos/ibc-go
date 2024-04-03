package types

import (
	"bytes"
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

// MigrateContract calls the migrate entry point on the contract with the given
// migrateMsg. The contract must exist and the checksum must be found in the
// store. If the checksum is the same as the current checksum, an error is returned.
// This does not update the checksum in the client state.
// TODO(jim): Move to lcm? Keeper?
func (cs ClientState) MigrateContract(
	ctx sdk.Context, vm ibcwasm.WasmEngine, cdc codec.BinaryCodec, clientStore storetypes.KVStore,
	clientID string, newChecksum, migrateMsg []byte,
) error {
	if !HasChecksum(ctx, newChecksum) {
		return ErrWasmChecksumNotFound
	}

	if bytes.Equal(cs.Checksum, newChecksum) {
		return errorsmod.Wrapf(ErrWasmCodeExists, "new checksum (%s) is the same as current checksum (%s)", hex.EncodeToString(newChecksum), hex.EncodeToString(cs.Checksum))
	}

	// update the checksum, this needs to be done before the contract migration
	// so that wasmMigrate can call the right code. Note that this is not
	// persisted to the client store.
	cs.Checksum = newChecksum

	err := wasmMigrate(ctx, vm, cdc, clientStore, &cs, clientID, migrateMsg)
	if err != nil {
		return err
	}

	return nil
}
