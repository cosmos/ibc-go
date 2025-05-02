package types

const (
	// ModuleName defines the interchain accounts module name
	ModuleName = "gmp"

	// PortID is the default IBC port id that the gmp module
	PortID = "gmpport"

	// Version defines the current version for gmp
	Version = "ics27-2"

	// RouterKey is the message route for gmp
	RouterKey = ModuleName

	// QuerierRoute is the querier route for gmp
	QuerierRoute = ModuleName

	// accountsKey is the key used when generating a module address for the gmp module
	accountsKey = "gmp-accounts"
)
