package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

var (
	// defaultRecvFee is the default packet receive fee used for testing purposes
	defaultRecvFee = sdk.NewCoins(sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(100)})

	// defaultAckFee is the default packet acknowledgement fee used for testing purposes
	defaultAckFee = sdk.NewCoins(sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(200)})

	// defaultTimeoutFee is the default packet timeout fee used for testing purposes
	defaultTimeoutFee = sdk.NewCoins(sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(300)})

	// invalidFee is an invalid coin set used to trigger error cases for testing purposes
	invalidFee = sdk.Coins{sdk.Coin{Denom: "invalid-denom", Amount: sdkmath.NewInt(-2)}}

	// defaultAccAddress is the default account used for testing purposes
	defaultAccAddress = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
)

const invalidAddress = "invalid-address"

func TestFeeTotal(t *testing.T) {
	var fee types.Fee

	testCases := []struct {
		name     string
		malleate func()
		expTotal sdk.Coins
	}{
		{
			"success",
			func() {},
			sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(300))),
		},
		{
			"success: empty fees",
			func() {
				fee = types.NewFee(sdk.NewCoins(), sdk.NewCoins(), sdk.NewCoins())
			},
			sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(0))),
		},
		{
			"success: multiple denoms",
			func() {
				fee = types.NewFee(
					sdk.NewCoins(
						defaultRecvFee[0],
						sdk.NewCoin("denom", sdkmath.NewInt(300)),
					),
					sdk.NewCoins(
						defaultAckFee[0],
						sdk.NewCoin("denom", sdkmath.NewInt(200)),
					),
					sdk.NewCoins(
						defaultTimeoutFee[0],
						sdk.NewCoin("denom", sdkmath.NewInt(100)),
					),
				)
			},
			sdk.NewCoins(
				sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(300)),
				sdk.NewCoin("denom", sdkmath.NewInt(500)),
			),
		},
		{
			"success: many denoms",
			func() {
				fee = types.NewFee(
					sdk.NewCoins(
						defaultRecvFee[0],
						sdk.NewCoin("denom", sdkmath.NewInt(200)),
						sdk.NewCoin("denom4", sdkmath.NewInt(100)),
						sdk.NewCoin("denom5", sdkmath.NewInt(300)),
					),
					sdk.NewCoins(
						defaultAckFee[0],
						sdk.NewCoin("denom", sdkmath.NewInt(200)),
						sdk.NewCoin("denom2", sdkmath.NewInt(100)),
						sdk.NewCoin("denom3", sdkmath.NewInt(300)),
						sdk.NewCoin("denom4", sdkmath.NewInt(100)),
					),
					sdk.NewCoins(
						defaultTimeoutFee[0],
						sdk.NewCoin("denom", sdkmath.NewInt(100)),
						sdk.NewCoin("denom2", sdkmath.NewInt(200)),
						sdk.NewCoin("denom5", sdkmath.NewInt(300)),
					),
				)
			},
			sdk.NewCoins(
				sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(300)),
				sdk.NewCoin("denom", sdkmath.NewInt(400)),
				sdk.NewCoin("denom2", sdkmath.NewInt(200)),
				sdk.NewCoin("denom3", sdkmath.NewInt(300)),
				sdk.NewCoin("denom4", sdkmath.NewInt(200)),
				sdk.NewCoin("denom5", sdkmath.NewInt(300)),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			fee = types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

			tc.malleate() // malleate mutates test data

			require.Equal(t, tc.expTotal, fee.Total())
		})
	}
}

func TestPacketFeeValidation(t *testing.T) {
	var packetFee types.PacketFee

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
			"success with empty slice for Relayers",
			func() {
				packetFee.Relayers = []string{}
			},
			nil,
		},
		{
			"should fail when refund address is invalid",
			func() {
				packetFee.RefundAddress = invalidAddress
			},
			errors.New("failed to convert RefundAddress into sdk.AccAddress"),
		},
		{
			"should fail when all fees are invalid",
			func() {
				packetFee.Fee.AckFee = invalidFee
				packetFee.Fee.RecvFee = invalidFee
				packetFee.Fee.TimeoutFee = invalidFee
			},
			ibcerrors.ErrInvalidCoins,
		},
		{
			"should fail with single invalid fee",
			func() {
				packetFee.Fee.AckFee = invalidFee
			},
			ibcerrors.ErrInvalidCoins,
		},
		{
			"should fail with two invalid fees",
			func() {
				packetFee.Fee.TimeoutFee = invalidFee
				packetFee.Fee.AckFee = invalidFee
			},
			ibcerrors.ErrInvalidCoins,
		},
		{
			"should pass with two empty fees",
			func() {
				packetFee.Fee.TimeoutFee = sdk.Coins{}
				packetFee.Fee.AckFee = sdk.Coins{}
			},
			nil,
		},
		{
			"should pass with one empty fee",
			func() {
				packetFee.Fee.TimeoutFee = sdk.Coins{}
			},
			nil,
		},
		{
			"should fail if all fees are empty",
			func() {
				packetFee.Fee.AckFee = sdk.Coins{}
				packetFee.Fee.RecvFee = sdk.Coins{}
				packetFee.Fee.TimeoutFee = sdk.Coins{}
			},
			ibcerrors.ErrInvalidCoins,
		},
		{
			"should fail with non empty Relayers",
			func() {
				packetFee.Relayers = []string{"relayer"}
			},
			types.ErrRelayersNotEmpty,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee = types.NewPacketFee(fee, defaultAccAddress, nil)

			tc.malleate() // malleate mutates test data

			err := packetFee.Validate()

			if tc.expErr == nil {
				require.NoError(t, err, tc.name)
			} else {
				ibctesting.RequireErrorIsOrContains(t, err, tc.expErr, err.Error())
			}
		})
	}
}
