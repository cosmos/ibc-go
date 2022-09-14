package types

import (
	"encoding/hex"
	"fmt"
)

const (
	SubModuleName = "wasm-manager"
)

func CodeID(codeID []byte) []byte {
	return []byte(fmt.Sprintf("code_id/%s", hex.EncodeToString(codeID)))
}
