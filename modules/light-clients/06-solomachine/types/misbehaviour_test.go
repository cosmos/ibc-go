package types_test

import (
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/06-solomachine/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *SoloMachineTestSuite) TestMisbehaviour() {
	misbehaviour := suite.solomachine.CreateConflictingSignaturesHeader()

	suite.Require().Equal(exported.Solomachine, misbehaviour.ClientType())
	suite.Require().Equal(suite.solomachine.ClientID, misbehaviour.GetClientID())
}

func (suite *SoloMachineTestSuite) TestMisbehaviourValidateBasic() {
	// test singlesig and multisig public keys
	for _, solomachine := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		testCases := []struct {
			name                 string
			malleateMisbehaviour func(misbehaviour *types.ConflictingSignaturesHeader)
			expPass              bool
		}{
			{
				"valid misbehaviour",
				func(*types.ConflictingSignaturesHeader) {},
				true,
			},
			{
				"invalid client ID",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.ClientId = "(badclientid)"
				},
				false,
			},
			{
				"sequence is zero",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.Sequence = 0
				},
				false,
			},
			{
				"signature one sig is empty",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.SignatureOne.Signature = []byte{}
				},
				false,
			},
			{
				"signature two sig is empty",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.SignatureTwo.Signature = []byte{}
				},
				false,
			},
			{
				"signature one data is empty",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.SignatureOne.Data = nil
				},
				false,
			},
			{
				"signature two data is empty",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.SignatureTwo.Data = []byte{}
				},
				false,
			},
			{
				"signatures are identical",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.SignatureTwo.Signature = misbehaviour.SignatureOne.Signature
				},
				false,
			},
			{
				"data signed is identical",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.SignatureTwo.Data = misbehaviour.SignatureOne.Data
				},
				false,
			},
			{
				"data type for SignatureOne is unspecified",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.SignatureOne.DataType = types.UNSPECIFIED
				}, false,
			},
			{
				"data type for SignatureTwo is unspecified",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.SignatureTwo.DataType = types.UNSPECIFIED
				}, false,
			},
			{
				"timestamp for SignatureOne is zero",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.SignatureOne.Timestamp = 0
				}, false,
			},
			{
				"timestamp for SignatureTwo is zero",
				func(misbehaviour *types.ConflictingSignaturesHeader) {
					misbehaviour.SignatureTwo.Timestamp = 0
				}, false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {

				misbehaviour := solomachine.CreateConflictingSignaturesHeader()
				tc.malleateMisbehaviour(misbehaviour)

				err := misbehaviour.ValidateBasic()

				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}
