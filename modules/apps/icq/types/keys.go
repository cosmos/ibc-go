package types

import "fmt"

const (
	// ModuleName defines the interchain query module name
	ModuleName = "interchainquery"

	// PortID is the default port id that the interchain query module binds to
	PortID = "icqhost"

	// PortPrefix is the default port prefix that the interchain query controller submodules bind to
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

// ContainsQueryPath returns true if the path is present in allowQueries, otherwise false
func ContainsQueryPath(allowQueries []string, path string) bool {
	for _, v := range allowQueries {
		if v == path {
			return true
		}
	}

	return false
}
