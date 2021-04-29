package types

const (
	// ModuleName defines the CCV child module name
	ModuleName = "child"

	// PortID is the default port id that child module binds to
	PortID = "child"

	// StoreKey is the store key string for IBC child
	StoreKey = ModuleName

	// RouterKey is the message route for IBC child
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC child
	QuerierRoute = ModuleName
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = []byte{0x01}
)
