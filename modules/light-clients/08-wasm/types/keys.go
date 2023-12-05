package types

const (
	// ModuleName for the wasm client
	ModuleName = "08-wasm"

	// StoreKey is the store key string for 08-wasm
	StoreKey = ModuleName

<<<<<<< HEAD
	// KeyChecksums is the key under which all checksums are stored
	KeyChecksums = "checksums"
=======
	// Wasm is the client type for IBC light clients created using 08-wasm
	Wasm = ModuleName
>>>>>>> e3ab9bec (fix: remove 08-wasm from 02-client exported (#5306))
)
