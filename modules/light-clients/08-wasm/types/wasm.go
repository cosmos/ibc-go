package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

// GetAllCodeHashes is a helper to get all code hashes from the store.
// It returns an empty slice if no code hashes are found
func GetAllCodeHashes(ctx sdk.Context) ([][]byte, error) {
	iterator, err := ibcwasm.CodeHashes.Iterate(ctx, nil)
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

// HasCodeHash returns true if the given code hash exists in the store and
// false otherwise.
func HasCodeHash(ctx sdk.Context, codeHash []byte) bool {
	found, err := ibcwasm.CodeHashes.Has(ctx, codeHash)
	if err != nil {
		return false
	}

	return found
}
