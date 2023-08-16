package keeper

import (
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

// ExportGenesis returns the 08-wasm module's exported genesis. This includes the code
// for all contracts previously stored.
func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	codeHashes := types.GetCodeHashes(ctx, k.cdc)

	// Grab code from wasmVM and add to genesis state.
	var genesisState types.GenesisState
	for _, codeHash := range codeHashes.CodeHashes {
		code, err := k.wasmVM.GetCode(codeHash)
		if err != nil {
			panic(err)
		}
		genesisState.Contracts = append(genesisState.Contracts, types.Contract{
			CodeBytes: code,
		})
	}

	return genesisState
}
