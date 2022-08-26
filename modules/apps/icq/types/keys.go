package types

const (
	// ModuleName defines the interchain query module name
	ModuleName = "interchainquery"

	// PortID is the default port id that the interchain query module binds to
	PortID = "icqhost"

	// Version defines the current version for interchain query
	Version = "icq-1"

	// StoreKey is the store key string for interchain query
	StoreKey = ModuleName

	// RouterKey is the message route for interchain query
	RouterKey = ModuleName

	// QuerierRoute is the querier route for interchain query
	QuerierRoute = ModuleName
)

// PortKey defines the key to store the port ID in store
var PortKey = []byte{0x01}

// ContainsQueryPath returns true if the path is present in allowQueries, otherwise false
func ContainsQueryPath(allowQueries []string, path string) bool {
	for _, v := range allowQueries {
		if v == path {
			return true
		}
	}

	return false
}
