package types

import (
	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

// GetCodeHasheKeySet returns the KeySet collection for the code hashes.
func GetCodeHashKeySet() collections.KeySet[[]byte] {
	wasmStoreService := ibcwasm.GetWasmStoreService()
	sb := collections.NewSchemaBuilder(wasmStoreService)

	return collections.NewKeySet(sb, CodeHashesKey, "code_hashes", collections.BytesKey)
}

// GetAllCodeHashes is a helper to get all code hashes from the store.
// It returns an empty slice if no code hashes are found
func GetAllCodeHashes(ctx sdk.Context) ([][]byte, error) {
	keyset := GetCodeHashKeySet()

	iterator, err := keyset.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}

	codeHashes, err := iterator.Keys()
	if err != nil {
		return nil, err
	}

	if codeHashes == nil {
		codeHashes = [][]byte{}
	}

	return codeHashes, nil
}

// AddCodeHash adds a new code hash to the list of stored code hashes.
func AddCodeHash(ctx sdk.Context, codeHash []byte) error {
	keyset := GetCodeHashKeySet()

	return keyset.Set(ctx, codeHash)
}

// HasCodeHash returns true if the given code hash exists in the store and
// false otherwise.
func HasCodeHash(ctx sdk.Context, codeHash []byte) bool {
	keyset := GetCodeHashKeySet()

	has, err := keyset.Has(ctx, codeHash)
	if err != nil {
		return false
	}

	return has
}
