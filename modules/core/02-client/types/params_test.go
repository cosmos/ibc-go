package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

func TestIsAllowedClient(t *testing.T) {
	testCases := []struct {
		name       string
		clientType string
		params     types.Params
		expPass    bool
	}{
		{"success: valid client", exported.Tendermint, types.DefaultParams(), true},
		{"success: valid client with custom params", exported.Tendermint, types.NewParams(exported.Tendermint), true},
		{"success: invalid blank client", " ", types.DefaultParams(), false},
		{"success: invalid client with custom params", exported.Localhost, types.NewParams(exported.Tendermint), false},
		{"success: wildcard allow all clients", "test-client-type", types.NewParams(types.AllowAllClients), true},
		{"success: wildcard allow all clients with blank client", " ", types.NewParams(types.AllowAllClients), false},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expPass, tc.params.IsAllowedClient(tc.clientType), tc.name)
	}
}

func TestValidateParams(t *testing.T) {
	testCases := []struct {
		name     string
		params   types.Params
		expError error
	}{
		{"default params", types.DefaultParams(), nil},
		{"custom params", types.NewParams(exported.Tendermint), nil},
		{"blank client", types.NewParams(" "), errors.New("client type 0 cannot be blank")},
		{"duplicate clients", types.NewParams(exported.Tendermint, exported.Tendermint), errors.New("duplicate client type: 07-tendermint")},
		{"allow all clients plus valid client", types.NewParams(types.AllowAllClients, exported.Tendermint), errors.New("allow list must have only one element because the allow all clients wildcard (*) is present")},
		{"too many allowed clients", types.NewParams(make([]string, types.MaxAllowedClientsLength+1)...), errors.New("allowed clients length must not exceed 200 items")},
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
