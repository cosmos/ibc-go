package solomachine_test

import (
	"errors"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *SoloMachineTestSuite) TestConsensusState() {
	consensusState := s.solomachine.ConsensusState()

	s.Require().Equal(exported.Solomachine, consensusState.ClientType())
	s.Require().Equal(s.solomachine.Time, consensusState.GetTimestamp())
}

func (s *SoloMachineTestSuite) TestConsensusStateValidateBasic() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
		testCases := []struct {
			name           string
			consensusState *solomachine.ConsensusState
			expErr         error
		}{
			{
				"valid consensus state",
				sm.ConsensusState(),
				nil,
			},
			{
				"timestamp is zero",
				&solomachine.ConsensusState{
					PublicKey:   sm.ConsensusState().PublicKey,
					Timestamp:   0,
					Diversifier: sm.Diversifier,
				},
				errors.New("timestamp cannot be 0: invalid consensus state"),
			},
			{
				"diversifier is blank",
				&solomachine.ConsensusState{
					PublicKey:   sm.ConsensusState().PublicKey,
					Timestamp:   sm.Time,
					Diversifier: " ",
				},
				errors.New("diversifier cannot contain only spaces: invalid consensus state"),
			},
			{
				"pubkey is nil",
				&solomachine.ConsensusState{
					Timestamp:   sm.Time,
					Diversifier: sm.Diversifier,
					PublicKey:   nil,
				},
				errors.New("public key cannot be empty: invalid consensus state"),
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				err := tc.consensusState.ValidateBasic()

				if tc.expErr == nil {
					s.Require().NoError(err)
				} else {
					s.Require().Error(err)
					s.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}
