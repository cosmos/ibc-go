package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	ica "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts"
	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func TestMsgUpdateParamsValidateBasic(t *testing.T) {
	testCases := []struct {
		name   string
		msg    *types.MsgUpdateParams
		expErr error
	}{
		{
			"success: valid signer address",
			types.NewMsgUpdateParams(sdk.AccAddress(ibctesting.TestAccAddress).String(), types.DefaultParams()),
			nil,
		},
		{
			"failure: invalid signer address",
			types.NewMsgUpdateParams("signer", types.DefaultParams()),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: invalid allowed message",
			types.NewMsgUpdateParams("signer", types.Params{
				AllowMessages: []string{""},
			}),
			ibcerrors.ErrInvalidAddress,
		},
	}

	for _, tc := range testCases {
		tc := tc

		err := tc.msg.ValidateBasic()
		if tc.expErr == nil {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, tc.expErr)
		}
	}
}

func TestMsgUpdateParamsGetSigners(t *testing.T) {
	testCases := []struct {
		name    string
		address sdk.AccAddress
		errMsg  string
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), ""},
		{"failure: nil address", nil, "empty address string is not allowed"},
	}

	for _, tc := range testCases {
		tc := tc

		msg := types.NewMsgUpdateParams(tc.address.String(), types.DefaultParams())
		encodingCfg := moduletestutil.MakeTestEncodingConfig(testutil.CodecOptions{}, ica.AppModule{})
		signers, _, err := encodingCfg.Codec.GetMsgSigners(msg)
		if tc.errMsg == "" {
			require.NoError(t, err)
			require.Equal(t, tc.address.Bytes(), signers[0])
		} else {
			require.ErrorContains(t, err, tc.errMsg)
		}
	}
}

func TestMsgModuleQuerySafeValidateBasic(t *testing.T) {
	queryRequest := types.QueryRequest{
		Path: "/cosmos.bank.v1beta1.Query/Balance",
		Data: []byte{},
	}

	testCases := []struct {
		name   string
		msg    *types.MsgModuleQuerySafe
		expErr error
	}{
		{
			"success: valid signer address",
			types.NewMsgModuleQuerySafe(sdk.AccAddress(ibctesting.TestAccAddress).String(), []types.QueryRequest{queryRequest}),
			nil,
		},
		{
			"failure: invalid signer address",
			types.NewMsgModuleQuerySafe("signer", []types.QueryRequest{queryRequest}),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: empty query requests",
			types.NewMsgModuleQuerySafe(sdk.AccAddress(ibctesting.TestAccAddress).String(), []types.QueryRequest{}),
			ibcerrors.ErrInvalidRequest,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()

			if tc.expErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expErr)
			}
		})
	}
}

func TestMsgModuleQuerySafeGetSigners(t *testing.T) {
	testCases := []struct {
		name    string
		address sdk.AccAddress
		errMsg  string
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), ""},
		{"failure: nil address", nil, "empty address string is not allowed"},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			msg := types.NewMsgModuleQuerySafe(tc.address.String(), []types.QueryRequest{})
			encodingCfg := moduletestutil.MakeTestEncodingConfig(testutil.CodecOptions{}, ica.AppModule{})
			signers, _, err := encodingCfg.Codec.GetMsgSigners(msg)
			if tc.errMsg == "" {
				require.NoError(t, err)
				require.Equal(t, tc.address.Bytes(), signers[0])
			} else {
				require.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}
