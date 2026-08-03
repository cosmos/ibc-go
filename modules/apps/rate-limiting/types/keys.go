package types

import (
	"encoding/binary"

	"cosmossdk.io/collections"
)

const (
	// ModuleName defines the IBC rate-limiting name
	ModuleName = "ratelimit"

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
	RateLimitKeyPrefix = bytes("rate-limit")
	// PendingSendPacketPrefix is the legacy pending send packet prefix. It is
	// only used by migrations that clear old pending packet state.
	PendingSendPacketPrefix = bytes("pending-send-packet")
	// PendingReceivePacketPrefix is the legacy pending receive packet prefix. It
	// is only used by migrations that clear old pending packet state.
	PendingReceivePacketPrefix = bytes("pending-receive-packet")
	DenomBlacklistKeyPrefix    = bytes("denom-blacklist")
	// TODO: Fix IBCGO-2368
	AddressWhitelistKeyPrefix = bytes("address-blacklist")
	HourEpochKey              = bytes("hour-epoch")

	PendingSendPacketsKey    = collections.NewPrefix(0)
	PendingReceivePacketsKey = collections.NewPrefix(1)

	PendingSendPacketChannelLength = 64
)

// RateLimitItemKey returns an unambiguous key for a (denom, channelOrClientID) pair:
//
//	uvarint(len(channelOrClientID)) || channelOrClientID || denom
//
// Channel-first ordering groups a channel or client's rate limits by prefix.
func RateLimitItemKey(denom string, channelOrClientID string) []byte {
	return lengthPrefixedKey(channelOrClientID, denom)
}

// Get the pending packet key from the channel ID and sequence number
// The channel ID must be fixed length to allow for extracting the underlying
// values from a key
func PendingPacketKey(channelID string, sequenceNumber uint64) ([]byte, error) {
	if err := validatePendingPacketChannelID(channelID); err != nil {
		return nil, err
	}

	channelIDBz := make([]byte, PendingSendPacketChannelLength)
	copy(channelIDBz, channelID)

	sequenceNumberBz := make([]byte, 8)
	binary.BigEndian.PutUint64(sequenceNumberBz, sequenceNumber)

	return append(channelIDBz, sequenceNumberBz...), nil
}

// AddressWhitelistKey returns an unambiguous key for a (sender, receiver) pair:
//
//	uvarint(len(sender)) || sender || receiver
func AddressWhitelistKey(sender, receiver string) []byte {
	return lengthPrefixedKey(sender, receiver)
}

// lengthPrefixedKey returns uvarint(len(first)) || first || second.
//
// Grown by append rather than sized up front on purpose: a make() capacity
// computed from the argument lengths trips CodeQL go/allocation-size-overflow,
// and bounding that computation with a panic trips beginendblock-panic, since
// BeginBlocker builds these keys and must never halt the chain. The extra
// allocations are noise next to the store read that follows.
func lengthPrefixedKey(first, second string) []byte {
	key := binary.AppendUvarint(nil, uint64(len(first)))
	key = append(key, first...)
	return append(key, second...)
}
