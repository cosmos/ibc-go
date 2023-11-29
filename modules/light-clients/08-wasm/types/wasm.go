package types

import (
	"bytes"
	"slices"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

// Checksum is a type alias used for wasm byte code checksums.
type Checksum = wasmvmtypes.Checksum

// CreateChecksum creates a sha256 checksum from the given wasm code, it forwards the
// call to the wasmvm package. The code is checked for the following conditions:
// - code length is zero.
// - code length is less than 4 bytes (magic number length).
// - code does not start with the wasm magic number.
func CreateChecksum(code []byte) (Checksum, error) {
	return wasmvm.CreateChecksum(code)
}

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

	var checksums []Checksum
	for _, checksum := range hashes.Checksums {
		checksums = append(checksums, checksum)
	}

	return checksums, nil
}

// AddChecksum adds a checksum to the list of stored checksums in state.
func AddChecksum(ctx sdk.Context, cdc codec.BinaryCodec, storeKey storetypes.StoreKey, checksum Checksum) error {
	store := ctx.KVStore(storeKey)
	checksums, err := GetAllChecksums(ctx, cdc)
	if err != nil {
		return err
	}

	checksums = append(checksums, checksum)

	var hashBz [][]byte
	for _, checksum := range checksums {
		hashBz = append(hashBz, checksum)
	}

	hashes := Checksums{Checksums: hashBz}
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

	var hashBz [][]byte
	for _, checksum := range checksums {
		hashBz = append(hashBz, checksum)
	}

	hashes := Checksums{Checksums: hashBz}
	bz, err := cdc.Marshal(&hashes)
	if err != nil {
		return err
	}
	store.Set([]byte(KeyChecksums), bz)

	return nil
}
