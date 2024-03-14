//go:build !ibcwasm_novm 

package ibcwasm

import wasmvm "github.com/CosmWasm/wasmvm"

var _ WasmEngine = (*wasmvm.VM)(nil)
