package types

import (
	"fmt"
)

const (
	// ModuleName defines the IBC rate-limiting name
	ModuleName = "ratelimiting"

	// StoreKey is the store key string for IBC rate-limiting
	StoreKey = ModuleName

	// RouterKey is the message route for IBC rate-limiting
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC rate-limiting
	QuerierRoute = ModuleName

	// PortID is the default port id that rate-limiting module binds to
	PortID = "ratelimiting"

	// Version defines the current version for the rate-limiting module
	Version = "ratelimiting-1"

	// ParamsKey is the store key for rate-limiting module parameters
	ParamsKey = "params"
)

var (
	PortKeyPrefix             = "port"
	PathKeyPrefix             = "path"
	RateLimitKeyPrefix        = "rate-limit"
	PendingSendPacketPrefix   = "pending-send-packet"
	DenomBlacklistKeyPrefix   = "denom-blacklist"
	AddressWhitelistKeyPrefix = "address-blacklist"
	HourEpochKey              = "hour-epoch"

	PendingSendPacketChannelLength = 16
)

func KeyPort(portID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", PortKeyPrefix, portID))
}

// Get the rate limit byte key built from the denom and channelId
func KeyRateLimitItem(denom string, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", RateLimitKeyPrefix, denom, channelID))
}

// Get the pending send packet key from the channel ID and sequence number
// The channel ID must be fixed length to allow for extracting the underlying
// values from a key
func KeyPendingSendPacket(channelId string, sequenceNumber uint64) []byte {
	return []byte(fmt.Sprintf("%s/%d", channelId, sequenceNumber))
}

// Get the whitelist path key from a sender and receiver address
func KeyAddressWhitelist(sender, receiver string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", AddressWhitelistKeyPrefix, sender, receiver))
}
