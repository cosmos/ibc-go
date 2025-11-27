package types_test

import (
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
)

func (s *TypesTestSuite) TestConsensusStateValidateBasic() {
	testCases := []struct {
		name           string
		consensusState *types.ConsensusState
		expectPass     bool
	}{
		{
			"success",
			types.NewConsensusState([]byte("data")),
			true,
		},
		{
			"data is nil",
			types.NewConsensusState(nil),
			false,
		},
		{
			"data is empty",
			types.NewConsensusState([]byte{}),
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// check just to increase coverage
			s.Require().Equal(types.Wasm, tc.consensusState.ClientType())

			err := tc.consensusState.ValidateBasic()
			if tc.expectPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
