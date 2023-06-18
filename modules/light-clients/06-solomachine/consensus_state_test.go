package solomachine_test

import (
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
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
			expPass        bool
		}{
			{
				"valid consensus state",
				sm.ConsensusState(),
				true,
			},
			{
				"timestamp is zero",
				&solomachine.ConsensusState{
					PublicKey:   sm.ConsensusState().PublicKey,
					Timestamp:   0,
					Diversifier: sm.Diversifier,
				},
				false,
			},
			{
				"diversifier is blank",
				&solomachine.ConsensusState{
					PublicKey:   sm.ConsensusState().PublicKey,
					Timestamp:   sm.Time,
					Diversifier: " ",
				},
				false,
			},
			{
				"pubkey is nil",
				&solomachine.ConsensusState{
					Timestamp:   sm.Time,
					Diversifier: sm.Diversifier,
					PublicKey:   nil,
				},
				false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			s.Run(tc.name, func() {
				err := tc.consensusState.ValidateBasic()

				if tc.expPass {
					s.Require().NoError(err)
				} else {
					s.Require().Error(err)
				}
			})
		}
	}
}
