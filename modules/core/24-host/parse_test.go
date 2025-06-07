package host_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestParseIdentifier(t *testing.T) {
	testCases := []struct {
		name       string
		identifier string
		prefix     string
		expSeq     uint64
		expErr     error
	}{
		{"valid 0", "connection-0", "connection-", 0, nil},
		{"valid 1", "connection-1", "connection-", 1, nil},
		{"valid large sequence", connectiontypes.FormatConnectionIdentifier(math.MaxUint64), "connection-", math.MaxUint64, nil},
		// one above uint64 max
		{"invalid uint64", "connection-18446744073709551616", "connection-", 0, errors.New("the value '18446744073709551616' cannot be parsed as a valid uint64")},
		// uint64 == 20 characters
		{"invalid large sequence", "connection-2345682193567182931243", "connection-", 0, errors.New("the sequence number '2345682193567182931243' exceeds the valid range for a uint64")},
		{"capital prefix", "Connection-0", "connection-", 0, errors.New("the prefix 'Connection' should be in lowercase")},
		{"double prefix", "connection-connection-0", "connection-", 0, errors.New("only a single 'connection-' prefix is allowed")},
		{"doesn't have prefix", "connection-0", "prefix", 0, errors.New("the connection ID is missing the required prefix 'connection-'")},
		{"missing dash", "connection0", "connection-", 0, errors.New("the connection ID is missing the dash ('-') between the prefix 'connection' and the sequence number")},
		{"blank id", "               ", "connection-", 0, errors.New("invalid blank connection ID")},
		{"empty id", "", "connection-", 0, errors.New("invalid empty connection id")},
		{"negative sequence", "connection--1", "connection-", 0, errors.New("the sequence number '-1' is negative and invalid")},
	}

	for _, tc := range testCases {
		seq, err := host.ParseIdentifier(tc.identifier, tc.prefix)
		require.Equal(t, tc.expSeq, seq)

		if tc.expErr == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}

func TestMustParseClientStatePath(t *testing.T) {
	testCases := []struct {
		name   string
		path   string
		expErr error
	}{
		{"valid", string(host.FullClientStateKey(ibctesting.FirstClientID)), nil},
		{"path too large", fmt.Sprintf("clients/clients/%s/clientState", ibctesting.FirstClientID), errors.New("path exceeds maximum allowed length for a client state path")},
		{"path too small", fmt.Sprintf("clients/%s", ibctesting.FirstClientID), errors.New("path is shorter than the minimum allowed length")},
		{"path does not begin with client store", fmt.Sprintf("cli/%s/%s", ibctesting.FirstClientID, host.KeyClientState), errors.New("the path must start with 'clients/' but starts with 'cli/'")},
		{"path does not end with client state key", fmt.Sprintf("%s/%s/consensus", string(host.KeyClientStorePrefix), ibctesting.FirstClientID), errors.New("the path must end with the client state key 'clientState'")},
		{"client ID is empty", string(host.FullClientStateKey("")), errors.New("the client ID is empty, which is invalid")},
		{"client ID is only spaces", string(host.FullClientStateKey("   ")), errors.New("Ensure the client ID is not empty and does not contain only spaces")},
	}

	for _, tc := range testCases {
		if tc.expErr == nil {
			require.NotPanics(t, func() {
				clientID := host.MustParseClientStatePath(tc.path)
				require.Equal(t, ibctesting.FirstClientID, clientID)
			})
		} else {
			require.Panics(t, func() {
				host.MustParseClientStatePath(tc.path)
			})
		}
	}
}

func TestMustParseConnectionPath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected string
		expErr   error
	}{
		{"valid", "a/connection", "connection", nil},
		{"valid localhost", "/connection-localhost", "connection-localhost", nil},
		{"invalid empty path", "", "", errors.New("path cannot be empty")},
	}

	for _, tc := range testCases {
		if tc.expErr == nil {
			require.NotPanics(t, func() {
				connID := host.MustParseConnectionPath(tc.path)
				require.Equal(t, tc.expected, connID)
			})
		} else {
			require.Panics(t, func() {
				host.MustParseConnectionPath(tc.path)
			})
		}
	}
}
