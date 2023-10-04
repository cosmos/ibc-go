package types_test

import (
	"time"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

func (suite *TypesTestSuite) TestConsensusStateValidateBasic() {
	testCases := []struct {
		name           string
		consensusState *types.ConsensusState
		expectPass     bool
	}{
		{
			"success",
			types.NewConsensusState([]byte("data"), uint64(time.Now().Unix())),
			true,
		},
		{
			"data is nil",
			types.NewConsensusState(nil, uint64(time.Now().Unix())),
			false,
		},
		{
			"data is empty",
			types.NewConsensusState([]byte{}, uint64(time.Now().Unix())),
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// check just to increase coverage
			suite.Require().Equal(exported.Wasm, tc.consensusState.ClientType())

			err := tc.consensusState.ValidateBasic()
			if tc.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
