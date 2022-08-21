package types

const (
	ModuleName   = "queryibc"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

var (
	// QueryKey defines the key to store the query in store
	QueryKey = []byte{0x01}
	// QueryResultKey defines the key to store query result in store
	QueryResultKey = []byte{0x02}
)
