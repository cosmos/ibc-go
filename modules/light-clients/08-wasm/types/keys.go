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

	LastInstanceIDKey = "lastInstanceId"
)

var PrefixCodeIDKey = []byte("code_id/")

func CodeID(codeID []byte) []byte {
	return []byte(fmt.Sprintf("code_id/%s", hex.EncodeToString(codeID)))
}
