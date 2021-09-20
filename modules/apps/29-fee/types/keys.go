package types

import "fmt"

const (
	// ModuleName defines the 29-fee name
	ModuleName = "feeibc"

	// StoreKey is the store key string for IBC transfer
	StoreKey = ModuleName

	// PortKey is the port id that is wrapped by fee middleware
	PortKey = "feetransfer"

	// RouterKey is the message route for IBC transfer
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC transfer
	QuerierRoute = ModuleName

	Version = "fee29-1"

	// RelayerAddressKeyPrefix is the key prefix for relayer address mapping
	RelayerAddressKeyPrefix = "relayerAddress"
)

func FeeEnabledKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("fee_enabled/%s/%s", portID, channelID))
}

// KeyRelayerAddress returns the key for relayer address -> counteryparty address mapping
func KeyRelayerAddress(address string) []byte {
	return []byte(fmt.Sprintf("%s/%s", RelayerAddressKeyPrefix, address))
}
