package types

import "fmt"

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

	// ChannelStatusKeyPrefix is the key prefix for storing the validation status of the CCV channel
	ChannelStatusKeyPrefix = "channelstatus"

	// UnbondingChangesPrefix is the key prefix for storing unbonding changes
	UnbondingChangesPrefix = "unbondingchanges"
)

const (
	Uninitialized Status = iota
	Initialized
	Validating
	Invalid
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

// ChannelStatusKey returns the key under which the Status of a baby chain is stored.
func ChannelStatusKey(channelID string) []byte {
	return []byte(ChannelStatusKeyPrefix + "/" + channelID)
}

// UnbondingChanges stores the validator set changes that are still unbonding
func UnbondingChanges(chainID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("%s/%s/%d", UnbondingChangesPrefix, chainID, sequence))
}
