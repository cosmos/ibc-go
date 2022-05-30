package types

import (
	"encoding/hex"

	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	tmtypes "github.com/tendermint/tendermint/types"
)

// ParseHexHash parses a hex hash in string format to bytes and validates its correctness.
func ParseHexHash(hexHash string) (tmbytes.HexBytes, error) {
	hash, err := hex.DecodeString(hexHash)
	if err != nil {
		return nil, err
	}

	if err := tmtypes.ValidateHash(hash); err != nil {
		return nil, err
	}

	return hash, nil
}

// GetFullClassPath returns the full classId according to the ICS721 specification:
// tracePath + "/" + BaseClassId
// If there exists no trace then the base BaseClassId is returned.
func (ct ClassTrace) GetFullClassPath() string {
	if ct.Path == "" {
		return ct.BaseClassId
	}
	return ct.GetPrefix() + ct.BaseClassId
}

// GetPrefix returns the receiving classId prefix composed by the trace info and a separator.
func (ct ClassTrace) GetPrefix() string {
	return ct.Path + "/"
}
