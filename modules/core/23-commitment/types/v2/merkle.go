package v2

import (
	"errors"
	"fmt"

	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

var _ exported.Path = (*MerklePath)(nil)

// NewMerklePath creates a new MerklePath instance
// The keys must be passed in from root-to-leaf order
func NewMerklePath(keyPath ...[]byte) MerklePath {
	return MerklePath{
		KeyPath: keyPath,
	}
}

// GetKey will return a byte representation of the key
func (mp MerklePath) GetKey(i uint64) ([]byte, error) {
	if i >= uint64(len(mp.KeyPath)) {
		return nil, fmt.Errorf("index out of range. %d (index) >= %d (len)", i, len(mp.KeyPath))
	}
	return mp.KeyPath[i], nil
}

// Empty returns true if the path is empty
func (mp MerklePath) Empty() bool {
	return len(mp.KeyPath) == 0
}

// ValidateAsPrefix validates the MerklePath to ensure it is a valid prefix
// Thus every element of the merkle path must be non-empty except for the last element
// which may be empty. In this case, the ICS24 path will be appended to the last element
// to form the full path.
func (mp MerklePath) ValidateAsPrefix() error {
	if mp.Empty() {
		return errors.New("path cannot have length 0")
	}

	for i, key := range mp.KeyPath {
		if len(key) == 0 && i != len(mp.KeyPath)-1 {
			return fmt.Errorf("key at index %d cannot be empty", i)
		}
	}
	return nil
}

// ValidateAsPath validates the MerklePath as a fully constructed path.
// Here every element must be non-empty since the MerklePath is no longer
// acting as a prefix but is instead the full path intended for verification.
func (mp MerklePath) ValidateAsPath() error {
	if mp.Empty() {
		return errors.New("path cannot have length 0")
	}

	for i, key := range mp.KeyPath {
		if len(key) == 0 {
			return fmt.Errorf("key at index %d cannot be empty", i)
		}
	}
	return nil
}
