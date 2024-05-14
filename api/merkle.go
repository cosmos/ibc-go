package api

import (
	"errors"
	"fmt"
)

// NewMerklePrefix constructs new MerklePrefix instance
func NewMerklePrefix(keyPrefix []byte) MerklePrefix {
	return MerklePrefix{
		KeyPrefix: keyPrefix,
	}
}

// Bytes returns the key prefix bytes
func (mp MerklePrefix) Bytes() []byte {
	return mp.KeyPrefix
}

// Empty returns true if the prefix is empty
func (mp MerklePrefix) Empty() bool {
	return len(mp.Bytes()) == 0
}

// NewMerklePath creates a new MerklePath instance
// The keys must be passed in from root-to-leaf order
func NewMerklePath(keyPath ...string) MerklePath {
	return MerklePath{
		KeyPath: keyPath,
	}
}

// GetKey will return a byte representation of the key
func (mp MerklePath) GetKey(i uint64) ([]byte, error) {
	if i >= uint64(len(mp.KeyPath)) {
		return nil, fmt.Errorf("index out of range. %d (index) >= %d (len)", i, len(mp.KeyPath))
	}
	return []byte(mp.KeyPath[i]), nil
}

// Empty returns true if the path is empty
func (mp MerklePath) Empty() bool {
	return len(mp.KeyPath) == 0
}

// ApplyPrefix constructs a new commitment path from the arguments. It prepends the prefix key
// with the given path.
func ApplyPrefix(prefix []byte, path MerklePath) (MerklePath, error) {
	if len(prefix) == 0 {
		return MerklePath{}, errors.New("prefix can't be empty")
	}
	return NewMerklePath(append([]string{string(prefix)}, path.KeyPath...)...), nil
}
