package testvalues

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	feetypes "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
)

const (
	StartingTokenAmount int64 = 10_000_000
	IBCTransferAmount   int64 = 10_000
	// ImmediatelyTimeout can be used as the value for timeout in nano seconds when we want
	// something to timeout immediately.
	ImmediatelyTimeout uint64 = 1
)

func DefaultFee(denom string) feetypes.Fee {
	return feetypes.Fee{
		RecvFee:    sdk.NewCoins(sdk.NewCoin(denom, sdk.NewInt(50))),
		AckFee:     sdk.NewCoins(sdk.NewCoin(denom, sdk.NewInt(25))),
		TimeoutFee: sdk.NewCoins(sdk.NewCoin(denom, sdk.NewInt(10))),
	}
}

func DefaultTransferAmount(denom string) sdk.Coin {
	return sdk.Coin{Denom: denom, Amount: sdk.NewInt(IBCTransferAmount)}
}
