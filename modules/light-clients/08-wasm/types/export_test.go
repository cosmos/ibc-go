package types

/*
	This file is to allow for unexported functions and fields to be accessible to the testing package.
*/

// MaxWasmSize is the maximum size of a wasm code in bytes.
const MaxWasmSize = maxWasmSize

// these fields are exported aliases for the payload fields passed to the wasm vm.
// these are used to specify callback functions to handle specific queries in the mock vm.
type (
	// CallbackFn types
	QueryFn = queryFn
	SudoFn  = sudoFn
)
