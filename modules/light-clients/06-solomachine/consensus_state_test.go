package solomachine_test

import (
	"errors"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (suite *SoloMachineTestSuite) TestConsensusState() {
	consensusState := suite.solomachine.ConsensusState()

	suite.Require().Equal(exported.Solomachine, consensusState.ClientType())
	suite.Require().Equal(suite.solomachine.Time, consensusState.GetTimestamp())
}

func (suite *SoloMachineTestSuite) TestConsensusStateValidateBasic() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

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
			suite.Run(tc.name, func() {
				err := tc.consensusState.ValidateBasic()

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
					suite.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}
