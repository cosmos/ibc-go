package solomachine_test

import (
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	solomachine "github.com/cosmos/ibc-go/v3/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *SoloMachineTestSuite) TestVerifySignature() {
	cdc := suite.chainA.App.AppCodec()
	signBytes := []byte("sign bytes")

	singleSignature := suite.solomachine.GenerateSignature(signBytes)
	singleSigData, err := solomachine.UnmarshalSignatureData(cdc, singleSignature)
	suite.Require().NoError(err)

	multiSignature := suite.solomachineMulti.GenerateSignature(signBytes)
	multiSigData, err := solomachine.UnmarshalSignatureData(cdc, multiSignature)
	suite.Require().NoError(err)

	testCases := []struct {
		name      string
		publicKey cryptotypes.PubKey
		sigData   signing.SignatureData
		expPass   bool
	}{
		{
			"single signature with regular public key",
			suite.solomachine.PublicKey,
			singleSigData,
			true,
		},
		{
			"multi signature with multisig public key",
			suite.solomachineMulti.PublicKey,
			multiSigData,
			true,
		},
		{
			"single signature with multisig public key",
			suite.solomachineMulti.PublicKey,
			singleSigData,
			false,
		},
		{
			"multi signature with regular public key",
			suite.solomachine.PublicKey,
			multiSigData,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			err := solomachine.VerifySignature(tc.publicKey, signBytes, tc.sigData)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *SoloMachineTestSuite) TestClientStateSignBytes() {
	cdc := suite.chainA.App.AppCodec()

	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {
		// success
		path := sm.GetClientStatePath(counterpartyClientIdentifier)
		bz, err := solomachine.ClientStateSignBytes(cdc, sm.Sequence, sm.Time, sm.Diversifier, path, sm.ClientState())
		suite.Require().NoError(err)
		suite.Require().NotNil(bz)

		// nil client state
		bz, err = solomachine.ClientStateSignBytes(cdc, sm.Sequence, sm.Time, sm.Diversifier, path, nil)
		suite.Require().Error(err)
		suite.Require().Nil(bz)
	}
}

func (suite *SoloMachineTestSuite) TestConsensusStateSignBytes() {
	cdc := suite.chainA.App.AppCodec()

	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {
		// success
		path := sm.GetConsensusStatePath(counterpartyClientIdentifier, consensusHeight)
		bz, err := solomachine.ConsensusStateSignBytes(cdc, sm.Sequence, sm.Time, sm.Diversifier, path, sm.ConsensusState())
		suite.Require().NoError(err)
		suite.Require().NotNil(bz)

		// nil consensus state
		bz, err = solomachine.ConsensusStateSignBytes(cdc, sm.Sequence, sm.Time, sm.Diversifier, path, nil)
		suite.Require().Error(err)
		suite.Require().Nil(bz)
	}
}
