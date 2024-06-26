package types

import (
	"unicode/utf8"

	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types/v2"
)

// IsValidUTF8 returns true if the provided key path bytes contain valid utf8 encoded runes.
func IsValidUTF8(keyPath [][]byte) bool {
	for _, bz := range keyPath {
		if !utf8.Valid(bz) {
			return false
		}
	}

	return true
}

// ToLegacyMerklePath converts a v2 23-commitment MerklePath to a v1 23-commitment MerklePath.
func ToLegacyMerklePath(merklePath commitmenttypesv2.MerklePath) commitmenttypes.MerklePath {
	var keyPath []string
	for _, bz := range merklePath.KeyPath {
		keyPath = append(keyPath, string(bz))
	}

	return commitmenttypes.MerklePath{
		KeyPath: keyPath,
	}
}
