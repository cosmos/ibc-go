package types

import (
	fmt "fmt"
	"strings"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Deprecated: usage of this function should be replaced by `Denom.hasPrefix`
// SenderChainIsSource returns false if the denomination originally came
// from the receiving chain and true otherwise.
func SenderChainIsSource(sourcePort, sourceChannel, denom string) bool {
	// This is the prefix that would have been prefixed to the denomination
	// on sender chain IF and only if the token originally came from the
	// receiving chain.

	return !ReceiverChainIsSource(sourcePort, sourceChannel, denom)
}

// Deprecated: usage of this function should be replaced by `Denom.hasPrefix`
// ReceiverChainIsSource returns true if the denomination originally came
// from the receiving chain and false otherwise.
func ReceiverChainIsSource(sourcePort, sourceChannel, denom string) bool {
	// The prefix passed in should contain the SourcePort and SourceChannel.
	// If  the receiver chain originally sent the token to the sender chain
	// the denom will have the sender's SourcePort and SourceChannel as the
	// prefix.

	voucherPrefix := GetDenomPrefix(sourcePort, sourceChannel)
	return strings.HasPrefix(denom, voucherPrefix)
}

// Deprecated: usage of this function should be replaced by `NewHop`
// GetDenomPrefix returns the receiving denomination prefix
func GetDenomPrefix(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/", portID, channelID)
}

// Deprecated: usage of this function should be replaced by `NewDenom`
// GetPrefixedDenom returns the denomination with the portID and channelID prefixed
func GetPrefixedDenom(portID, channelID, baseDenom string) string {
	return fmt.Sprintf("%s/%s/%s", portID, channelID, baseDenom)
}

// Deprecated: usage of this function should be replaced by `Token.ToCoin`
// GetTransferCoin creates a transfer coin with the port ID and channel ID
// prefixed to the base denom.
func GetTransferCoin(portID, channelID, baseDenom string, amount sdkmath.Int) sdk.Coin {
	denomTrace := ExtractDenomFromPath(GetPrefixedDenom(portID, channelID, baseDenom))
	return sdk.NewCoin(denomTrace.IBCDenom(), amount)
}

// Deprecated: usage of this function should be replaced by `ExtractDenomFromPath`
// ExtractDenomFromPath returns the denom from the full path.
func ParseDenomTrace(rawDenom string) Denom {
	return ExtractDenomFromPath(rawDenom)
}
