package types_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

// TestRateLimitItemKeyLayout pins the on-disk key bytes, which are state
// machine breaking to change. The uvarint length prefix of "channel-1" (9
// bytes) is the single byte 0x09: every length below 128 encodes to one byte.
func TestRateLimitItemKeyLayout(t *testing.T) {
	require.Equal(t, []byte("\x09channel-1uatom"), types.RateLimitItemKey("uatom", "channel-1"))
}

// TestRateLimitItemKeyTwoByteLength pins the layout at 128, the first
// identifier length whose uvarint prefix is two bytes (0x80 0x01), so a
// fixed-width single-byte length prefix cannot silently pass.
func TestRateLimitItemKeyTwoByteLength(t *testing.T) {
	longID := strings.Repeat("c", 128)
	expected := append([]byte("\x80\x01"), longID...)
	expected = append(expected, "uatom"...)
	require.Equal(t, expected, types.RateLimitItemKey("uatom", longID))
}

// TestRateLimitItemKeyUnambiguous covers a pair that collided under the old
// denom-first, separator-less layout, and a pair that would collide under a
// channel-first layout without the length prefix.
func TestRateLimitItemKeyUnambiguous(t *testing.T) {
	require.NotEqual(
		t,
		types.RateLimitItemKey("uatom", "channel-1"),
		types.RateLimitItemKey("uatomchannel", "-1"),
	)
	require.NotEqual(
		t,
		types.RateLimitItemKey("", "channel-10"),
		types.RateLimitItemKey("0", "channel-1"),
	)
}

func TestRateLimitItemKeyChannelFirst(t *testing.T) {
	// All rate limits for a channel share the channel-scoped prefix.
	prefix := types.RateLimitItemKey("", "channel-1")
	require.True(t, bytes.HasPrefix(types.RateLimitItemKey("uatom", "channel-1"), prefix))
	require.False(t, bytes.HasPrefix(types.RateLimitItemKey("uatom", "channel-10"), prefix))
}

// TestAddressWhitelistKeyLayout pins the on-disk key bytes, which are state
// machine breaking to change. The uvarint length prefix of "sender" (6 bytes)
// is the single byte 0x06.
func TestAddressWhitelistKeyLayout(t *testing.T) {
	require.Equal(t, []byte("\x06senderreceiver"), types.AddressWhitelistKey("sender", "receiver"))
}

// TestAddressWhitelistKeyUnambiguous covers a pair that collided under the old
// sender||receiver separator-less layout.
func TestAddressWhitelistKeyUnambiguous(t *testing.T) {
	require.NotEqual(
		t,
		types.AddressWhitelistKey("cosmos1a", "bcosmos1c"),
		types.AddressWhitelistKey("cosmos1ab", "cosmos1c"),
	)
}
