package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	conntypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
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
		malleate func(msg *types.MsgRegisterCounterparty)
		expError error
	}{
		{
			"success",
			func(msg *types.MsgRegisterCounterparty) {},
			nil,
		},
		{
			"failure: client id does not match clientID format",
			func(msg *types.MsgRegisterCounterparty) {
				msg.ClientId = "testclientid1"
			},
			host.ErrInvalidID,
		},
		{
			"failure: counterparty client id does not match clientID format",
			func(msg *types.MsgRegisterCounterparty) {
				msg.CounterpartyClientId = "testclientid2"
			},
			host.ErrInvalidID,
		},
		{
			"failure: empty client id",
			func(msg *types.MsgRegisterCounterparty) {
				msg.ClientId = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: empty counterparty client id",
			func(msg *types.MsgRegisterCounterparty) {
				msg.CounterpartyClientId = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: empty counterparty messaging key",
			func(msg *types.MsgRegisterCounterparty) {
				msg.CounterpartyMerklePrefix = [][]byte{}
			},
			types.ErrInvalidCounterparty,
		},
		{
			"failure: empty signer",
			func(msg *types.MsgRegisterCounterparty) {
				msg.Signer = "badsigner"
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: counterparty merkle prefix length too large",
			func(msg *types.MsgRegisterCounterparty) {
				tooLargePrefix := make([][]byte, types.MaxCounterpartyMerklePrefixElements+1)
				for i := range tooLargePrefix {
					tooLargePrefix[i] = []byte("key")
				}
				msg.CounterpartyMerklePrefix = tooLargePrefix
			},
			ibcerrors.ErrTooLarge,
		},
		{
			"failure: counterparty merkle prefix key too large",
			func(msg *types.MsgRegisterCounterparty) {
				largeKey := make([]byte, conntypes.MaxMerklePrefixLength+1)
				msg.CounterpartyMerklePrefix = [][]byte{largeKey}
			},
			ibcerrors.ErrTooLarge,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := types.NewMsgRegisterCounterparty(
				"testclientid-3",
				[][]byte{[]byte("ibc"), []byte("channel-9")},
				"testclientid-2",
				signer,
			)

			tc.malleate(msg)

			err := msg.ValidateBasic()
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
