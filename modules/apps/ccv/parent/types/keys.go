package types

import (
	"encoding/binary"
	"fmt"
	"time"
)

type Status int

const (
	// ModuleName defines the CCV parent module name
	ModuleName = "parent"

	// PortID is the default port id that transfer module binds to
	PortID = "parent"

	// StoreKey is the store key string for IBC transfer
	StoreKey = ModuleName

	// RouterKey is the message route for IBC transfer
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC transfer
	QuerierRoute = ModuleName

	// ChainToChannelKeyPrefix is the key prefix for storing mapping
	// from chainID to the channel ID that is used to send over validator set changes.
	ChainToChannelKeyPrefix = "chaintochannel"

	// ChannelToChainKeyPrefix is the key prefix for storing mapping
	// from the CCV channel ID to the baby chain ID.
	ChannelToChainKeyPrefix = "channeltochain"

	// ChainToClientKeyPrefix is the key prefix for storing the child chainID for a given child clientid.
	ChainToClientKeyPrefix = "chaintoclient"

	// PendingClientKeyPrefix is the key prefix for storing the pending identified child chain client before the spawn time occurs.
	// The key includes the BigEndian timestamp to allow for efficient chronological iteration
	PendingClientKeyPrefix = "pendingclient"
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = []byte{0x01}
)

// ChainToChannelKey returns the key under which the CCV channel ID will be stored for the given baby chain.
func ChainToChannelKey(chainID string) []byte {
	return []byte(ChainToChannelKeyPrefix + "/" + chainID)
}

// ChannelToChainKey returns the key under which the baby chain ID will be stored for the given channelID.
func ChannelToChainKey(channelID string) []byte {
	return []byte(ChannelToChainKeyPrefix + "/" + channelID)
}

// ChainToClientKey returns the key under which the clientID for the given chainID is stored.
func ChainToClientKey(chainID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", ChainToClientKeyPrefix, chainID))
}

// PendingClientKey returns the key under which a pending identified client is store
func PendingClientKey(timestamp time.Time, chainID string) []byte {
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(timestamp.UnixNano()))
	return []byte(fmt.Sprintf("%s/%s/%s", PendingClientKeyPrefix, timeBytes, chainID))
}
