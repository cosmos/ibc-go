package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	feetypes "github.com/cosmos/ibc-go/v5/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
	"github.com/cosmos/ibc-go/v5/testing/simapp"
)

func TestMsgRegisterAccountValidateBasic(t *testing.T) {
	var msg *types.MsgRegisterAccount

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
			"owner address is invalid",
			func() {
				msg.Owner = "invalid_address"
			},
			false,
		},
	}

	for i, tc := range testCases {

		msg = types.NewMsgRegisterAccount(
			ibctesting.FirstConnectionID,
			ibctesting.TestAccAddress,
			icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID),
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

func TestMsgRegisterAccountGetSigners(t *testing.T) {
	expSigner, err := sdk.AccAddressFromBech32(ibctesting.TestAccAddress)
	require.NoError(t, err)

	msg := types.NewMsgRegisterAccount(ibctesting.FirstConnectionID, ibctesting.TestAccAddress, "")
	require.Equal(t, []sdk.AccAddress{expSigner}, msg.GetSigners())
}

func TestMsgSubmitTxValidateBasic(t *testing.T) {
	var msg *types.MsgSubmitTx

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
			"owner address is invalid",
			func() {
				msg.Owner = "invalid_address"
			},
			false,
		},
		{
			"timeout height and timestamp are both not set",
			func() {
				msg.TimeoutTimestamp = 0
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

		msgBankSend := &banktypes.MsgSend{
			FromAddress: ibctesting.TestAccAddress,
			ToAddress:   ibctesting.TestAccAddress,
			Amount:      ibctesting.TestCoins,
		}

		data, err := icatypes.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, []sdk.Msg{msgBankSend})
		require.NoError(t, err)

		packetData := icatypes.InterchainAccountPacketData{
			Type: icatypes.EXECUTE_TX,
			Data: data,
		}

		msg = types.NewMsgSubmitTx(
			ibctesting.FirstConnectionID,
			ibctesting.TestAccAddress,
			clienttypes.ZeroHeight(),
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

func TestMsgSubmitTxGetSigners(t *testing.T) {
	expSigner, err := sdk.AccAddressFromBech32(ibctesting.TestAccAddress)
	require.NoError(t, err)

	msgBankSend := &banktypes.MsgSend{
		FromAddress: ibctesting.TestAccAddress,
		ToAddress:   ibctesting.TestAccAddress,
		Amount:      ibctesting.TestCoins,
	}

	data, err := icatypes.SerializeCosmosTx(simapp.MakeTestEncodingConfig().Marshaler, []sdk.Msg{msgBankSend})
	require.NoError(t, err)

	packetData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
	}

	msg := types.NewMsgSubmitTx(
		ibctesting.FirstConnectionID,
		ibctesting.TestAccAddress,
		clienttypes.ZeroHeight(),
		100000,
		packetData,
	)
	require.Equal(t, []sdk.AccAddress{expSigner}, msg.GetSigners())
}
