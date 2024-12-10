package solomachine_test

import (
	"bytes"
	"errors"

	solomachine "github.com/cosmos/ibc-go/v9/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
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
				errors.New("the ClientState is empty, which is invalid"),
			},
			{
				"sequence is zero",
				solomachine.NewClientState(0, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, sm.Diversifier, sm.Time}),
				errors.New("the sequence number is zero, which is not valid for a new client state"),
			},
			{
				"timestamp is zero",
				solomachine.NewClientState(1, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, sm.Diversifier, 0}),
				errors.New("the timestamp must be a valid time greater than zero"),
			},
			{
				"diversifier is blank",
				solomachine.NewClientState(1, &solomachine.ConsensusState{sm.ConsensusState().PublicKey, "  ", 1}),
				errors.New("the diversifier is blank - contains only whitespace, which is invalid"),
			},
			{
				"pubkey is empty",
				solomachine.NewClientState(1, &solomachine.ConsensusState{nil, sm.Diversifier, sm.Time}),
				errors.New("the public key cannot be empty"),
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				err := tc.clientState.Validate()

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
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
