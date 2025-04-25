package types_test

import (
	"errors"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// TestMustMarshalProtoJSON tests that the memo field is only emitted (marshalled) if it is populated
func (suite *TypesTestSuite) TestMustMarshalProtoJSON() {
	memo := "memo"
	packetData := types.NewFungibleTokenPacketData(sdk.DefaultBondDenom, "1", suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), memo)

	bz := packetData.GetBytes()
	exists := strings.Contains(string(bz), memo)
	suite.Require().True(exists)

	packetData.Memo = ""

	bz = packetData.GetBytes()
	exists = strings.Contains(string(bz), memo)
	suite.Require().False(exists)
}

func (suite *TypesTestSuite) TestCodecTypeRegistration() {
	testCases := []struct {
		name    string
		typeURL string
		expErr  error
	}{
		{
			"success: MsgTransfer",
			sdk.MsgTypeURL(&types.MsgTransfer{}),
			nil,
		},
		{
			"success: MsgUpdateParams",
			sdk.MsgTypeURL(&types.MsgUpdateParams{}),
			nil,
		},
		{
			"success: TransferAuthorization",
			sdk.MsgTypeURL(&types.TransferAuthorization{}),
			nil,
		},
		{
			"type not registered on codec",
			"ibc.invalid.MsgTypeURL",
			errors.New("unable to resolve type URL"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			encodingCfg := moduletestutil.MakeTestEncodingConfig(transfer.AppModuleBasic{})
			msg, err := encodingCfg.Codec.InterfaceRegistry().Resolve(tc.typeURL)

			if tc.expErr == nil {
				suite.Require().NotNil(msg)
				suite.Require().NoError(err)
			} else {
				suite.Require().Nil(msg)
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expErr)
			}
		})
	}
}
