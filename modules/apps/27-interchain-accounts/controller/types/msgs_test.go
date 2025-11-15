package types_test

import (
	"errors"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	ica "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestMsgRegisterInterchainAccountValidateBasic(t *testing.T) {
	var msg *types.MsgRegisterInterchainAccount

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"success: with empty channel version",
			func() {
				msg.Version = ""
			},
			nil,
		},
		{
			"connection id is invalid",
			func() {
				msg.ConnectionId = ""
			},
			host.ErrInvalidID,
		},
		{
			"owner address is empty",
			func() {
				msg.Owner = ""
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"owner address is too long",
			func() {
				msg.Owner = ibctesting.GenerateString(types.MaximumOwnerLength + 1)
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"order is not valid",
			func() {
				msg.Ordering = channeltypes.NONE
			},
			channeltypes.ErrInvalidChannelOrdering,
		},
	}

	for i, tc := range testCases {
		msg = types.NewMsgRegisterInterchainAccount(
			ibctesting.FirstConnectionID,
			ibctesting.TestAccAddress,
			icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID),
			channeltypes.ORDERED,
		)

		tc.malleate()

		err := msg.ValidateBasic()
		if tc.expErr == nil {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.ErrorIs(t, err, tc.expErr, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

func TestMsgRegisterInterchainAccountGetSigners(t *testing.T) {
	expSigner, err := sdk.AccAddressFromBech32(ibctesting.TestAccAddress)
	require.NoError(t, err)

	msg := types.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, ibctesting.TestAccAddress, "", channeltypes.ORDERED)
	encodingCfg := moduletestutil.MakeTestEncodingConfig(ica.AppModuleBasic{})
	signers, _, err := encodingCfg.Codec.GetMsgV1Signers(msg)
	require.NoError(t, err)
	require.Equal(t, expSigner.Bytes(), signers[0])
}

func TestMsgSendTxValidateBasic(t *testing.T) {
	var msg *types.MsgSendTx

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"connection id is invalid",
			func() {
				msg.ConnectionId = ""
			},
			host.ErrInvalidID,
		},
		{
			"owner address is empty",
			func() {
				msg.Owner = ""
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"owner address is too long",
			func() {
				msg.Owner = ibctesting.GenerateString(types.MaximumOwnerLength + 1)
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"relative timeout is not set",
			func() {
				msg.RelativeTimeout = 0
			},
			ibcerrors.ErrInvalidRequest,
		},
		{
			"messages array is empty",
			func() {
				msg.PacketData = icatypes.InterchainAccountPacketData{}
			},
			icatypes.ErrInvalidOutgoingData,
		},
	}

	for i, tc := range testCases {
		msgBankSend := &banktypes.MsgSend{
			FromAddress: ibctesting.TestAccAddress,
			ToAddress:   ibctesting.TestAccAddress,
			Amount:      ibctesting.TestCoins,
		}

		encodingConfig := moduletestutil.MakeTestEncodingConfig(ica.AppModuleBasic{})

		data, err := icatypes.SerializeCosmosTx(encodingConfig.Codec, []proto.Message{msgBankSend}, icatypes.EncodingProtobuf)
		require.NoError(t, err)

		packetData := icatypes.InterchainAccountPacketData{
			Type: icatypes.EXECUTE_TX,
			Data: data,
		}

		msg = types.NewMsgSendTx(
			ibctesting.TestAccAddress,
			ibctesting.FirstConnectionID,
			100000,
			packetData,
		)

		tc.malleate()

		err = msg.ValidateBasic()
		if tc.expErr == nil {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.ErrorIs(t, err, tc.expErr, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

func TestMsgSendTxGetSigners(t *testing.T) {
	expSigner, err := sdk.AccAddressFromBech32(ibctesting.TestAccAddress)
	require.NoError(t, err)

	msgBankSend := &banktypes.MsgSend{
		FromAddress: ibctesting.TestAccAddress,
		ToAddress:   ibctesting.TestAccAddress,
		Amount:      ibctesting.TestCoins,
	}

	encodingConfig := moduletestutil.MakeTestEncodingConfig(ica.AppModuleBasic{})

	data, err := icatypes.SerializeCosmosTx(encodingConfig.Codec, []proto.Message{msgBankSend}, icatypes.EncodingProtobuf)
	require.NoError(t, err)

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
	}

	msg := types.NewMsgSendTx(
		ibctesting.TestAccAddress,
		ibctesting.FirstConnectionID,
		100000,
		packetData,
	)
	signers, _, err := encodingConfig.Codec.GetMsgV1Signers(msg)
	require.NoError(t, err)
	require.Equal(t, expSigner.Bytes(), signers[0])
}

// TestMsgUpdateParamsValidateBasic tests ValidateBasic for MsgUpdateParams
func TestMsgUpdateParamsValidateBasic(t *testing.T) {
	testCases := []struct {
		name   string
		msg    *types.MsgUpdateParams
		expErr error
	}{
		{"success: valid signer and valid params", types.NewMsgUpdateParams(ibctesting.TestAccAddress, types.DefaultParams()), nil},
		{"failure: invalid signer with valid params", types.NewMsgUpdateParams("invalidAddress", types.DefaultParams()), ibcerrors.ErrInvalidAddress},
		{"failure: empty signer with valid params", types.NewMsgUpdateParams("", types.DefaultParams()), ibcerrors.ErrInvalidAddress},
	}

	for i, tc := range testCases {
		err := tc.msg.ValidateBasic()
		if tc.expErr == nil {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.ErrorIs(t, err, tc.expErr, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

// TestMsgUpdateParamsGetSigners tests GetSigners for MsgUpdateParams
func TestMsgUpdateParamsGetSigners(t *testing.T) {
	testCases := []struct {
		name    string
		address sdk.AccAddress
		expErr  error
	}{
		{"success: valid address", sdk.AccAddress(ibctesting.TestAccAddress), nil},
		{"failure: nil address", nil, errors.New("empty address string is not allowed")},
	}

	for _, tc := range testCases {
		msg := types.MsgUpdateParams{
			Signer: tc.address.String(),
			Params: types.DefaultParams(),
		}

		encodingCfg := moduletestutil.MakeTestEncodingConfig(ica.AppModuleBasic{})
		signers, _, err := encodingCfg.Codec.GetMsgV1Signers(&msg)
		if tc.expErr == nil {
			require.NoError(t, err)
			require.Equal(t, tc.address.Bytes(), signers[0])
		} else {
			require.ErrorContains(t, err, tc.expErr.Error())
		}
	}
}
