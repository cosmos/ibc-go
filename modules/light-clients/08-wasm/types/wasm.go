package types

import (
	"context"

	wasmvm "github.com/CosmWasm/wasmvm"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
)

// GetAllChecksums is a helper to get all checksums from the store.
// It returns an empty slice if no checksums are found
func GetAllChecksums(ctx context.Context) ([]wasmvm.Checksum, error) {
	iterator, err := ibcwasm.Checksums.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}

	keys, err := iterator.Keys()
	if err != nil {
		return nil, err
	}

	checksums := []wasmvm.Checksum{}
	for _, key := range keys {
		checksums = append(checksums, key)
	}

	return checksums, nil
}

// HasChecksum returns true if the given checksum exists in the store and
// false otherwise.
func HasChecksum(ctx context.Context, checksum wasmvm.Checksum) bool {
	found, err := ibcwasm.Checksums.Has(ctx, checksum)
	if err != nil {
		return false
	}

	return found
}
