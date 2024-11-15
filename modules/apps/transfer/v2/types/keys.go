package types

import "fmt"

// ForwardedPacketSequenceKey returns the key used to store the sequence of a forwarded packet.
// TODO make sure the string is correct
func ForwardedPacketSequenceKey(portID string, channelID string) []byte {
	return []byte(fmt.Sprintf("forwarded/%s/%s/sequence", portID, channelID))
}

// ForwardedPacketDestinationChannelKey returns the key used to store the destinationChannel of a forwarded packet.
// TODO make sure the string is correct
func ForwardedPacketDestinationChannelKey(portID string, channelID string) []byte {
	return []byte(fmt.Sprintf("forwarded/%s/%s/edstinationchannel", portID, channelID))
}
