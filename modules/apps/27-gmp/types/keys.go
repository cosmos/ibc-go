package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the gmp module name
	ModuleName = "gmp"

	// StoreKey is the primary storage key for the gmp module
	StoreKey = ModuleName

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

	// AccountAddrLen is the length of the ICS27 account address
	AccountAddrLen = 32
)

var (
	// AccountsKey is the key used to store the accounts in the keeper
	AccountsKey = collections.NewPrefix(0)
)
