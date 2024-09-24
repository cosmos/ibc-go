package types_test

import (
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

func (suite *TypesTestSuite) TestValidateGenesis() {
	testCases := []struct {
		name     string
		genState *types.GenesisState
		expPass  bool
	}{
		{
			"valid genesis",
			&types.GenesisState{
				Contracts: []types.Contract{{CodeBytes: []byte{1}}},
			},
			true,
		},
		{
			"invalid genesis",
			&types.GenesisState{
				Contracts: []types.Contract{{CodeBytes: []byte{}}},
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.genState.Validate()
		if tc.expPass {
			suite.Require().NoError(err)
		} else {
			suite.Require().Error(err)
		}
	}
}
