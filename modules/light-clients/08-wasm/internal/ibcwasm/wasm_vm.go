//go:build cgo && !nolink_libwasmvm

package ibcwasm

import wasmvm "github.com/CosmWasm/wasmvm"

var _ WasmEngine = (*wasmvm.VM)(nil)
