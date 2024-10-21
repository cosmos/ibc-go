package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	modulefee "github.com/cosmos/ibc-go/v9/modules/apps/29-fee"
	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func TestMsgRegisterPayeeValidation(t *testing.T) {
	var msg *types.MsgRegisterPayee

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
			"success: relayer and payee are equal",
			func() {
				msg.Relayer = defaultAccAddress
				msg.Payee = defaultAccAddress
			},
			nil,
		},
		{
			"invalid portID",
			func() {
				msg.PortId = ""
			},
			host.ErrInvalidID,
		},
		{
			"invalid channelID",
			func() {
				msg.ChannelId = ""
			},
			host.ErrInvalidID,
		},
		{
			"invalid relayer address",
			func() {
				msg.Relayer = invalidAddress
			},
			errors.New("failed to create sdk.AccAddress from relayer address"),
		},
		{
			"invalid payee address",
			func() {
				msg.Payee = invalidAddress
			},
			errors.New("failed to create sdk.AccAddress from payee address"),
		},
	}

	for i, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			relayerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
			payeeAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

			msg = types.NewMsgRegisterPayee(ibctesting.MockPort, ibctesting.FirstChannelID, relayerAddr.String(), payeeAddr.String())

			tc.malleate()

			err := msg.ValidateBasic()

			if tc.expErr == nil {
				require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
			} else {
				ibctesting.RequireErrorIsOrContains(t, err, tc.expErr, err.Error())
			}
		})
	}
}

func TestRegisterPayeeGetSigners(t *testing.T) {
	accAddress := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	msg := types.NewMsgRegisterPayee(ibctesting.MockPort, ibctesting.FirstChannelID, accAddress.String(), defaultAccAddress)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(testutil.CodecOptions{}, modulefee.AppModule{})
	signers, _, err := encodingCfg.Codec.GetMsgSigners(msg)
	require.NoError(t, err)
	require.Equal(t, accAddress.Bytes(), signers[0])
}

func TestMsgRegisterCountepartyPayeeValidation(t *testing.T) {
	var msg *types.MsgRegisterCounterpartyPayee

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
			"invalid portID",
			func() {
				msg.PortId = ""
			},
			host.ErrInvalidID,
		},
		{
			"invalid channelID",
			func() {
				msg.ChannelId = ""
			},
			host.ErrInvalidID,
		},
		{
			"validate with incorrect destination relayer address",
			func() {
				msg.Relayer = invalidAddress
			},
			errors.New("failed to create sdk.AccAddress from relayer address"),
		},
		{
			"invalid counterparty payee address",
			func() {
				msg.CounterpartyPayee = ""
			},
			types.ErrCounterpartyPayeeEmpty,
		},
		{
			"invalid counterparty payee address: whitespaced empty string",
			func() {
				msg.CounterpartyPayee = "  "
			},
			types.ErrCounterpartyPayeeEmpty,
		},
		{
			"invalid counterparty payee address: too long",
			func() {
				msg.CounterpartyPayee = ibctesting.GenerateString(types.MaximumCounterpartyPayeeLength + 1)
			},
			ibcerrors.ErrInvalidAddress,
		},
	}

	for i, tc := range testCases {
		i, tc := i, tc

		t.Run(tc.name, func(t *testing.T) {
			payeeAddr, err := sdk.AccAddressFromBech32(ibctesting.TestAccAddress)
			require.NoError(t, err)
			msg = types.NewMsgRegisterCounterpartyPayee(ibctesting.MockPort, ibctesting.FirstChannelID, defaultAccAddress, payeeAddr.String())

			tc.malleate()

			err = msg.ValidateBasic()

			if tc.expErr == nil {
				require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
			} else {
				ibctesting.RequireErrorIsOrContains(t, err, tc.expErr, err.Error())
			}
		})
	}
}

func TestRegisterCountepartyAddressGetSigners(t *testing.T) {
	accAddress := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	msg := types.NewMsgRegisterCounterpartyPayee(ibctesting.MockPort, ibctesting.FirstChannelID, accAddress.String(), defaultAccAddress)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(testutil.CodecOptions{}, modulefee.AppModule{})
	signers, _, err := encodingCfg.Codec.GetMsgSigners(msg)
	require.NoError(t, err)
	require.Equal(t, accAddress.Bytes(), signers[0])
}

