package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v11/modules/apps/prototypes/tokenfactory/types"
)

// InitGenesis initializes the tokenfactory module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(err)
	}

	for _, genDenom := range genState.FactoryDenoms {
		if err := k.setAuthorityMetadata(ctx, genDenom.Denom, genDenom.AuthorityMetadata); err != nil {
			panic(err)
		}

		if err := k.addDenomFromCreator(ctx, genDenom.AuthorityMetadata.Admin, genDenom.Denom); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the tokenfactory module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}

	genDenoms := make([]types.GenesisDenom, 0)

	if err := k.DenomAuthorityMetadataStore.Walk(ctx, nil, func(denom string, authMetadata types.DenomAuthorityMetadata) (bool, error) {
		genDenoms = append(genDenoms, types.GenesisDenom{
			Denom:             denom,
			AuthorityMetadata: authMetadata,
		})
		return false, nil
	}); err != nil {
		panic(err)
	}

	return &types.GenesisState{
		Params:        params,
		FactoryDenoms: genDenoms,
	}
}
