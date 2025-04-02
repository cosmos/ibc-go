package types

import (
	"slices"

	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
)

// BuildMerklePath takes the merkle path prefix and an ICS24 path
// and builds a new path by appending the ICS24 path to the last element of the merkle path prefix.
func BuildMerklePath(prefix [][]byte, path []byte) commitmenttypesv2.MerklePath {
	prefixLength := len(prefix)
	if prefixLength == 0 {
		panic("cannot build merkle path with empty prefix")
	}

	// copy prefix to avoid modifying the original slice
	fullPath := slices.Clone(prefix)
	// append path to last element
	fullPath[prefixLength-1] = append(fullPath[prefixLength-1], path...)
	return commitmenttypesv2.NewMerklePath(fullPath...)
}
