package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v4/testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

func TestMsgRegisterPayeeValidation(t *testing.T) {
	var msg *types.MsgRegisterPayee

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
			"invalid portID",
			func() {
				msg.PortId = ""
			},
			false,
		},
		{
			"invalid channelID",
			func() {
				msg.ChannelId = ""
			},
			false,
		},
		{
			"invalid request relayer and payee are equal",
			func() {
				msg.Relayer = defaultAccAddress
				msg.Payee = defaultAccAddress
			},
			false,
		},
		{
			"invalid relayer address",
			func() {
				msg.Relayer = "invalid-address"
			},
			false,
		},
		{
			"invalid payee address",
			func() {
				msg.Payee = "invalid-address"
			},
			false,
		},
	}

	for i, tc := range testCases {
		relayerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
		payeeAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

		msg = types.NewMsgRegisterPayee(ibctesting.MockPort, ibctesting.FirstChannelID, relayerAddr.String(), payeeAddr.String())

		tc.malleate()

		err := msg.ValidateBasic()

		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

func TestRegisterPayeeGetSigners(t *testing.T) {
	accAddress := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	msg := types.NewMsgRegisterPayee(ibctesting.MockPort, ibctesting.FirstChannelID, accAddress.String(), defaultAccAddress)
	require.Equal(t, []sdk.AccAddress{sdk.AccAddress(accAddress)}, msg.GetSigners())
}

func TestMsgRegisterCountepartyPayeeValidation(t *testing.T) {
	var msg *types.MsgRegisterCounterpartyPayee

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
			"invalid portID",
			func() {
				msg.PortId = ""
			},
			false,
		},
		{
			"invalid channelID",
			func() {
				msg.ChannelId = ""
			},
			false,
		},
		{
			"validate with incorrect destination relayer address",
			func() {
				msg.Relayer = "invalid-address"
			},
			false,
		},
		{
			"invalid counterparty payee address",
			func() {
				msg.CounterpartyPayee = ""
			},
			false,
		},
		{
			"invalid counterparty payee address: whitespaced empty string",
			func() {
				msg.CounterpartyPayee = "  "
			},
			false,
		},
	}

	for i, tc := range testCases {
		msg = types.NewMsgRegisterCounterpartyPayee(ibctesting.MockPort, ibctesting.FirstChannelID, defaultAccAddress, defaultAccAddress)

		tc.malleate()

		err := msg.ValidateBasic()

		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

func TestRegisterCountepartyAddressGetSigners(t *testing.T) {
	accAddress := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	msg := types.NewMsgRegisterCounterpartyPayee(ibctesting.MockPort, ibctesting.FirstChannelID, accAddress.String(), defaultAccAddress)
	require.Equal(t, []sdk.AccAddress{sdk.AccAddress(accAddress)}, msg.GetSigners())
}

func TestMsgPayPacketFeeValidation(t *testing.T) {
	var msg *types.MsgPayPacketFee

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
			"invalid channelID",
			func() {
				msg.SourceChannelId = ""
			},
			false,
		},
		{
			"invalid portID",
			func() {
				msg.SourcePortId = ""
			},
			false,
		},
		{
			"relayers is not nil",
			func() {
				msg.Relayers = []string{defaultAccAddress}
			},
			false,
		},
		{
			"invalid signer address",
			func() {
				msg.Signer = "invalid-address"
			},
			false,
		},
	}

	for _, tc := range testCases {
		fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
		msg = types.NewMsgPayPacketFee(fee, ibctesting.MockFeePort, ibctesting.FirstChannelID, defaultAccAddress, nil)

		tc.malleate() // malleate mutates test data

		err := msg.ValidateBasic()

		if tc.expPass {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

func TestPayPacketFeeGetSigners(t *testing.T) {
	refundAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
	msg := types.NewMsgPayPacketFee(fee, ibctesting.MockFeePort, ibctesting.FirstChannelID, refundAddr.String(), nil)

	require.Equal(t, []sdk.AccAddress{refundAddr}, msg.GetSigners())
}

func TestMsgPayPacketFeeRoute(t *testing.T) {
	var msg types.MsgPayPacketFee
	require.Equal(t, types.RouterKey, msg.Route())
}

func TestMsgPayPacketFeeType(t *testing.T) {
	var msg types.MsgPayPacketFee
	require.Equal(t, "payPacketFee", msg.Type())
}

func TestMsgPayPacketFeeGetSignBytes(t *testing.T) {
	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
	msg := types.NewMsgPayPacketFee(fee, ibctesting.MockFeePort, ibctesting.FirstChannelID, defaultAccAddress, nil)

	require.NotPanics(t, func() {
		_ = msg.GetSignBytes()
	})
}

func TestMsgPayPacketFeeAsyncValidation(t *testing.T) {
	var msg *types.MsgPayPacketFeeAsync

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
			"invalid channelID",
			func() {
				msg.PacketId.ChannelId = ""
			},
			false,
		},
		{
			"invalid portID",
			func() {
				msg.PacketId.PortId = ""
			},
			false,
		},
		{
			"invalid sequence",
			func() {
				msg.PacketId.Sequence = 0
			},
			false,
		},
		{
			"relayers is not nil",
			func() {
				msg.PacketFee.Relayers = []string{defaultAccAddress}
			},
			false,
		},
		{
			"invalid signer address",
			func() {
				msg.PacketFee.RefundAddress = "invalid-addr"
			},
			false,
		},
		{
			"should fail when all fees are invalid",
			func() {
				msg.PacketFee.Fee.AckFee = invalidFee
				msg.PacketFee.Fee.RecvFee = invalidFee
				msg.PacketFee.Fee.TimeoutFee = invalidFee
			},
			false,
		},
		{
			"should fail with single invalid fee",
			func() {
				msg.PacketFee.Fee.AckFee = invalidFee
			},
			false,
		},
		{
			"should fail with two invalid fees",
			func() {
				msg.PacketFee.Fee.AckFee = invalidFee
				msg.PacketFee.Fee.TimeoutFee = invalidFee
			},
			false,
		},
		{
			"should pass with two empty fees",
			func() {
				msg.PacketFee.Fee.AckFee = sdk.Coins{}
				msg.PacketFee.Fee.TimeoutFee = sdk.Coins{}
			},
			true,
		},
		{
			"should pass with one empty fee",
			func() {
				msg.PacketFee.Fee.TimeoutFee = sdk.Coins{}
			},
			true,
		},
		{
			"should fail if all fees are empty",
			func() {
				msg.PacketFee.Fee.AckFee = sdk.Coins{}
				msg.PacketFee.Fee.RecvFee = sdk.Coins{}
				msg.PacketFee.Fee.TimeoutFee = sdk.Coins{}
			},
			false,
		},
	}

	for _, tc := range testCases {
		packetID := channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)
		fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
		packetFee := types.NewPacketFee(fee, defaultAccAddress, nil)

		msg = types.NewMsgPayPacketFeeAsync(packetID, packetFee)

		tc.malleate() // malleate mutates test data

		err := msg.ValidateBasic()

		if tc.expPass {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

func TestPayPacketFeeAsyncGetSigners(t *testing.T) {
	refundAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	packetID := channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)
	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
	packetFee := types.NewPacketFee(fee, refundAddr.String(), nil)

	msg := types.NewMsgPayPacketFeeAsync(packetID, packetFee)

	require.Equal(t, []sdk.AccAddress{refundAddr}, msg.GetSigners())
}

func TestMsgPayPacketFeeAsyncRoute(t *testing.T) {
	var msg types.MsgPayPacketFeeAsync
	require.Equal(t, types.RouterKey, msg.Route())
}

func TestMsgPayPacketFeeAsyncType(t *testing.T) {
	var msg types.MsgPayPacketFeeAsync
	require.Equal(t, "payPacketFeeAsync", msg.Type())
}

func TestMsgPayPacketFeeAsyncGetSignBytes(t *testing.T) {
	packetID := channeltypes.NewPacketId(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)
	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
	packetFee := types.NewPacketFee(fee, defaultAccAddress, nil)

	msg := types.NewMsgPayPacketFeeAsync(packetID, packetFee)

	require.NotPanics(t, func() {
		_ = msg.GetSignBytes()
	})
}
