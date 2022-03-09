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
			malleateMisbehaviour func(duplicateSigHeader *types.DuplicateSignatureHeader)
			expPass              bool
		}{
			{
				"valid misbehaviour",
				func(*types.DuplicateSignatureHeader) {},
				true,
			},
			{
				"invalid client ID",
				func(duplicateSigHeader *types.DuplicateSignatureHeader) {
					duplicateSigHeader.ClientId = "(badclientid)"
				},
				false,
			},
			{
				"sequence is zero",
				func(duplicateSigHeader *types.DuplicateSignatureHeader) {
					duplicateSigHeader.Sequence = 0
				},
				false,
			},
			{
				"signature one sig is empty",
				func(duplicateSigHeader *types.DuplicateSignatureHeader) {
					duplicateSigHeader.SignatureOne.Signature = []byte{}
				},
				false,
			},
			{
				"signature two sig is empty",
				func(duplicateSigHeader *types.DuplicateSignatureHeader) {
					duplicateSigHeader.SignatureTwo.Signature = []byte{}
				},
				false,
			},
			{
				"signature one data is empty",
				func(duplicateSigHeader *types.DuplicateSignatureHeader) {
					duplicateSigHeader.SignatureOne.Data = nil
				},
				false,
			},
			{
				"signature two data is empty",
				func(duplicateSigHeader *types.DuplicateSignatureHeader) {
					duplicateSigHeader.SignatureTwo.Data = []byte{}
				},
				false,
			},
			{
				"signatures are identical",
				func(duplicateSigHeader *types.DuplicateSignatureHeader) {
					duplicateSigHeader.SignatureTwo.Signature = duplicateSigHeader.SignatureOne.Signature
				},
				false,
			},
			{
				"data signed is identical",
				func(duplicateSigHeader *types.DuplicateSignatureHeader) {
					duplicateSigHeader.SignatureTwo.Data = duplicateSigHeader.SignatureOne.Data
				},
				false,
			},
			{
				"data type for SignatureOne is unspecified",
				func(misbehaviour *types.DuplicateSignatureHeader) {
					misbehaviour.SignatureOne.DataType = types.UNSPECIFIED
				}, false,
			},
			{
				"data type for SignatureTwo is unspecified",
				func(misbehaviour *types.DuplicateSignatureHeader) {
					misbehaviour.SignatureTwo.DataType = types.UNSPECIFIED
				}, false,
			},
			{
				"timestamp for SignatureOne is zero",
				func(misbehaviour *types.DuplicateSignatureHeader) {
					misbehaviour.SignatureOne.Timestamp = 0
				}, false,
			},
			{
				"timestamp for SignatureTwo is zero",
				func(misbehaviour *types.DuplicateSignatureHeader) {
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
