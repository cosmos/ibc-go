package keeper_test

import (
	"testing"

	"github.com/cosmos/sandbox-ledger/x/tokenfactory/types"
	"github.com/stretchr/testify/require"

	_ "github.com/cosmos/sandbox-ledger/app"
)

func TestGenesis(t *testing.T) {
	params := types.DefaultParams()

	cases := []struct {
		name string
		gen  *types.GenesisState
	}{
		{
			name: "init: denoms and params",
			gen: &types.GenesisState{
				Params: params,
				FactoryDenoms: []types.GenesisDenom{
					{Denom: "token1", AuthorityMetadata: types.DenomAuthorityMetadata{Admin: creatorAddrA}},
					{Denom: "token2", AuthorityMetadata: types.DenomAuthorityMetadata{Admin: creatorAddrB}},
				},
			},
		},
		{
			name: "init empty",
			gen:  &types.GenesisState{Params: params, FactoryDenoms: []types.GenesisDenom{}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)
			k := wfapp.TokenFactoryKeeper

			require.NotNil(t, tc.gen)
			k.InitGenesis(ctx, *tc.gen)

			got := k.ExportGenesis(ctx)
			require.NotNil(t, got)

			require.Equal(t, tc.gen.Params, got.Params)

			wantDenoms := make(map[string]string)
			for _, d := range tc.gen.FactoryDenoms {
				wantDenoms[d.Denom] = d.AuthorityMetadata.Admin
			}
			gotDenoms := make(map[string]string)
			for _, d := range got.FactoryDenoms {
				gotDenoms[d.Denom] = d.AuthorityMetadata.Admin
			}
			require.Len(t, gotDenoms, len(wantDenoms))
			for denom, admin := range wantDenoms {
				require.Equal(t, admin, gotDenoms[denom])
			}
		})
	}
}
