package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"

	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
)

func TestIsModuleQuerySafe(t *testing.T) {
	testCases := []struct {
		name              string
		servicePath       string
		isModuleQuerySafe bool
	}{
		{
			"success: module query safe",
			"/cosmos.bank.v1beta1.Query/Balance",
			true,
		},
		{
			"success: not module query safe",
			"/ibc.applications.transfer.v1.Query/DenomTraces",
			false,
		},
		{
			"failure: invalid service path",
			"invalid",
			false,
		},
		{
			"failure: invalid method path",
			"/invalid",
			false,
		},
		{
			"failure: service path not found",
			"/invalid.Query/Balance",
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			logger := log.NewTestLogger(t)
			res := types.IsModuleQuerySafe(logger, tc.servicePath)
			require.Equal(t, tc.isModuleQuerySafe, res)
		})
	}
}
