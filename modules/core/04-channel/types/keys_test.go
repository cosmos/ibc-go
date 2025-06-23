package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// tests ParseChannelSequence and IsValidChannelID
func TestParseChannelSequence(t *testing.T) {
	testCases := []struct {
		name      string
		channelID string
		expSeq    uint64
		expErr    error
	}{
		{"valid 0", "channel-0", 0, nil},
		{"valid 1", "channel-1", 1, nil},
		{"valid large sequence", "channel-234568219356718293", 234568219356718293, nil},
		// one above uint64 max
		{"invalid uint64", "channel-18446744073709551616", 0, errors.New("invalid channel identifier: failed to parse identifier sequence")},
		// uint64 == 20 characters
		{"invalid large sequence", "channel-2345682193567182931243", 0, host.ErrInvalidID},
		{"capital prefix", "Channel-0", 0, host.ErrInvalidID},
		{"missing dash", "channel0", 0, host.ErrInvalidID},
		{"blank id", "               ", 0, host.ErrInvalidID},
		{"empty id", "", 0, host.ErrInvalidID},
		{"negative sequence", "channel--1", 0, host.ErrInvalidID},
	}

	for _, tc := range testCases {
		seq, err := types.ParseChannelSequence(tc.channelID)
		valid := types.IsValidChannelID(tc.channelID)
		require.Equal(t, tc.expSeq, seq)

		if tc.expErr == nil {
			require.NoError(t, err, tc.name)
			require.True(t, valid)
		} else {
			require.Error(t, err, tc.name)
			require.False(t, valid)
			require.ErrorContains(t, err, tc.expErr.Error())
		}
	}
}
