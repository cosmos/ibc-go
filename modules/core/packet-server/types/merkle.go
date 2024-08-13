package types

import (
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
)

// BuildMerklePath takes the merkle path prefix and an ICS24 path
// and builds a new path by appending the ICS24 path to the last element of the merkle path prefix.
func BuildMerklePath(prefix commitmenttypesv2.MerklePath, path []byte) commitmenttypesv2.MerklePath {
	if prefix.Empty() {
		return commitmenttypes.NewMerklePath(path)
	}

	// avoid mutating the provided prefix
	prefixKeys := make([][]byte, len(prefix.KeyPath))
	copy(prefixKeys, prefix.KeyPath)

	lastElement := prefixKeys[len(prefixKeys)-1]
	// append path to last element
	newLastElement := cloneAppend(lastElement, path)
	prefixKeys[len(prefixKeys)-1] = newLastElement
	return commitmenttypes.NewMerklePath(prefixKeys...)
}

func cloneAppend(bz []byte, tail []byte) []byte {
	res := make([]byte, len(bz)+len(tail))
	copy(res, bz)
	copy(res[len(bz):], tail)
	return res
}
