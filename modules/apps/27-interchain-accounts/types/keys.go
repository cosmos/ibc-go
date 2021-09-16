package types

import (
	"fmt"
)

const (
	// ModuleName defines the Interchain Account module name
	ModuleName = "interchainaccounts"

	// Version defines the current version the IBC interchainaccounts
	// module supports
	Version = "ics27-1"

	// PortID is the default port id that the interchainaccounts module binds to
	PortID = "ibcaccount"

	// StoreKey is the store key string for IBC interchainaccounts
	StoreKey = ModuleName

	// RouterKey is the message route for IBC interchainaccounts
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC interchainaccounts
	QuerierRoute = ModuleName

	// Delimiter is the delimiter used for the interchainaccounts version string
	Delimiter = "|"
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = "portID"
)

func KeyActiveChannel(portId string) []byte {
	return []byte(fmt.Sprintf("activeChannel/%s", portId))
}

func KeyOwnerAccount(portId string) []byte {
	return []byte(fmt.Sprintf("owner/%s", portId))
}

func GetIdentifier(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/", portID, channelID)
}
