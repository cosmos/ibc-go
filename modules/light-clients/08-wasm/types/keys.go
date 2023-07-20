package types

import (
	"encoding/hex"
	"fmt"
)

const (
	// ModuleName for the wasm client
	ModuleName = "08-wasm"

	// StoreKey is the store key string for 08-wasm
	StoreKey = ModuleName

	KeyCodeHashPrefix = "codeHash"
)

// CodeHashKey returns a store key under which the wasm code for a light client
// is stored in a client prefixed store
func CodeHashKey(codeHash []byte) []byte {
	return []byte(fmt.Sprintf("%s/%s", KeyCodeHashPrefix, hex.EncodeToString(codeHash)))
}
