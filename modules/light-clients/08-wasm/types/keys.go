package types

import "cosmossdk.io/collections"

const (
	// ModuleName for the wasm client
	ModuleName = "08-wasm"

	// StoreKey is the store key string for 08-wasm
	StoreKey = ModuleName
)

var (
	// CodeHashesKey is the key under which all code hashes are stored
	CodeHashesKey = collections.NewPrefix(0)
)
