package types

import (
	storetypes "cosmossdk.io/store/types"
)

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

// MaxWasmSize is the maximum size of a wasm code in bytes.
const MaxWasmSize = maxWasmSize

func GetClientID(clientStore storetypes.KVStore) (string, error) {
	return getClientID(clientStore)
}
