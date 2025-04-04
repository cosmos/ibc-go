package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestIsAllowedRelayer(t *testing.T) {
	testCases := []struct {
		name    string
		relayer sdk.AccAddress
		config  types.Config
		expPass bool
	}{
		{"success: valid relayer with default config", signer1, types.DefaultConfig(), true},
		{"success: valid relayer with custom config", signer1, types.NewConfig(ibctesting.TestAccAddress), true},
		{"success: valid relayer with multiple relayers in config", signer1, types.NewConfig(signer2.String(), ibctesting.TestAccAddress), true},
		{"failure: invalid relayer with custom config", signer2, types.NewConfig(ibctesting.TestAccAddress, signer3.String()), false},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expPass, tc.config.IsAllowedRelayer(tc.relayer), tc.name)
	}
}

func TestValidateConfig(t *testing.T) {
	tooManyRelayers := make([]string, types.MaxAllowedRelayersLength+1)
	for i := range tooManyRelayers {
		tooManyRelayers[i] = ibctesting.TestAccAddress
	}
	testCases := []struct {
		name   string
		config types.Config
		expErr error
	}{
		{
			name:   "default config",
			config: types.DefaultConfig(),
			expErr: nil,
		},
		{
			name:   "custom config",
			config: types.NewConfig(ibctesting.TestAccAddress),
			expErr: nil,
		},
		{
			name:   "multiple relayers",
			config: types.NewConfig(ibctesting.TestAccAddress, signer2.String()),
			expErr: nil,
		},
		{
			name:   "too many allowed relayers",
			config: types.NewConfig(tooManyRelayers...),
			expErr: errors.New("allowed relayers length must not exceed 20 items"),
		},
		{
			name:   "invalid relayer address",
			config: types.NewConfig("invalidAddress"),
			expErr: errors.New("invalid relayer address"),
		},
		{
			name:   "invalid relayer address with valid ones",
			config: types.NewConfig("invalidAddress", ibctesting.TestAccAddress),
			expErr: errors.New("invalid relayer address"),
		},
	}

	for _, tc := range testCases {
		err := tc.config.Validate()
		if tc.expErr == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
			ibctesting.RequireErrorIsOrContains(t, err, tc.expErr, err.Error())
		}
	}
}
