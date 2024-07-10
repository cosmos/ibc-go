package types_test

import (
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	ica "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts"
	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	feetypes "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func TestMsgRegisterInterchainAccountValidateBasic(t *testing.T) {
	var msg *types.MsgRegisterInterchainAccount

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"success: with empty channel version",
			func() {
				msg.Version = ""
			},
			true,
		},
		{
			"success: with fee enabled channel version",
			func() {
				feeMetadata := feetypes.Metadata{
					FeeVersion: feetypes.Version,
					AppVersion: icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID),
				}

				bz := feetypes.ModuleCdc.MustMarshalJSON(&feeMetadata)
				msg.Version = string(bz)
			},
			true,
		},
		{
			"connection id is invalid",
			func() {
				msg.ConnectionId = ""
			},
			false,
		},
		{
			"owner address is empty",
			func() {
				msg.Owner = ""
			},
			false,
		},
		{
			"owner address is too long",
			func() {
				msg.Owner = ibctesting.GenerateString(types.MaximumOwnerLength + 1)
			},
			false,
		},
		{
			"order is not valid",
			func() {
				msg.Ordering = channeltypes.NONE
			},
			false,
		},
	}

	for i, tc := range testCases {
		i, tc := i, tc

		msg = types.NewMsgRegisterInterchainAccount(
			ibctesting.FirstConnectionID,
			ibctesting.TestAccAddress,
			icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID),
			channeltypes.ORDERED,
		)

		tc.malleate()

		err := msg.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
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
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"connection id is invalid",
			func() {
				msg.ConnectionId = ""
			},
			false,
		},
		{
			"owner address is empty",
			func() {
				msg.Owner = ""
			},
			false,
		},
		{
			"owner address is too long",
			func() {
				msg.Owner = ibctesting.GenerateString(types.MaximumOwnerLength + 1)
			},
			false,
		},
		{
			"relative timeout is not set",
			func() {
				msg.RelativeTimeout = 0
			},
			false,
		},
		{
			"messages array is empty",
			func() {
				msg.PacketData = icatypes.InterchainAccountPacketData{}
			},
			false,
		},
	}

	for i, tc := range testCases {
		i, tc := i, tc

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
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
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
		name    string
		msg     *types.MsgUpdateParams
		expPass bool
	}{
		{"success: valid signer and valid params", types.NewMsgUpdateParams(ibctesting.TestAccAddress, types.DefaultParams()), true},
		{"failure: invalid signer with valid params", types.NewMsgUpdateParams("invalidAddress", types.DefaultParams()), false},
		{"failure: empty signer with valid params", types.NewMsgUpdateParams("", types.DefaultParams()), false},
	}

	for i, tc := range testCases {
		i, tc := i, tc

		err := tc.msg.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

// TestMsgUpdateParamsGetSigners tests GetSigners for MsgUpdateParams
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

		msg := types.MsgUpdateParams{
			Signer: tc.address.String(),
			Params: types.DefaultParams(),
		}

		encodingCfg := moduletestutil.MakeTestEncodingConfig(ica.AppModuleBasic{})
		signers, _, err := encodingCfg.Codec.GetMsgV1Signers(&msg)
		if tc.expPass {
			require.NoError(t, err)
			require.Equal(t, tc.address.Bytes(), signers[0])
		} else {
			require.Error(t, err)
		}

	}
}
