package types

import commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"

// LegacyMerklePath defines a struct containing a key path.
// This maintains backwards compatibility for encoding Path fields in VerifyMembershipMsg and VerifyNonMembershipMsg contract api types.
type LegacyMerklePath struct {
	KeyPath []string `json:"key_path,omitempty"`
}

// ToLegacyMerklePath takes a 23-commitment MerklePath and converts its key path parts from bytes to strings.
func ToLegacyMerklePath(merklePath commitmenttypes.MerklePath) LegacyMerklePath {
	var keyPath []string
	for _, bz := range merklePath.KeyPath {
		keyPath = append(keyPath, string(bz))
	}

	return LegacyMerklePath{
		KeyPath: keyPath,
	}
}
