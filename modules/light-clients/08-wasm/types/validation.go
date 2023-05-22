package types

const maxWasmSize = 3 * 1024 * 1024

var wasmIdent = []byte("\x00\x61\x73\x6D")

// ValidateWasmCode valides that the size of the wasm code is in the allowed range
// and that the contents are of a wasm binary.
func ValidateWasmCode(code []byte) error {
	if len(code) == 0 {
		return ErrWasmEmptyCode
	}
	if len(code) > maxWasmSize {
		return ErrWasmCodeTooLarge
	}

	return nil
}

// MaxWasmByteSize returns the maximum allowed number of bytes for wasm bytecode
func MaxWasmByteSize() uint64 {
	return maxWasmSize
}
