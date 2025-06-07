package types_test

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
)

func (s *TypesTestSuite) TestValidateGenesis() {
	testCases := []struct {
		name     string
		genState *types.GenesisState
		expErr   error
	}{
		{
			"valid genesis",
			&types.GenesisState{
				Contracts: []types.Contract{{CodeBytes: []byte{1}}},
			},
			nil,
		},
		{
			"invalid genesis",
			&types.GenesisState{
				Contracts: []types.Contract{{CodeBytes: []byte{}}},
			},
			errorsmod.Wrap(types.ErrWasmEmptyCode, "wasm bytecode validation failed"),
		},
	}

	for _, tc := range testCases {
		err := tc.genState.Validate()
		if tc.expErr == nil {
			s.Require().NoError(err)
		} else {
			s.Require().Error(err)
			s.Require().ErrorIs(err, tc.expErr)
		}
	}
}
