package keeper

import (
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

// InitGenesis initializes the 08-wasm module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) error {
	for _, contract := range gs.Contracts {
		_, err := k.storeWasmCode(ctx, contract.CodeBytes)
		if err != nil {
			return err
		}
	}
	return nil
}

// ExportGenesis returns the 08-wasm module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte(types.KeyCodeHashPrefix))
	defer iterator.Close()

	var genesisState types.GenesisState
	for ; iterator.Valid(); iterator.Next() {
		genesisState.Contracts = append(genesisState.Contracts, types.Contract{
			CodeBytes: iterator.Value(),
		})
	}
	return genesisState
}