func TestMsgPayPacketFeeValidation(t *testing.T) {
	var msg *types.MsgPayPacketFee

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
			"success with empty relayers",
			func() {
				msg.Relayers = []string{}
			},
			nil,
		},
		{
			"invalid channelID",
			func() {
				msg.SourceChannelId = ""
			},
			host.ErrInvalidID,
		},
		{
			"invalid portID",
			func() {
				msg.SourcePortId = ""
			},
			host.ErrInvalidID,
		},
		{
			"relayers is not nil",
			func() {
				msg.Relayers = []string{defaultAccAddress}
			},
			types.ErrRelayersNotEmpty,
		},
		{
			"invalid signer address",
			func() {
				msg.Signer = invalidAddress
			},
			errors.New("failed to convert msg.Signer into sdk.AccAddress"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
		msg = types.NewMsgPayPacketFee(fee, ibctesting.MockFeePort, ibctesting.FirstChannelID, defaultAccAddress, nil)

		tc.malleate() // malleate mutates test data

		err := msg.ValidateBasic()

		if tc.expErr == nil {
			require.NoError(t, err, tc.name)
		} else {
			ibctesting.RequireErrorIsOrContains(t, err, tc.expErr, err.Error())
		}
	}
}

func TestPayPacketFeeGetSigners(t *testing.T) {
	refundAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
	msg := types.NewMsgPayPacketFee(fee, ibctesting.MockFeePort, ibctesting.FirstChannelID, refundAddr.String(), nil)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(testutil.CodecOptions{}, modulefee.AppModule{})
	signers, _, err := encodingCfg.Codec.GetMsgSigners(msg)
	require.NoError(t, err)
	require.Equal(t, refundAddr.Bytes(), signers[0])
}

func TestMsgPayPacketFeeAsyncValidation(t *testing.T) {
	var msg *types.MsgPayPacketFeeAsync

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
			"success with empty relayers",
			func() {
				msg.PacketFee.Relayers = []string{}
			},
			nil,
		},
		{
			"should pass with two empty fees",
			func() {
				msg.PacketFee.Fee.AckFee = sdk.Coins{}
				msg.PacketFee.Fee.TimeoutFee = sdk.Coins{}
			},
			nil,
		},
		{
			"should pass with one empty fee",
			func() {
				msg.PacketFee.Fee.TimeoutFee = sdk.Coins{}
			},
			nil,
		},
		{
			"invalid channelID",
			func() {
				msg.PacketId.ChannelId = ""
			},
			host.ErrInvalidID,
		},
		{
			"invalid portID",
			func() {
				msg.PacketId.PortId = ""
			},
			host.ErrInvalidID,
		},
		{
			"invalid sequence",
			func() {
				msg.PacketId.Sequence = 0
			},
			channeltypes.ErrInvalidPacket,
		},
		{
			"relayers is not nil",
			func() {
				msg.PacketFee.Relayers = []string{defaultAccAddress}
			},
			types.ErrRelayersNotEmpty,
		},
		{
			"invalid signer address",
			func() {
				msg.PacketFee.RefundAddress = "invalid-addr"
			},
			errors.New("failed to convert RefundAddress into sdk.AccAddress"),
		},
		{
			"should fail when all fees are invalid",
			func() {
				msg.PacketFee.Fee.AckFee = invalidFee
				msg.PacketFee.Fee.RecvFee = invalidFee
				msg.PacketFee.Fee.TimeoutFee = invalidFee
			},
			ibcerrors.ErrInvalidCoins,
		},
		{
			"should fail with single invalid fee",
			func() {
				msg.PacketFee.Fee.AckFee = invalidFee
			},
			ibcerrors.ErrInvalidCoins,
		},
		{
			"should fail with two invalid fees",
			func() {
				msg.PacketFee.Fee.AckFee = invalidFee
				msg.PacketFee.Fee.TimeoutFee = invalidFee
			},
			ibcerrors.ErrInvalidCoins,
		},
		{
			"should fail if all fees are empty",
			func() {
				msg.PacketFee.Fee.AckFee = sdk.Coins{}
				msg.PacketFee.Fee.RecvFee = sdk.Coins{}
				msg.PacketFee.Fee.TimeoutFee = sdk.Coins{}
			},
			ibcerrors.ErrInvalidCoins,
		},
	}

	for _, tc := range testCases {
		tc := tc

		packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)
		fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
		packetFee := types.NewPacketFee(fee, defaultAccAddress, nil)

		msg = types.NewMsgPayPacketFeeAsync(packetID, packetFee)

		tc.malleate() // malleate mutates test data

		err := msg.ValidateBasic()

		if tc.expErr == nil {
			require.NoError(t, err, tc.name)
		} else {
			ibctesting.RequireErrorIsOrContains(t, err, tc.expErr, err.Error())
		}
	}
}

func TestPayPacketFeeAsyncGetSigners(t *testing.T) {
	refundAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)
	fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
	packetFee := types.NewPacketFee(fee, refundAddr.String(), nil)
	msg := types.NewMsgPayPacketFeeAsync(packetID, packetFee)

	encodingCfg := moduletestutil.MakeTestEncodingConfig(testutil.CodecOptions{}, modulefee.AppModule{})
	signers, _, err := encodingCfg.Codec.GetMsgSigners(msg)
	require.NoError(t, err)
	require.Equal(t, refundAddr.Bytes(), signers[0])
}
