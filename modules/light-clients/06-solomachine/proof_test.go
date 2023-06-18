package solomachine_test

import (
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
)

func (s *SoloMachineTestSuite) TestVerifySignature() {
	cdc := s.chainA.App.AppCodec()
	signBytes := []byte("sign bytes")

	singleSignature := s.solomachine.GenerateSignature(signBytes)
	singleSigData, err := solomachine.UnmarshalSignatureData(cdc, singleSignature)
	s.Require().NoError(err)

	multiSignature := s.solomachineMulti.GenerateSignature(signBytes)
	multiSigData, err := solomachine.UnmarshalSignatureData(cdc, multiSignature)
	s.Require().NoError(err)

	testCases := []struct {
		name      string
		publicKey cryptotypes.PubKey
		sigData   signing.SignatureData
		expPass   bool
	}{
		{
			"single signature with regular public key",
			s.solomachine.PublicKey,
			singleSigData,
			true,
		},
		{
			"multi signature with multisig public key",
			s.solomachineMulti.PublicKey,
			multiSigData,
			true,
		},
		{
			"single signature with multisig public key",
			s.solomachineMulti.PublicKey,
			singleSigData,
			false,
		},
		{
			"multi signature with regular public key",
			s.solomachine.PublicKey,
			multiSigData,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			err := solomachine.VerifySignature(tc.publicKey, signBytes, tc.sigData)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
