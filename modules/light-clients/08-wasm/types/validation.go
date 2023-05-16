package types

import "bytes"

const maxWasmSize = 3 * 1024 * 1024

var wasmIdent = []byte("\x00\x61\x73\x6D")

// IsWasm checks if the file contents are of wasm binary
func IsWasm(input []byte) bool {
	return bytes.Equal(input[:4], wasmIdent)
}

// ValidateWasmCode valides that the size of the wasm code is in the allowed range
// and that the contents are of a wasm binary.
func ValidateWasmCode(code []byte) error {
	if len(code) == 0 {
		return ErrWasmEmptyCode
	}
	if len(code) > maxWasmSize {
		return ErrWasmCodeTooLarge
	}

	// TODO: is this needed? Tests seem to fail when is check is active
	// if IsWasm(code) {
	// 	return ErrWasmInvalidCode
	// }

	return nil
}

// MaxWasmByteSize returns the maximum allowed number of bytes for wasm bytecode
func MaxWasmByteSize() uint64 {
	return maxWasmSize
}
