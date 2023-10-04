package types

import (
	"bytes"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

// GetCodeHashes returns all the code hashes stored.
func GetCodeHashes(ctx sdk.Context, cdc codec.BinaryCodec) CodeHashes {
	wasmStoreKey := ibcwasm.GetWasmStoreKey(cdc)
	store := ctx.KVStore(wasmStoreKey)
	bz := store.Get([]byte(KeyCodeHashes))
	if len(bz) == 0 {
		return CodeHashes{}
	}
	var hashes CodeHashes
	cdc.MustUnmarshal(bz, &hashes)
	return hashes
}

// AddCodeHash adds a new code hash to the list of stored code hashes.
func AddCodeHash(ctx sdk.Context, cdc codec.BinaryCodec, codeHash []byte) {
	codeHashes := GetCodeHashes(ctx, cdc)
	codeHashes.Hashes = append(codeHashes.Hashes, codeHash)

	wasmStoreKey := ibcwasm.GetWasmStoreKey(cdc)
	store := ctx.KVStore(wasmStoreKey)
	bz := cdc.MustMarshal(&codeHashes)
	store.Set([]byte(KeyCodeHashes), bz)
}

// HasCodeHash returns true if the given code hash exists in the store and
// false otherwise.
func HasCodeHash(ctx sdk.Context, cdc codec.BinaryCodec, codeHash []byte) bool {
	codeHashes := GetCodeHashes(ctx, cdc)

	for _, hash := range codeHashes.Hashes {
		if bytes.Equal(hash, codeHash) {
			return true
		}
	}
	return false
}
