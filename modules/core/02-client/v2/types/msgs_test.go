package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

var (
	signer1 = sdk.MustAccAddressFromBech32(ibctesting.TestAccAddress)
	signer2 = sdk.AccAddress([]byte("signer2"))
	signer3 = sdk.AccAddress([]byte("signer3"))
)

func TestMsgRegisterCounterpartyValidateBasic(t *testing.T) {
	signer := ibctesting.TestAccAddress
	testCases := []struct {
		name     string
		msg      *types.MsgRegisterCounterparty
		expError error
	}{
		{
			"success",
			types.NewMsgRegisterCounterparty(
				"testclientid-3",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"testclientid-2",
				signer,
			),
			nil,
		},
		{
			"failure: client id does not match clientID format",
			types.NewMsgRegisterCounterparty(
				"testclientid1",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"testclientid-3",
				signer,
			),
			host.ErrInvalidID,
		},
		{
			"failure: counterparty client id does not match clientID format",
			types.NewMsgRegisterCounterparty(
				"testclientid-1",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"testclientid2",
				signer,
			),
			host.ErrInvalidID,
		},
		{
			"failure: empty client id",
			types.NewMsgRegisterCounterparty(
				"",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"testclientid-3",
				signer,
			),
			host.ErrInvalidID,
		},
		{
			"failure: empty counterparty client id",
			types.NewMsgRegisterCounterparty(
				"testclientid-1",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"",
				signer,
			),
			host.ErrInvalidID,
		},
		{
			"failure: empty counterparty messaging key",
			types.NewMsgRegisterCounterparty(
				"testclientid-1",
				[][]byte{},
				"testclientid-3",
				signer,
			),
			types.ErrInvalidCounterparty,
		},
		{
			"failure: empty signer",
			types.NewMsgRegisterCounterparty(
				"testclientid-2",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"testclientid-3",
				"badsigner",
			),
			ibcerrors.ErrInvalidAddress,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expError == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expError)
			}
		})
	}
}

func TestMsgUpdateClientConfigValidateBasic(t *testing.T) {
	tooManyRelayers := make([]string, types.MaxAllowedRelayersLength+1)
	for i := range tooManyRelayers {
		tooManyRelayers[i] = ibctesting.TestAccAddress
	}
	signer := ibctesting.TestAccAddress
	testCases := []struct {
		name     string
		msg      *types.MsgUpdateClientConfig
		expError error
	}{
		{
			"success",
			types.NewMsgUpdateClientConfig(
				"testclientid-2",
				signer,
				types.NewConfig(ibctesting.TestAccAddress),
			),
			nil,
		},
		{
			"success with multiple relayers",
			types.NewMsgUpdateClientConfig(
				"testclientid-2",
				signer,
				types.NewConfig(ibctesting.TestAccAddress, signer2.String(), signer3.String()),
			),
			nil,
		},
		{
			"success with no relayers",
			types.NewMsgUpdateClientConfig(
				"testclientid-2",
				signer,
				types.DefaultConfig(),
			),
			nil,
		},
		{
			"failure: client id does not match clientID format",
			types.NewMsgUpdateClientConfig(
				"testclientid1",
				signer,
				types.NewConfig(ibctesting.TestAccAddress),
			),
			errorsmod.Wrapf(host.ErrInvalidID, "client ID %s must be in valid format: {string}-{number}", "testclientid1"),
		},
		{
			"failure: empty client id",
			types.NewMsgUpdateClientConfig(
				"",
				signer,
				types.NewConfig(ibctesting.TestAccAddress),
			),
			errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank"),
		},
		{
			"failure: empty signer",
			types.NewMsgUpdateClientConfig(
				"testclientid-2",
				"",
				types.NewConfig(ibctesting.TestAccAddress),
			),
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: empty address string is not allowed"),
		},
		{
			"failure: invalid signer",
			types.NewMsgUpdateClientConfig(
				"testclientid-2",
				"badsigner",
				types.NewConfig(ibctesting.TestAccAddress),
			),
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: decoding bech32 failed: invalid separator index -1"),
		},
		{
			"failure: invalid allowed relayer address",
			types.NewMsgUpdateClientConfig(
				"testclientid-2",
				signer,
				types.NewConfig("invalidAddress"),
			),
			fmt.Errorf("invalid relayer address: %s", "invalidAddress"),
		},
		{
			"failure: invalid allowed relayer address with valid ones",
			types.NewMsgUpdateClientConfig(
				"testclientid-2",
				signer,
				types.NewConfig("invalidAddress", ibctesting.TestAccAddress),
			),
			fmt.Errorf("invalid relayer address: %s", "invalidAddress"),
		},
		{
			"failure: too many allowed relayers",
			types.NewMsgUpdateClientConfig(
				"testclientid-2",
				signer,
				types.NewConfig(tooManyRelayers...),
			),
			fmt.Errorf("allowed relayers length must not exceed %d items", types.MaxAllowedRelayersLength),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expError == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.expError.Error(), err.Error())
			}
		})
	}
}
