package types

import "fmt"

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
)

var (
	// ModuleAccountKey defines the key used to construct to the IBC interchainaccounts module account address
	ModuleAccountKey = []byte(ModuleName)

	// Key to store portID in our store
	PortKey = "portID"

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_capability"

	KeyPrefixRegisteredAccount = []byte("register")
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
