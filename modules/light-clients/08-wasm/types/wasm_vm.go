// SPDX-License-Identifier: Apache-2.0

//go:build cgo && !nolink_libwasmvm

package types

import wasmvm "github.com/CosmWasm/wasmvm/v3"

var _ WasmEngine = (*wasmvm.VM)(nil)
