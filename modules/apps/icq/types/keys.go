package types

import "fmt"

const (
	// ModuleName defines the interchain query module name
	ModuleName = "interchainquery"

	// PortID is the default port id that the interchain query submodules binds to
	PortID = "icq"

	// PortPrefix is the default port prefix that the interchain query controller submodule binds to
	PortPrefix = "icqcontroller-"

	// Version defines the current version for interchain query
	Version = "icq-1"

	// StoreKey is the store key string for interchain query
	StoreKey = ModuleName

	// RouterKey is the message route for interchain query
	RouterKey = ModuleName

	// QuerierRoute is the querier route for interchain query
	QuerierRoute = ModuleName
)

var (
	// PortKeyPrefix defines the key prefix used to store ports
	PortKeyPrefix = "port"
)

// KeyPort creates and returns a new key used for port store operations
func KeyPort(portID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", PortKeyPrefix, portID))
}
