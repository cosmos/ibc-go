package types

import (
	"bytes"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetCodeHashes returns all the code hashes stored.
func GetCodeHashes(ctx sdk.Context, cdc codec.BinaryCodec) CodeHashes {
	store := ctx.KVStore(WasmStoreKey)
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
	hashes := GetCodeHashes(ctx, cdc)
	hashes.CodeHashes = append(hashes.CodeHashes, codeHash)

	store := ctx.KVStore(WasmStoreKey)
	bz := cdc.MustMarshal(&hashes)
	store.Set([]byte(KeyCodeHashes), bz)
}

// HasCodeHash returns true if the given code hash exists in the store and
// false otherwise.
func HasCodeHash(ctx sdk.Context, cdc codec.BinaryCodec, codeHash []byte) bool {
	hashes := GetCodeHashes(ctx, cdc)

	for _, hash := range hashes.CodeHashes {
		if bytes.Equal(hash, codeHash) {
			return true
		}
	}
	return false
}
