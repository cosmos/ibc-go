package types

import (
	"fmt"
)

// SenderChainIsSource returns false if the denomination originally came
// from the receiving chain and true otherwise.
func (t Token) SenderChainIsSource(sourcePort, sourceChannel string) bool {
	// This is the prefix that would have been prefixed to the denomination
	// on sender chain IF and only if the token originally came from the
	// receiving chain.

	return !t.ReceiverChainIsSource(sourcePort, sourceChannel)
}

// ReceiverChainIsSource returns true if the denomination originally came
// from the receiving chain and false otherwise.
func (t Token) ReceiverChainIsSource(sourcePort, sourceChannel string) bool {
	// The prefix passed in should contain the SourcePort and SourceChannel.
	// If  the receiver chain originally sent the token to the sender chain
	// the denom will have the sender's SourcePort and SourceChannel as the
	// prefix.
	if len(t.Trace) == 0 {
		return false
	}

	return t.Trace[0].PortId == sourcePort && t.Trace[0].ChannelId == sourceChannel
}

// GetDenomPrefix returns the receiving denomination prefix
func GetDenomPrefix(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/", portID, channelID)
}

// GetPrefixedDenom returns the denomination with the portID and channelID prefixed
func GetPrefixedDenom(portID, channelID, baseDenom string) string {
	return fmt.Sprintf("%s/%s/%s", portID, channelID, baseDenom)
}
