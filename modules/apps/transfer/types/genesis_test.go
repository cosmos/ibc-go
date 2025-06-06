package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

func TestValidateGenesis(t *testing.T) {
	testCases := []struct {
		name     string
		genState *types.GenesisState
		expErr   error
	}{
		{
			name:     "default",
			genState: types.DefaultGenesisState(),
			expErr:   nil,
		},
		{
			"valid genesis",
			&types.GenesisState{
				PortId: "portidone",
			},
			nil,
		},
		{
			"invalid client",
			&types.GenesisState{
				PortId: "(INVALIDPORT)",
			},
			host.ErrInvalidID,
		},
	}

	for _, tc := range testCases {
		err := tc.genState.Validate()
		if tc.expErr == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.ErrorIs(t, err, tc.expErr, tc.name)
		}
	}
}
