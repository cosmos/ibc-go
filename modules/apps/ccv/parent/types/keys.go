package types

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
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = []byte{0x01}
)
