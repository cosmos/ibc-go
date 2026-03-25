package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// SubModuleName defines the channelv2 module name.
	SubModuleName = "channelv2"

	// KeyAsyncPacket defines the key to store the async packet.
	KeyAsyncPacket = "async_packet"
)

// AsyncPacketKey returns the key under which the packet is stored
// if the receiving application returns an async acknowledgement.
func AsyncPacketKey(clientID string, sequence uint64) []byte {
	return append(AsyncPacketPrefixKey(clientID), sdk.Uint64ToBigEndian(sequence)...)
}

// AsyncPacketPrefixKey returns the prefix key under which all async packets are stored
// for a given clientID.
func AsyncPacketPrefixKey(clientID string) []byte {
	return append([]byte(clientID), []byte(KeyAsyncPacket)...)
}
