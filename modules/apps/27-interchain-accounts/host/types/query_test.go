package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestIsModuleQuerySafe(t *testing.T) {
	testCases := []struct {
		name              string
		servicePath       string
		isModuleQuerySafe bool
		expErr            error
	}{
		{
			"success",
			"cosmos.bank.v1beta1.Query.Balance",
			true,
			nil,
		},
		{
			"failure",
			"ibc.applications.transfer.v1.Query.DenomTraces",
			false,
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_ = ibctesting.NewCoordinator(t, 1)

			res, err := types.IsModuleQuerySafe(tc.servicePath)
			require.Equal(t, tc.isModuleQuerySafe, res)
			require.Equal(t, tc.expErr, err)
		})
	}
}
