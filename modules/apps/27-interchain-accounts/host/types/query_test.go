package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

func TestIsModuleQuerySafe(t *testing.T) {
	testCases := []struct {
		name              string
		servicePath       string
		isModuleQuerySafe bool
		expErr            error
	}{
		{
			"success: module query safe",
			"cosmos.bank.v1beta1.Query.Balance",
			true,
			nil,
		},
		{
			"success: not module query safe",
			"ibc.applications.transfer.v1.Query.DenomTraces",
			false,
			nil,
		},
		{
			"failure: invalid method path",
			"invalid",
			false,
			errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "invalid query method path"),
		},
		{
			"failure: service path not found",
			"invalid.Query.Balance",
			false,
			errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "failed to find descriptor"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res, err := types.IsModuleQuerySafe(tc.servicePath)
			require.Equal(t, tc.isModuleQuerySafe, res)
			require.ErrorIs(t, err, tc.expErr)
		})
	}
}
