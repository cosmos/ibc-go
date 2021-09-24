package types

import (
	"fmt"
)

const (
	// ModuleName defines the interchain accounts module name
	ModuleName = "interchainaccounts"

	// VersionPrefix defines the current version for interchain accounts
	VersionPrefix = "ics27-1"

	// PortID is the default port id that the interchain accounts module binds to
	PortID = "ibcaccount"

	// StoreKey is the store key string for interchain accounts
	StoreKey = ModuleName

	// RouterKey is the message route for interchain accounts
	RouterKey = ModuleName

	// QuerierRoute is the querier route for interchain accounts
	QuerierRoute = ModuleName

	// Delimiter is the delimiter used for the interchain accounts version string
	Delimiter = "|"
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = []byte{0x01}
)

// NewVersion returns a complete version string in the format: VersionPrefix + Delimter + AccAddress
func NewAppVersion(versionPrefix, accAddr string) string {
	return fmt.Sprint(versionPrefix, Delimiter, accAddr)
}

// KeyActiveChannel creates and returns a new key used for active channels store operations
func KeyActiveChannel(portID string) []byte {
	return []byte(fmt.Sprintf("activeChannel/%s", portID))
}

// KeyOwnerAccount creates and returns a new key used for owner account store operations
func KeyOwnerAccount(portID string) []byte {
	return []byte(fmt.Sprintf("owner/%s", portID))
}
