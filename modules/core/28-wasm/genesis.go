package wasm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/keeper"
	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/types"
)

// InitGenesis initializes the ibc channel submodule's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, gs types.GenesisState) {
	for _, wasm := range gs.WasmLightClients {
		if err := k.SetWasmLightClient(ctx, wasm); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the ibc channel submodule's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) types.GenesisState {
	return types.GenesisState{k.GetWasmLightClients(ctx)}
}
