package types

const (
	// ModuleName defines the interchain accounts host module name
	ModuleName = "icahost"

	// PortID is the default port id that the interchain accounts module binds to
	// PortID = "ibcaccount"

	// StoreKey is the store key string for interchain accounts
	StoreKey = ModuleName

	// RouterKey is the message route for interchain accounts
	RouterKey = ModuleName

	// QuerierRoute is the querier route for interchain accounts
	QuerierRoute = ModuleName
)
