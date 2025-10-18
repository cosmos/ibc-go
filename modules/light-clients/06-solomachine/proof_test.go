package solomachine_test

import (
	"errors"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
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
		expErr    error
	}{
		{
			"single signature with regular public key",
			s.solomachine.PublicKey,
			singleSigData,
			nil,
		},
		{
			"multi signature with multisig public key",
			s.solomachineMulti.PublicKey,
			multiSigData,
			nil,
		},
		{
			"single signature with multisig public key",
			s.solomachineMulti.PublicKey,
			singleSigData,
			errors.New("invalid signature data type, expected *signing.MultiSignatureData, got *signing.MultiSignatureData: signature verification failed"),
		},
		{
			"multi signature with regular public key",
			s.solomachine.PublicKey,
			multiSigData,
			errors.New("invalid signature data type, expected *signing.SingleSignatureData, got *signing.SingleSignatureData: signature verification failed"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := solomachine.VerifySignature(tc.publicKey, signBytes, tc.sigData)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}
