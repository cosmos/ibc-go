package solomachine_test

import (
	"errors"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
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
		expErr    error
	}{
		{
			"single signature with regular public key",
			suite.solomachine.PublicKey,
			singleSigData,
			nil,
		},
		{
			"multi signature with multisig public key",
			suite.solomachineMulti.PublicKey,
			multiSigData,
			nil,
		},
		{
			"single signature with multisig public key",
			suite.solomachineMulti.PublicKey,
			singleSigData,
			errors.New("invalid signature data type, expected *signing.MultiSignatureData, got *signing.MultiSignatureData: signature verification failed"),
		},
		{
			"multi signature with regular public key",
			suite.solomachine.PublicKey,
			multiSigData,
			errors.New("invalid signature data type, expected *signing.SingleSignatureData, got *signing.SingleSignatureData: signature verification failed"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := solomachine.VerifySignature(tc.publicKey, signBytes, tc.sigData)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}
