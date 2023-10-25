package types

import (
	"bytes"
	"slices"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

// GetCodeHashes returns all the code hashes stored.
func GetCodeHashes(ctx sdk.Context, cdc codec.BinaryCodec) (CodeHashes, error) {
	wasmStoreKey := ibcwasm.GetWasmStoreKey()
	store := ctx.KVStore(wasmStoreKey)
	bz := store.Get([]byte(KeyCodeHashes))
	if len(bz) == 0 {
		return CodeHashes{}, nil
	}
	var hashes CodeHashes
	err := cdc.Unmarshal(bz, &hashes)
	if err != nil {
		return CodeHashes{}, err
	}
	return hashes, nil
}

// AddCodeHash adds a new code hash to the list of stored code hashes.
func AddCodeHash(ctx sdk.Context, cdc codec.BinaryCodec, codeHash []byte) error {
	codeHashes, err := GetCodeHashes(ctx, cdc)
	if err != nil {
		return err
	}

	codeHashes.Hashes = append(codeHashes.Hashes, codeHash)

	wasmStoreKey := ibcwasm.GetWasmStoreKey()
	store := ctx.KVStore(wasmStoreKey)
	bz, err := cdc.Marshal(&codeHashes)
	if err != nil {
		return err
	}

	store.Set([]byte(KeyCodeHashes), bz)

	return nil
}

// HasCodeHash returns true if the given code hash exists in the store and
// false otherwise.
func HasCodeHash(ctx sdk.Context, cdc codec.BinaryCodec, codeHash []byte) bool {
	codeHashes, err := GetCodeHashes(ctx, cdc)
	if err != nil {
		return false
	}

	return slices.ContainsFunc(codeHashes.Hashes, func(h []byte) bool { return bytes.Equal(codeHash, h) })
}
