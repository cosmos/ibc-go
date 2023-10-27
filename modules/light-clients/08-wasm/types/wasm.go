package types

import (
	"context"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

// getCodeHasheKeySet returns the KeySet collection for the code hashes.
func getCodeHashKeySet(ctx context.Context) collections.KeySet[[]byte] {
	wasmStoreService := ibcwasm.GetWasmStoreService()
	sb := collections.NewSchemaBuilder(wasmStoreService)

	return collections.NewKeySet(sb, CodeHashesKey, "code_hashes", collections.BytesKey)
}


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
func AddCodeHash(ctx sdk.Context, codeHash []byte) error {
	keyset := getCodeHashKeySet(ctx)

	return keyset.Set(ctx, codeHash)
}

// HasCodeHash returns true if the given code hash exists in the store and
// false otherwise.
func HasCodeHash(ctx sdk.Context, codeHash []byte) bool {
	keyset := getCodeHashKeySet(ctx)

	has, err := keyset.Has(ctx, codeHash)
	if err != nil {
		return false
	}

	return has
}
