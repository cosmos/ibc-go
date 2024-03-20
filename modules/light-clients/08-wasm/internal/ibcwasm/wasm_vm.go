//go:build !ibcwasm_novm

package ibcwasm

import wasmvm "github.com/CosmWasm/wasmvm/v2"

var _ WasmEngine = (*wasmvm.VM)(nil)
