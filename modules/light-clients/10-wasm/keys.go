package wasm

import (
	"encoding/hex"
	"fmt"
)

const (
	// SubModuleName for the wasm client
	SubModuleName     = "wasm-client"
	LastInstanceIDKey = "lastInstanceId"
)

func CodeID(codeID []byte) []byte {
	return []byte(fmt.Sprintf("code_id/%s", hex.EncodeToString(codeID)))
}
