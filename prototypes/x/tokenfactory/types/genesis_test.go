package types_test

import (
	"testing"

	"github.com/cosmos/sandbox-ledger/testutil"
	"github.com/cosmos/sandbox-ledger/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func TestValidateGenesis(t *testing.T) {
	testutil.SafeSetAddressPrefixes()
	tests := []struct {
		name      string
		genState  types.GenesisState
		expectErr bool
	}{
		{
			name:      "default genesis",
			genState:  *types.DefaultGenesis(),
			expectErr: false,
		},
		{
			name: "valid genesis with denoms",
			genState: types.GenesisState{
				Params: types.DefaultParams(),
				FactoryDenoms: []types.GenesisDenom{
					{
						Denom: "uwfdeposit",
						AuthorityMetadata: types.DenomAuthorityMetadata{
							Admin: "cosmos1nsh9vj9znccakn6xwlhlwx92acdt79yeqrkz4y",
						},
					},
					{
						Denom: "uwfdeposit2",
						AuthorityMetadata: types.DenomAuthorityMetadata{
							Admin: "cosmos1uu635yk0hz3cvrypnryrggltrjq7975jrmeg97",
						},
					},
				},
			},
			expectErr: false,
		},
		{
			name: "invalid denom in genesis",
			genState: types.GenesisState{
				Params: types.DefaultParams(),
				FactoryDenoms: []types.GenesisDenom{
					{
						Denom: "invalid-denom",
						AuthorityMetadata: types.DenomAuthorityMetadata{
							Admin: "wf1creator",
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "empty admin in genesis",
			genState: types.GenesisState{
				Params: types.DefaultParams(),
				FactoryDenoms: []types.GenesisDenom{
					{
						Denom: "token1",
						AuthorityMetadata: types.DenomAuthorityMetadata{
							Admin: "",
						},
					},
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateGenesis(tt.genState)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestParamsValidate(t *testing.T) {
	testutil.SafeSetAddressPrefixes()
	require.NoError(t, types.DefaultParams().Validate())
	require.NoError(t, types.Params{}.Validate())
}
