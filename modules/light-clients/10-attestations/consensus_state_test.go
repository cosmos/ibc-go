package attestations_test

import (
	"time"
)

func (s *AttestationsTestSuite) TestConsensusStateValidateBasic() {
	testCases := []struct {
		name           string
		consensusState func() interface{ ValidateBasic() error }
		expErr         bool
	}{
		{
			"valid consensus state",
			func() interface{ ValidateBasic() error } {
				return s.createConsensusState(uint64(time.Second.Nanoseconds()))
			},
			false,
		},
		{
			"zero timestamp",
			func() interface{ ValidateBasic() error } {
				return s.createConsensusState(0)
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			consensusState := tc.consensusState()
			err := consensusState.ValidateBasic()
			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
