package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetPrefixedDenom returns the denomination with the portID and channelID prefixed
func GetPrefixedDenom(portID, channelID, baseDenom string) string {
	return fmt.Sprintf("%s/%s/%s", portID, channelID, baseDenom)
}

// GetTransferCoin creates a transfer coin with the port ID and channel ID
// prefixed to the base denom.
func GetTransferCoin(portID, channelID, baseDenom string, amount sdkmath.Int) sdk.Coin {
	denom := Denom{
		Base:  baseDenom,
		Trace: []string{fmt.Sprintf("%s/%s", portID, channelID)},
	}
	return sdk.NewCoin(denom.IBCDenom(), amount)
}
