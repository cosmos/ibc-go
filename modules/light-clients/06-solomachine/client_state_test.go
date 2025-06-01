package solomachine_test

import (
	"bytes"
	"errors"

	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const (
	counterpartyClientIdentifier = "chainA"
	testConnectionID             = "connectionid"
	testChannelID                = "testchannelid"
	testPortID                   = "testportid"
)

func (suite *SoloMachineTestSuite) TestClientStateValidate() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {
		testCases := []struct {
			name        string
			clientState *solomachine.ClientState
			expErr      error
		}{
			{
				"valid client state",
				sm.ClientState(),
				nil,
			},
			{
				"empty ClientState",
				&solomachine.ClientState{},
				errors.New("sequence cannot be 0: light client is invalid"),
			},
			{
				"sequence is zero",
				solomachine.NewClientState(0, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, sm.Diversifier, sm.Time}),
				errors.New("sequence cannot be 0: light client is invalid"),
			},
			{
				"timestamp is zero",
				solomachine.NewClientState(1, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, sm.Diversifier, 0}),
				errors.New("timestamp cannot be 0: invalid consensus state"),
			},
			{
				"diversifier is blank",
				solomachine.NewClientState(1, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, "  ", 1}),
				errors.New("diversifier cannot contain only spaces: invalid consensus state"),
			},
			{
				"pubkey is empty",
				solomachine.NewClientState(1, &solomachine.ConsensusState{nil, sm.Diversifier, sm.Time}),
				errors.New("public key cannot be empty: invalid consensus state"),
			},
		}

		for _, tc := range testCases {
			suite.Run(tc.name, func() {
				err := tc.clientState.Validate()

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

func (suite *SoloMachineTestSuite) TestSignBytesMarshalling() {
	sm := suite.solomachine
	path := []byte("solomachine")
	signBytesNilData := solomachine.SignBytes{
		Sequence:    sm.GetHeight().GetRevisionHeight(),
		Timestamp:   sm.Time,
		Diversifier: sm.Diversifier,
		Path:        path,
		Data:        nil,
	}

	signBytesEmptyArray := solomachine.SignBytes{
		Sequence:    sm.GetHeight().GetRevisionHeight(),
		Timestamp:   sm.Time,
		Diversifier: sm.Diversifier,
		Path:        path,
		Data:        []byte{},
	}

	signBzNil, err := suite.chainA.Codec.Marshal(&signBytesNilData)
	suite.Require().NoError(err)

	signBzEmptyArray, err := suite.chainA.Codec.Marshal(&signBytesEmptyArray)
	suite.Require().NoError(err)

	suite.Require().True(bytes.Equal(signBzNil, signBzEmptyArray))
}
