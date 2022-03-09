package types_test

import (
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/06-solomachine/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *SoloMachineTestSuite) TestMisbehaviour() {
	misbehaviour := suite.solomachine.CreateMisbehaviour()

	suite.Require().Equal(exported.Solomachine, misbehaviour.ClientType())
	suite.Require().Equal(suite.solomachine.ClientID, misbehaviour.GetClientID())
}

func (suite *SoloMachineTestSuite) TestMisbehaviourValidateBasic() {
	// test singlesig and multisig public keys
	for _, solomachine := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {

		testCases := []struct {
			name                 string
			malleateMisbehaviour func(duplicateSigHeader *types.DuplicateSignatures)
			expPass              bool
		}{
			{
				"valid misbehaviour",
				func(*types.DuplicateSignatures) {},
				true,
			},
			{
				"invalid client ID",
				func(duplicateSigHeader *types.DuplicateSignatures) {
					duplicateSigHeader.ClientId = "(badclientid)"
				},
				false,
			},
			{
				"sequence is zero",
				func(duplicateSigHeader *types.DuplicateSignatures) {
					duplicateSigHeader.Sequence = 0
				},
				false,
			},
			{
				"signature one sig is empty",
				func(duplicateSigHeader *types.DuplicateSignatures) {
					duplicateSigHeader.SignatureOne.Signature = []byte{}
				},
				false,
			},
			{
				"signature two sig is empty",
				func(duplicateSigHeader *types.DuplicateSignatures) {
					duplicateSigHeader.SignatureTwo.Signature = []byte{}
				},
				false,
			},
			{
				"signature one data is empty",
				func(duplicateSigHeader *types.DuplicateSignatures) {
					duplicateSigHeader.SignatureOne.Data = nil
				},
				false,
			},
			{
				"signature two data is empty",
				func(duplicateSigHeader *types.DuplicateSignatures) {
					duplicateSigHeader.SignatureTwo.Data = []byte{}
				},
				false,
			},
			{
				"signatures are identical",
				func(duplicateSigHeader *types.DuplicateSignatures) {
					duplicateSigHeader.SignatureTwo.Signature = duplicateSigHeader.SignatureOne.Signature
				},
				false,
			},
			{
				"data signed is identical",
				func(duplicateSigHeader *types.DuplicateSignatures) {
					duplicateSigHeader.SignatureTwo.Data = duplicateSigHeader.SignatureOne.Data
				},
				false,
			},
			{
				"data type for SignatureOne is unspecified",
				func(misbehaviour *types.DuplicateSignatures) {
					misbehaviour.SignatureOne.DataType = types.UNSPECIFIED
				}, false,
			},
			{
				"data type for SignatureTwo is unspecified",
				func(misbehaviour *types.DuplicateSignatures) {
					misbehaviour.SignatureTwo.DataType = types.UNSPECIFIED
				}, false,
			},
			{
				"timestamp for SignatureOne is zero",
				func(misbehaviour *types.DuplicateSignatures) {
					misbehaviour.SignatureOne.Timestamp = 0
				}, false,
			},
			{
				"timestamp for SignatureTwo is zero",
				func(misbehaviour *types.DuplicateSignatures) {
					misbehaviour.SignatureTwo.Timestamp = 0
				}, false,
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {

				misbehaviour := solomachine.CreateMisbehaviour()
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
