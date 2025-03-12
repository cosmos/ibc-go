package types_test

import (
	"crypto/sha256"
	fmt "fmt"
	"testing"

	errorsmod "cosmossdk.io/errors"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

var (
	signer1 = sdk.MustAccAddressFromBech32(ibctesting.TestAccAddress)
	hash2   = sha256.Sum256([]byte("signer2"))
	signer2 = sdk.AccAddress(hash2[:])
	hash3   = sha256.Sum256([]byte("signer3"))
	signer3 = sdk.AccAddress(hash3[:])
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
		tc := tc
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

func TestMsgUpdateClientV2ParamsValidateBasic(t *testing.T) {
	tooManyRelayers := make([]string, types.MaxAllowedRelayersLength+1)
	for i, _ := range tooManyRelayers {
		tooManyRelayers[i] = ibctesting.TestAccAddress
	}
	signer := ibctesting.TestAccAddress
	testCases := []struct {
		name     string
		msg      *types.MsgUpdateClientV2Params
		expError error
	}{
		{
			"success",
			types.NewMsgUpdateClientV2Params(
				"testclientid-2",
				signer,
				ibctesting.TestAccAddress,
			),
			nil,
		},
		{
			"success with multiple relayers",
			types.NewMsgUpdateClientV2Params(
				"testclientid-2",
				signer,
				ibctesting.TestAccAddress, signer2.String(), signer3.String(),
			),
			nil,
		},
		{
			"success with no relayers",
			types.NewMsgUpdateClientV2Params(
				"testclientid-2",
				signer,
			),
			nil,
		},
		{
			"failure: client id does not match clientID format",
			types.NewMsgUpdateClientV2Params(
				"testclientid1",
				signer,
				ibctesting.TestAccAddress,
			),
			errorsmod.Wrapf(host.ErrInvalidID, "client ID %s must be in valid format: {string}-{number}", "testclientid1"),
		},
		{
			"failure: empty client id",
			types.NewMsgUpdateClientV2Params(
				"",
				signer,
				ibctesting.TestAccAddress,
			),
			errorsmod.Wrapf(host.ErrInvalidID, "identifier cannot be blank"),
		},
		{
			"failure: empty signer",
			types.NewMsgUpdateClientV2Params(
				"testclientid-2",
				"",
				ibctesting.TestAccAddress,
			),
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: empty address string is not allowed"),
		},
		{
			"failure: invalid signer",
			types.NewMsgUpdateClientV2Params(
				"testclientid-2",
				"badsigner",
				ibctesting.TestAccAddress,
			),
			errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: decoding bech32 failed: invalid separator index -1"),
		},
		{
			"failure: invalid allowed relayer address",
			types.NewMsgUpdateClientV2Params(
				"testclientid-2",
				signer,
				"invalidAddress",
			),
			fmt.Errorf("invalid relayer address: %s", "invalidAddress"),
		},
		{
			"failure: invalid allowed relayer address with valid ones",
			types.NewMsgUpdateClientV2Params(
				"testclientid-2",
				signer,
				"invalidAddress",
				ibctesting.TestAccAddress,
			),
			fmt.Errorf("invalid relayer address: %s", "invalidAddress"),
		},
		{
			"failure: too many allowed relayers",
			types.NewMsgUpdateClientV2Params(
				"testclientid-2",
				signer,
				tooManyRelayers...,
			),
			fmt.Errorf("allowed relayers length must not exceed %d items", types.MaxAllowedRelayersLength),
		},
	}

	for _, tc := range testCases {
		tc := tc
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
