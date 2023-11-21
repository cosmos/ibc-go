package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
)

// Checksum is a type alias used for wasm byte code checksums.
type Checksum []byte

func AddChecksum(ctx context.Context, cdc codec.BinaryCodec, storeKey storetypes.StoreKey, checksum Checksum) error {
	return nil
}

// GetAllChecksums is a helper to get all checksums from the store.
// It returns an empty slice if no checksums are found
func GetAllChecksums(ctx context.Context) ([]Checksum, error) {
	return nil, nil
	// TODO: fix this
	/*
		iterator, err := ibcwasm.Checksums.Iterate(ctx, nil)
		if err != nil {
			return nil, err
		}

		keys, err := iterator.Keys()
		if err != nil {
			return nil, err
		}

		checksums := []Checksum{}
		for _, key := range keys {
			checksums = append(checksums, key)
		}

		return checksums, nil
	*/
}

// HasChecksum returns true if the given checksum exists in the store and
// false otherwise.
func HasChecksum(ctx context.Context, checksum Checksum) bool {
	return false
	// TODO(jim): fix this
	/* found, err := ibcwasm.Checksums.Has(ctx, checksum)
	if err != nil {
		return false
	}

	return found
	*/
}
