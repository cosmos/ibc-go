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
			name: "valid consensus state",
			consensusState: func() interface{ ValidateBasic() error } {
				return s.createConsensusState(uint64(time.Second.Nanoseconds()))
			},
			expErr: false,
		},
		{
			name: "zero timestamp",
			consensusState: func() interface{ ValidateBasic() error } {
				return s.createConsensusState(0)
			},
			expErr: true,
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
