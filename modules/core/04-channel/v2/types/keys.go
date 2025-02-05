package types

import "fmt"

const (
	// SubModuleName defines the channelv2 module name.
	SubModuleName = "channelv2"

	// KeyAsyncPacket defines the key to store the async packet.
	KeyAsyncPacket = "async_packet"
)

// AsyncPacketKey returns the key under which the packet is stored
// if the receiving application returns an async acknowledgement.
func AsyncPacketKey(clientID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("%s/%s/%d", KeyAsyncPacket, clientID, sequence))
}
