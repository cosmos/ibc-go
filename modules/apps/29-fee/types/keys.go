package types

import "fmt"

const (
	// ModuleName defines the 29-fee name
	ModuleName = "feeibc"

	// StoreKey is the store key string for IBC transfer
	StoreKey = ModuleName

	// RouterKey is the message route for IBC transfer
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC transfer
	QuerierRoute = ModuleName

	Version = "fee29-1"

	KeyAppCapability = "app_capabilities"

	// RelayerAddressKeyPrefix is the key prefix for relayer address mapping
	RelayerAddressKeyPrefix = "relayerAddress"
)

func AppCapabilityName(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s", KeyAppCapability, channelID, portID)
}

// KeyRelayerAddress returns the key for relayer address -> counteryparty address mapping
func KeyRelayerAddress(address string) []byte {
	return []byte(fmt.Sprintf("%s/%s", RelayerAddressKeyPrefix, address))
}
