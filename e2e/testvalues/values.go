package testvalues

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"

	feetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
)

const (
	StartingTokenAmount int64         = 100_000_000
	IBCTransferAmount   int64         = 10_000
	InvalidAddress      string        = "<invalid-address>"
	VotingPeriod        time.Duration = time.Second * 30
)

// ImmediatelyTimeout returns an ibc.IBCTimeout which will cause an IBC transfer to timeout immediately.
func ImmediatelyTimeout() *ibc.IBCTimeout {
	return &ibc.IBCTimeout{
		NanoSeconds: 1,
	}
}

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

func TendermintClientID(id int) string {
	return fmt.Sprintf("07-tendermint-%d", id)
}

func SolomachineClientID(id int) string {
	return fmt.Sprintf("06-solomachine-%d", id)
}
