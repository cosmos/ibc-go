package types_test

import (
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
		tc := tc
		require.Equal(t, tc.expPass, tc.config.IsAllowedRelayer(tc.relayer), tc.name)
	}
}

func TestValidateConfig(t *testing.T) {
	tooManyRelayers := make([]string, types.MaxAllowedRelayersLength+1)
	for i := range tooManyRelayers {
		tooManyRelayers[i] = ibctesting.TestAccAddress
	}
	testCases := []struct {
		name    string
		config  types.Config
		expPass bool
	}{
		{"default config", types.DefaultConfig(), true},
		{"custom config", types.NewConfig(ibctesting.TestAccAddress), true},
		{"multiple relayers", types.NewConfig(ibctesting.TestAccAddress, signer2.String()), true},
		{"too many allowed relayers", types.NewConfig(tooManyRelayers...), false},
		{"invalid relayer address", types.NewConfig("invalidAddress"), false},
		{"invalid relayer address with valid ones", types.NewConfig("invalidAddress", ibctesting.TestAccAddress), false},
	}

	for _, tc := range testCases {
		tc := tc

		err := tc.config.Validate()
		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}
