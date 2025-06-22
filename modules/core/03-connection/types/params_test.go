package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
)

func TestValidateParams(t *testing.T) {
	testCases := []struct {
		name     string
		params   types.Params
		expError error
	}{
		{"default params", types.DefaultParams(), nil},
		{"custom params", types.NewParams(10), nil},
		{"blank client", types.NewParams(0), errors.New("MaxExpectedTimePerBlock cannot be zero")},
	}

	for _, tc := range testCases {
		err := tc.params.Validate()
		if tc.expError == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.ErrorContains(t, err, tc.expError.Error())
		}
	}
}
