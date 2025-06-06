package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

func TestIsAllowedClient(t *testing.T) {
	testCases := []struct {
		name       string
		clientType string
		params     Params
		expPass    bool
	}{
		{"success: valid client", exported.Tendermint, DefaultParams(), true},
		{"success: valid client with custom params", exported.Tendermint, NewParams(exported.Tendermint), true},
		{"success: invalid blank client", " ", DefaultParams(), false},
		{"success: invalid client with custom params", exported.Localhost, NewParams(exported.Tendermint), false},
		{"success: wildcard allow all clients", "test-client-type", NewParams(AllowAllClients), true},
		{"success: wildcard allow all clients with blank client", " ", NewParams(AllowAllClients), false},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expPass, tc.params.IsAllowedClient(tc.clientType), tc.name)
	}
}

func TestValidateParams(t *testing.T) {
	testCases := []struct {
		name     string
		params   Params
		expError error
	}{
		{"default params", DefaultParams(), nil},
		{"custom params", NewParams(exported.Tendermint), nil},
		{"blank client", NewParams(" "), errors.New("client type 0 cannot be blank")},
		{"duplicate clients", NewParams(exported.Tendermint, exported.Tendermint), errors.New("duplicate client type: 07-tendermint")},
		{"allow all clients plus valid client", NewParams(AllowAllClients, exported.Tendermint), errors.New("allow list must have only one element because the allow all clients wildcard (*) is present")},
		{"too many allowed clients", NewParams(make([]string, MaxAllowedClientsLength+1)...), errors.New("allowed clients length must not exceed 200 items")},
	}

	for _, tc := range testCases {
		err := tc.params.Validate()
		if tc.expError == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
			require.ErrorContains(t, err, tc.expError.Error())
		}
	}
}
