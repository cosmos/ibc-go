package types

import (
	"bytes"
	"slices"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

// Checksum is a type alias used for wasm byte code checksums.
type Checksum = []byte

// GetAllChecksums is a helper to get all checksums from the store.
// It returns an empty slice if no checksums are found
func GetAllChecksums(ctx sdk.Context, cdc codec.BinaryCodec) ([]Checksum, error) {
	wasmStoreKey := ibcwasm.GetWasmStoreKey()
	store := ctx.KVStore(wasmStoreKey)

	bz := store.Get([]byte(KeyChecksums))
	if len(bz) == 0 {
		return []Checksum{}, nil
	}

	var hashes Checksums
	err := cdc.Unmarshal(bz, &hashes)
	if err != nil {
		return []Checksum{}, err
	}
	return hashes.Checksums, nil
}

// AddChecksum adds a checksum to the list of stored checksums in state.
func AddChecksum(ctx sdk.Context, cdc codec.BinaryCodec, storeKey storetypes.StoreKey, checksum Checksum) error {
	store := ctx.KVStore(storeKey)
	checksums, err := GetAllChecksums(ctx, cdc)
	if err != nil {
		return err
	}

	checksums = append(checksums, checksum)
	hashes := Checksums{Checksums: checksums}

	bz, err := cdc.Marshal(&hashes)
	if err != nil {
		return err
	}
	store.Set([]byte(KeyChecksums), bz)

	return nil
}

// HasChecksum returns true if the given checksum exists in the store and
// false otherwise.
func HasChecksum(ctx sdk.Context, cdc codec.BinaryCodec, checksum Checksum) bool {
	checksums, err := GetAllChecksums(ctx, cdc)
	if err != nil {
		return false
	}

	return slices.ContainsFunc(checksums, func(h Checksum) bool { return bytes.Equal(checksum, h) })
}

// RemoveChecksum removes the given checksum from the list of stored checksums in state.
func RemoveChecksum(ctx sdk.Context, cdc codec.BinaryCodec, storeKey storetypes.StoreKey, checksum Checksum) error {
	store := ctx.KVStore(storeKey)
	checksums, err := GetAllChecksums(ctx, cdc)
	if err != nil {
		return err
	}

	checksums = slices.DeleteFunc(checksums, func(h Checksum) bool { return bytes.Equal(checksum, h) })
	hashes := Checksums{Checksums: checksums}

	bz, err := cdc.Marshal(&hashes)
	if err != nil {
		return err
	}
	store.Set([]byte(KeyChecksums), bz)

	return nil
}
