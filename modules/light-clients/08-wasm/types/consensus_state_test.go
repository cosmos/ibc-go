package types_test

import (
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

func (suite *WasmTestSuite) TestConsensusStateValidateBasic() {
	testCases := []struct {
		name           string
		consensusState *types.ConsensusState
		expectPass     bool
	}{
		{
			"success",
			&types.ConsensusState{
				Timestamp: uint64(suite.now.Unix()),
				Data:      []byte("data"),
			},
			true,
		},
		{
			"timestamp is zero",
			&types.ConsensusState{
				Timestamp: 0,
				Data:      []byte("data"),
			},
			false,
		},
		{
			"data is nil",
			&types.ConsensusState{
				Timestamp: uint64(suite.now.Unix()),
				Data:      nil,
			},
			false,
		},
		{
			"data is empty",
			&types.ConsensusState{
				Timestamp: uint64(suite.now.Unix()),
				Data:      []byte(""),
			},
			false,
		},
	}

	for i, tc := range testCases {
		tc := tc

		// check just to increase coverage
		suite.Require().Equal(exported.Wasm, tc.consensusState.ClientType())

		err := tc.consensusState.ValidateBasic()
		if tc.expectPass {
			suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
		} else {
			suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}
