package solomachine_test

import (
	"errors"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *SoloMachineTestSuite) TestMisbehaviour() {
	misbehaviour := s.solomachine.CreateMisbehaviour()

	s.Require().Equal(exported.Solomachine, misbehaviour.ClientType())
}

func (s *SoloMachineTestSuite) TestMisbehaviourValidateBasic() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{s.solomachine, s.solomachineMulti} {
		testCases := []struct {
			name                 string
			malleateMisbehaviour func(misbehaviour *solomachine.Misbehaviour)
			expErr               error
		}{
			{
				"valid misbehaviour",
				func(*solomachine.Misbehaviour) {},
				nil,
			},
			{
				"sequence is zero",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.Sequence = 0
				},
				errors.New("sequence cannot be 0: invalid light client misbehaviour"),
			},
			{
				"signature one sig is empty",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureOne.Signature = []byte{}
				},
				errors.New("signature one failed basic validation: signature cannot be empty: invalid signature and data"),
			},
			{
				"signature two sig is empty",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Signature = []byte{}
				},
				errors.New("signature two failed basic validation: signature cannot be empty: invalid signature and data"),
			},
			{
				"signature one data is empty",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureOne.Data = nil
				},
				errors.New("signature one failed basic validation: data for signature cannot be empty: invalid signature and data"),
			},
			{
				"signature two data is empty",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Data = []byte{}
				},
				errors.New("signature two failed basic validation: data for signature cannot be empty: invalid signature and data"),
			},
			{
				"signatures are identical",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Signature = misbehaviour.SignatureOne.Signature
				},
				errors.New("misbehaviour signatures cannot be equal: invalid light client misbehaviour"),
			},
			{
				"data signed is identical but path differs",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Data = misbehaviour.SignatureOne.Data
				},
				nil,
			},
			{
				"data signed and path are identical",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Path = misbehaviour.SignatureOne.Path
					misbehaviour.SignatureTwo.Data = misbehaviour.SignatureOne.Data
				},
				errors.New("misbehaviour signature data must be signed over different messages: invalid light client misbehaviour"),
			},
			{
				"data path for SignatureOne is unspecified",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureOne.Path = []byte{}
				},
				errors.New("signature one failed basic validation: path for signature cannot be empty: invalid signature and data"),
			},
			{
				"data path for SignatureTwo is unspecified",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Path = []byte{}
				},
				errors.New("signature two failed basic validation: path for signature cannot be empty: invalid signature and data"),
			},
			{
				"timestamp for SignatureOne is zero",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureOne.Timestamp = 0
				},
				errors.New("signature one failed basic validation: timestamp cannot be 0: invalid signature and data"),
			},
			{
				"timestamp for SignatureTwo is zero",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Timestamp = 0
				},
				errors.New("signature two failed basic validation: timestamp cannot be 0: invalid signature and data"),
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				misbehaviour := sm.CreateMisbehaviour()
				tc.malleateMisbehaviour(misbehaviour)

				err := misbehaviour.ValidateBasic()

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
