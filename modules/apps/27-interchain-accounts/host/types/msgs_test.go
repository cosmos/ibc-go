package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

type TypesTestSuite struct {
	testifysuite.Suite

	chainA *ibctesting.TestChain
}

func TestMsgUpdateParamsValidateBasic(t *testing.T) {
	testCases := []struct {
		name    string
		msg     *types.MsgUpdateParams
		expPass bool
	}{
		{
			"success: valid signer address",
			types.NewMsgUpdateParams(sdk.AccAddress(ibctesting.TestAccAddress).String(), types.DefaultParams()),
			true,
		},
		{
			"failure: invalid signer address",
			types.NewMsgUpdateParams("signer", types.DefaultParams()),
			false,
		},
		{
			"failure: invalid allowed message",
			types.NewMsgUpdateParams("signer", types.Params{
				AllowMessages: []string{""},
			}),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		err := tc.msg.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

func (suite *TypesTestSuite) TestMsgUpdateParamsGetSigners() {
	testCases := []struct {
		name    string
		address sdk.AccAddress
		expPass bool
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), true},
		{"failure: nil address", nil, false},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			msg := types.NewMsgUpdateParams(tc.address.String(), types.DefaultParams())
			signers, _, err := suite.chainA.Codec.GetMsgV1Signers(msg)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.address.Bytes(), signers[0])
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
