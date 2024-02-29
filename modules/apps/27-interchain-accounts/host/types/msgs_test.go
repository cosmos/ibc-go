package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

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

func TestMsgUpdateParamsGetSigners(t *testing.T) {
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

		msg := types.NewMsgUpdateParams(tc.address.String(), types.DefaultParams())
		encodingCfg := moduletestutil.MakeTestEncodingConfig(ica.AppModuleBasic{})
		signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)
		if tc.expPass {
			require.NoError(t, err)
			require.Equal(t, tc.address.Bytes(), signers[0])
		} else {
			require.Error(t, err)
		}
	}
}
