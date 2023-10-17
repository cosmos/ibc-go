package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestMsgStoreCode_ValidateBasic(t *testing.T) {
	signer := sdk.AccAddress(ibctesting.TestAccAddress).String()
	testCases := []struct {
		name   string
		msg    *types.MsgStoreCode
		expErr error
	}{
		{
			"success: valid signer address, valid length code",
			types.NewMsgStoreCode(signer, wasmtesting.Code),
			nil,
		},
		{
			"failure: code is empty",
			types.NewMsgStoreCode(signer, []byte("")),
			types.ErrWasmEmptyCode,
		},
		{
			"failure: code is too large",
			types.NewMsgStoreCode(signer, make([]byte, types.MaxWasmSize+1)),
			types.ErrWasmCodeTooLarge,
		},
		{
			"failure: signer is invalid",
			types.NewMsgStoreCode("invalid", wasmtesting.Code),
			ibcerrors.ErrInvalidAddress,
		},
	}

	for _, tc := range testCases {
		tc := tc

		err := tc.msg.ValidateBasic()
		expPass := tc.expErr == nil
		if expPass {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

func TestMsgStoreCode_GetSigners(t *testing.T) {
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

		msg := types.NewMsgStoreCode(tc.address.String(), wasmtesting.Code)
		if tc.expPass {
			require.Equal(t, []sdk.AccAddress{tc.address}, msg.GetSigners())
		} else {
			require.Panics(t, func() {
				msg.GetSigners()
			})
		}
	}
}
