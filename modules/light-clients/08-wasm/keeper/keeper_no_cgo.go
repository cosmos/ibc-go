//go:build !cgo

package keeper

import (
	storetypes "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

// NewKeeperWithVM creates a new Keeper instance with the provided Wasm VM.
// This constructor function is meant to be used when the chain uses x/wasm
// and the same Wasm VM instance should be shared with it.
func NewKeeperWithVM(
	_ codec.BinaryCodec,
	_ storetypes.KVStoreService,
	_ types.ClientKeeper,
	_ string,
	_ ibcwasm.WasmEngine,
	_ ibcwasm.QueryRouter,
	_ ...Option,
) Keeper {
	panic("not implemented, please build with cgo enabled")
}

// NewKeeperWithConfig creates a new Keeper instance with the provided Wasm configuration.
// This constructor function is meant to be used when the chain does not use x/wasm
// and a Wasm VM needs to be instantiated using the provided parameters.
func NewKeeperWithConfig(
	_ codec.BinaryCodec,
	_ storetypes.KVStoreService,
	_ types.ClientKeeper,
	_ string,
	_ types.WasmConfig,
	_ ibcwasm.QueryRouter,
	_ ...Option,
) Keeper {
	panic("not implemented, please build with cgo enabled")
}
