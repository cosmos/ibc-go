package types

import "fmt"

const (
	// ModuleName defines the 29-fee name
	ModuleName = "ibcfee"

	// StoreKey is the store key string for IBC transfer
	StoreKey = ModuleName

	// PortKey is the port id that is wrapped by fee middleware
	PortKey = "feetransfer"

	// RouterKey is the message route for IBC transfer
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC transfer
	QuerierRoute = ModuleName

	// Key prefix for relayer address mapping
	RelayerAddressKeyPrefix = "relayerAddress"
)

// Key for relayer source address -> counteryparty address mapping
func KeyRelayerAddress(address string) []byte {
	return []byte(fmt.Sprintf("%s/%s", RelayerAddressKeyPrefix, address))
}
