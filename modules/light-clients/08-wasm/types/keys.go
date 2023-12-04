package types

const (
	// ModuleName for the wasm client
	ModuleName = "08-wasm"

	// StoreKey is the store key string for 08-wasm
	StoreKey = ModuleName

	// Wasm is used to indicate that the light client is a on-chain wasm program
	Wasm string = ModuleName
)
