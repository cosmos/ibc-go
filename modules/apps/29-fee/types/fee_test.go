package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestFeeTotal(t *testing.T) {
	fee := Fee{
		AckFee:     sdk.NewCoins(sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}),
		RecvFee:    sdk.NewCoins(sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}),
		TimeoutFee: sdk.NewCoins(sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}),
	}

	total := fee.Total()
	require.Equal(t, sdk.NewInt(300), total.AmountOf(sdk.DefaultBondDenom))
}

// TestFeeValidation tests Validate
func TestFeeValidation(t *testing.T) {
	var (
		fee        Fee
		ackFee     sdk.Coins
		receiveFee sdk.Coins
		timeoutFee sdk.Coins
	)

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
			"should fail when all fees are invalid",
			func() {
				ackFee = invalidCoins
				receiveFee = invalidCoins
				timeoutFee = invalidCoins
			},
			false,
		},
		{
			"should fail with single invalid fee",
			func() {
				ackFee = invalidCoins
			},
			false,
		},
		{
			"should fail with two invalid fees",
			func() {
				timeoutFee = invalidCoins
				ackFee = invalidCoins
			},
			false,
		},
		{
			"should pass with two empty fees",
			func() {
				timeoutFee = sdk.Coins{}
				ackFee = sdk.Coins{}
			},
			true,
		},
		{
			"should pass with one empty fee",
			func() {
				timeoutFee = sdk.Coins{}
			},
			true,
		},
		{
			"should fail if all fees are empty",
			func() {
				ackFee = sdk.Coins{}
				receiveFee = sdk.Coins{}
				timeoutFee = sdk.Coins{}
			},
			false,
		},
	}

	for _, tc := range testCases {
		// build message
		ackFee = validCoins
		receiveFee = validCoins
		timeoutFee = validCoins

		// malleate
		tc.malleate()
		fee = Fee{receiveFee, ackFee, timeoutFee}
		err := fee.Validate()

		if tc.expPass {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
