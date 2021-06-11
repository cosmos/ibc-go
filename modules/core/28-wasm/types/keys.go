package types

import "fmt"

const (
	SubModuleName = "wasm-manager"
)

func LatestWASMCode(clientType string) []byte {
	return []byte(fmt.Sprintf("%s/latest", clientType))
}

func WASMCode(clientType string, hash string) []byte {
	return []byte(fmt.Sprintf("%s/%s", clientType, hash))
}

func WASMCodeEntry(clientType string, codeID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/entry", clientType, codeID))
}
