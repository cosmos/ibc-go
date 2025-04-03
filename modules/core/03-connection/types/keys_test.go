package types_test

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// tests ParseConnectionSequence and IsValidConnectionID
func TestParseConnectionSequence(t *testing.T) {
	testCases := []struct {
		name         string
		connectionID string
		expSeq       uint64
		expError     error
	}{
		{"valid 0", "connection-0", 0, nil},
		{"valid 1", "connection-1", 1, nil},
		{"valid large sequence", types.FormatConnectionIdentifier(math.MaxUint64), math.MaxUint64, nil},
		// one above uint64 max
		{"invalid uint64", "connection-18446744073709551616", 0, errors.New("invalid connection identifier: failed to parse identifier sequence")},
		// uint64 == 20 characters
		{"invalid large sequence", "connection-2345682193567182931243", 0, host.ErrInvalidID},
		{"capital prefix", "Connection-0", 0, host.ErrInvalidID},
		{"double prefix", "connection-connection-0", 0, host.ErrInvalidID},
		{"missing dash", "connection0", 0, host.ErrInvalidID},
		{"blank id", "               ", 0, host.ErrInvalidID},
		{"empty id", "", 0, host.ErrInvalidID},
		{"negative sequence", "connection--1", 0, host.ErrInvalidID},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			seq, err := types.ParseConnectionSequence(tc.connectionID)
			valid := types.IsValidConnectionID(tc.connectionID)
			require.Equal(t, tc.expSeq, seq)

			if tc.expError == nil {
				require.NoError(t, err, tc.name)
				require.True(t, valid)
			} else {
				require.ErrorContains(t, err, tc.expError.Error())
				require.False(t, valid)
			}
		})
	}
}
