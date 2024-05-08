package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

func TestValidateParams(t *testing.T) {
	testCases := []struct {
		name    string
		params  Params
		expPass bool
	}{
		{"default params", DefaultParams(), true},
		{"custom params", NewParams(exported.Tendermint), true},
		{"blank client", NewParams(" "), false},
<<<<<<< HEAD
=======
		{"duplicate clients", NewParams(exported.Tendermint, exported.Tendermint), false},
		{"allow all clients plus valid client", NewParams(AllowAllClients, exported.Tendermint), false},
		{"too many allowed clients", NewParams(make([]string, MaxAllowedClientsLength+1)...), false},
>>>>>>> 478f4c60 (imp: check length of slices of messages (#6256))
	}

	for _, tc := range testCases {
		err := tc.params.Validate()
		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}
