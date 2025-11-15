package types_test

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// tests ParseClientIdentifier and IsValidClientID
func TestParseClientIdentifier(t *testing.T) {
	testCases := []struct {
		name       string
		clientID   string
		clientType string
		expSeq     uint64
		expErr     error
	}{
		{"valid 0", "tendermint-0", "tendermint", 0, nil},
		{"valid 1", "tendermint-1", "tendermint", 1, nil},
		{"valid solemachine", "solomachine-v1-1", "solomachine-v1", 1, nil},
		{"valid large sequence", types.FormatClientIdentifier("tendermint", math.MaxUint64), "tendermint", math.MaxUint64, nil},
		{"valid short client type", "t-0", "t", 0, nil},
		// one above uint64 max
		{"invalid uint64", "tendermint-18446744073709551616", "tendermint", 0, errors.New("failed to parse client identifier sequence")},
		// uint64 == 20 characters
		{"invalid large sequence", "tendermint-2345682193567182931243", "tendermint", 0, host.ErrInvalidID},
		{"invalid newline in clientID", "tendermin\nt-1", "tendermin\nt", 0, host.ErrInvalidID},
		{"invalid newline character before dash", "tendermint\n-1", "tendermint", 0, host.ErrInvalidID},
		{"missing dash", "tendermint0", "tendermint", 0, host.ErrInvalidID},
		{"blank id", "               ", "    ", 0, host.ErrInvalidID},
		{"empty id", "", "", 0, host.ErrInvalidID},
		{"negative sequence", "tendermint--1", "tendermint", 0, host.ErrInvalidID},
		{"invalid format", "tendermint-tm", "tendermint", 0, host.ErrInvalidID},
		{"empty clientype", " -100", "tendermint", 0, host.ErrInvalidID},
		{"with in the middle tabs", "a\t\t\t-100", "tendermint", 0, host.ErrInvalidID},
		{"leading tabs", "\t\t\ta-100", "tendermint", 0, host.ErrInvalidID},
		{"with whitespace", "                  a-100", "tendermint", 0, host.ErrInvalidID},
		{"leading hyphens", "-----a-100", "tendermint", 0, host.ErrInvalidID},
		{"with slash", "tendermint/-100", "tendermint", 0, host.ErrInvalidID},
		{"non-ASCII:: emoji", "🚨😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎😎-100", "tendermint", 0, host.ErrInvalidID},
		{"non-ASCII:: others", "世界-100", "tendermint", 0, host.ErrInvalidID},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clientType, seq, err := types.ParseClientIdentifier(tc.clientID)
			valid := types.IsValidClientID(tc.clientID)
			require.Equal(t, tc.expSeq, seq, tc.clientID)

			if tc.expErr == nil {
				require.NoError(t, err, tc.name)
				require.True(t, valid)
				require.Equal(t, tc.clientType, clientType)
			} else {
				require.Error(t, err, tc.name, tc.clientID)
				require.False(t, valid)
				require.Empty(t, clientType)
				require.ErrorContains(t, err, tc.expErr.Error())
			}
		})
	}
}
