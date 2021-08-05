package types

import "fmt"

const (
	// ModuleName defines the 29-fee name
	ModuleName = "ibcfee"

	// StoreKey is the store key string for IBC transfer
	StoreKey = ModuleName

	// RouterKey is the message route for IBC transfer
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC transfer
	QuerierRoute = ModuleName

	Version = "fee29-1"

	KeyAppCapability = "app_capabilities"
)

func AppCapabilityName(channelID, portID string) string {
	return fmt.Sprintf("%s/%s/%s", KeyAppCapability, channelID, portID)
}
