package types

import (
	"encoding/binary"

	errorsmod "cosmossdk.io/errors"
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
)

func bytes(p string) []byte {
	return []byte(p)
}

var (
	RateLimitKeyPrefix      = bytes("rate-limit")
	PendingSendPacketPrefix = bytes("pending-send-packet")
	DenomBlacklistKeyPrefix = bytes("denom-blacklist")
	// TODO: Fix IBCGO-2368
	AddressWhitelistKeyPrefix = bytes("address-blacklist")
	HourEpochKey              = bytes("hour-epoch")

	PendingSendPacketChannelLength = 64
)

// Get the rate limit byte key built from the denom and channelId
func RateLimitItemKey(denom string, channelID string) []byte {
	return append(bytes(denom), bytes(channelID)...)
}

// Get the pending send packet key from the channel ID and sequence number
// The channel ID must be fixed length to allow for extracting the underlying
// values from a key
func PendingSendPacketKey(channelID string, sequenceNumber uint64) ([]byte, error) {
	if len(channelID) > PendingSendPacketChannelLength {
		return nil, errorsmod.Wrapf(ErrInvalidChannelID, "channel %s with length %d is greater than the allowed length %d", channelID, len(channelID), PendingSendPacketChannelLength)
	}

	channelIDBz := make([]byte, PendingSendPacketChannelLength)
	copy(channelIDBz, channelID)

	sequenceNumberBz := make([]byte, 8)
	binary.BigEndian.PutUint64(sequenceNumberBz, sequenceNumber)

	return append(channelIDBz, sequenceNumberBz...), nil
}

// Get the whitelist path key from a sender and receiver address
func AddressWhitelistKey(sender, receiver string) []byte {
	return append(bytes(sender), bytes(receiver)...)
}
