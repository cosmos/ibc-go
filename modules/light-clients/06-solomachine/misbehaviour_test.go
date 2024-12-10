package solomachine_test

import (
	"errors"

	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v9/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *SoloMachineTestSuite) TestMisbehaviour() {
	misbehaviour := suite.solomachine.CreateMisbehaviour()

	suite.Require().Equal(exported.Solomachine, misbehaviour.ClientType())
}

func (suite *SoloMachineTestSuite) TestMisbehaviourValidateBasic() {
	// test singlesig and multisig public keys
	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

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
				errors.New("the sequence number is zero, which is invalid"),
			},
			{
				"signature one sig is empty",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureOne.Signature = []byte{}
				},
				errors.New("the first signature is empty, which is invalid"),
			},
			{
				"signature two sig is empty",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Signature = []byte{}
				},
				errors.New("the second signature is empty, which is invalid"),
			},
			{
				"signature one data is empty",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureOne.Data = nil
				},
				errors.New("the data for the first signature is empty, which is invalid"),
			},
			{
				"signature two data is empty",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Data = []byte{}
				},
				errors.New("the data for the second signature cannot be empty"),
			},
			{
				"signatures are identical",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Signature = misbehaviour.SignatureOne.Signature
				},
				errors.New("the second signature is identical to the first signature, which is invalid"),
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
				errors.New("the second signature's data and path are identical to the first, which is invalid"),
			},
			{
				"data path for SignatureOne is unspecified",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureOne.Path = []byte{}
				},
				errors.New("the data path for SignatureOne is empty, which is invalid"),
			},
			{
				"data path for SignatureTwo is unspecified",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Path = []byte{}
				},
				errors.New("the data path for SignatureTwo is empty, which is invalid"),
			},
			{
				"timestamp for SignatureOne is zero",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureOne.Timestamp = 0
				},
				errors.New("the timestamp for SignatureOne cannot be zero; it must be a valid, positive timestamp"),
			},
			{
				"timestamp for SignatureTwo is zero",
				func(misbehaviour *solomachine.Misbehaviour) {
					misbehaviour.SignatureTwo.Timestamp = 0
				},
				errors.New("the timestamp for SignatureTwo is zero, which is invalid"),
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				misbehaviour := sm.CreateMisbehaviour()
				tc.malleateMisbehaviour(misbehaviour)

				err := misbehaviour.ValidateBasic()

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}
