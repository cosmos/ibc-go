package types

import "fmt"

const (
	// ModuleName defines the IBC transfer name
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
