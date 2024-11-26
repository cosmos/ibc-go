package types

import "fmt"

const (
	// ModuleName defines the IBC transfer-v2 name
	ModuleName = "transfer-v2"

	// StoreKey is the store key string for IBC transfer-v2
	StoreKey = ModuleName
)

// ForwardedPacketSequenceKey returns the key used to store the sequence of a forwarded packet.
// TODO(bznein) make sure the string is correct
func ForwardedPacketSequenceKey(portID string, channelID string) []byte {
	return []byte(fmt.Sprintf("forwarded/%s/%s/sequence", portID, channelID))
}

// ForwardedPacketDestinationChannelKey returns the key used to store the destinationChannel of a forwarded packet.
// TODO(bznein) make sure the string is correct
func ForwardedPacketDestinationChannelKey(portID string, channelID string) []byte {
	return []byte(fmt.Sprintf("forwarded/%s/%s/edstinationchannel", portID, channelID))
}
