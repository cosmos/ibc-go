//go:build cgo && !nolink_libwasmvm

package types

import wasmvm "github.com/CosmWasm/wasmvm/v2"

var _ WasmEngine = (*wasmvm.VM)(nil)
