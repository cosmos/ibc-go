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

// RateLimitItemKey returns the rate limit key for a (denom, channelOrClientID)
// pair. The layout is:
//
//	uvarint(len(channelOrClientID)) || channelOrClientID || denom
//
// Channel/client first so that all rate limits for a channel or client are a
// contiguous range (the same channel-then-denom field order as the pending
// packet collections, though not the same byte order: the uvarint prefix
// groups identifiers by length before comparing bytes, while
// collections.StringKey sorts lexicographically), and length prefixed so that
// distinct pairs can never concatenate to the same key.
//
// The uvarint is self-delimiting, so the encoding stays unambiguous for any
// identifier length, with no ceiling to enforce. Nothing in this module bounds
// identifier length (msgs only shape-check channel and client IDs, and genesis
// does not validate them), but every length below 128 encodes to the same
// single byte a fixed-width uint8 prefix would produce.
func RateLimitItemKey(denom string, channelOrClientID string) []byte {
	key := binary.AppendUvarint(make([]byte, 0, 1+len(channelOrClientID)+len(denom)), uint64(len(channelOrClientID)))
	key = append(key, channelOrClientID...)
	return append(key, denom...)
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

// AddressWhitelistKey returns the whitelist key for a (sender, receiver)
// address pair. The layout is:
//
//	uvarint(len(sender)) || sender || receiver
//
// Length prefixed so that distinct pairs can never concatenate to the same
// key; the uvarint is self-delimiting, so the encoding stays unambiguous for
// any address length.
func AddressWhitelistKey(sender, receiver string) []byte {
	key := binary.AppendUvarint(make([]byte, 0, 1+len(sender)+len(receiver)), uint64(len(sender)))
	key = append(key, sender...)
	return append(key, receiver...)
}
