package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

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

// GetClassPrefix returns the receiving denomination prefix
func GetClassPrefix(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/", portID, channelID)
}

// IsAwayFromOrigin determine if non-fungible token is moving away from
// the origin chain (the chain issued by the native nft).
// Note that fullClassPath refers to the full path of the unencoded classID.
// The longer the fullClassPath, the farther it is from the origin chain
func IsAwayFromOrigin(sourcePort, sourceChannel, fullClassPath string) bool {
	prefixClassID := GetClassPrefix(sourcePort, sourceChannel)
	if !strings.HasPrefix(fullClassPath, prefixClassID) {
		return true
	}
	return fullClassPath[:len(prefixClassID)] != prefixClassID
}

// ParseClassTrace parses a string with the ibc prefix (denom trace) and the base denomination
// into a DenomTrace type.
//
// Examples:
//
// 	- "portidone/channelidone/uatom" => DenomTrace{Path: "portidone/channelidone", BaseDenom: "uatom"}
// 	- "uatom" => DenomTrace{Path: "", BaseDenom: "uatom"}
func ParseClassTrace(rawClassID string) ClassTrace {
	classSplit := strings.Split(rawClassID, "/")

	if classSplit[0] == rawClassID {
		return ClassTrace{
			Path:        "",
			BaseClassId: rawClassID,
		}
	}

	return ClassTrace{
		Path:        strings.Join(classSplit[:len(classSplit)-1], "/"),
		BaseClassId: classSplit[len(classSplit)-1],
	}
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

// Hash returns the hex bytes of the SHA256 hash of the ClassTrace fields using the following formula:
//
// hash = sha256(tracePath + "/" + baseClassId)
func (ct ClassTrace) Hash() tmbytes.HexBytes {
	hash := sha256.Sum256([]byte(ct.GetFullClassPath()))
	return hash[:]
}

// IBCClassID a coin denomination for an ICS20 fungible token in the format
// 'ibc/{hash(tracePath + baseDenom)}'. If the trace is empty, it will return the base denomination.
func (ct ClassTrace) IBCClassID() string {
	if ct.Path != "" {
		return fmt.Sprintf("%s/%s", ClassPrefix, ct.Hash())
	}
	return ct.BaseClassId
}
