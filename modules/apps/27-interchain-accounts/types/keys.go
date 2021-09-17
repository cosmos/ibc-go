package types

import "fmt"

const (
	// ModuleName defines the Interchain Account module name
	ModuleName = "interchainaccounts"

	// Version defines the current version the IBC tranfer
	// module supports
	Version = "ics27-1"

	PortID = "ibcaccount"

	StoreKey  = ModuleName
	RouterKey = ModuleName

	// Key to store portID in our store
	PortKey = "portID"

	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_capability"

	KeyAppCapabilityPrefix = "appCapabilities"
)

func KeyActiveChannel(portId string) []byte {
	return []byte(fmt.Sprintf("activeChannel/%s", portId))
}

func KeyOwnerAccount(portId string) []byte {
	return []byte(fmt.Sprintf("owner/%s", portId))
}

var (
	KeyPrefixRegisteredAccount = []byte("register")
)

func GetIdentifier(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/", portID, channelID)
}

// TODO: remove once generic function is added to 24-host
func AppCapabilityName(channelID, portID string) string {
	return fmt.Sprintf("%s/%s/%s", KeyAppCapabilityPrefix, channelID, portID)
}
