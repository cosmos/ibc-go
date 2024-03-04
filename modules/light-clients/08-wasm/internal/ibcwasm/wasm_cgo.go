//go:build cgo

package ibcwasm

import wasmvm "github.com/CosmWasm/wasmvm"

var _ WasmEngine = (*wasmvm.VM)(nil)
