package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/stretchr/testify/require"
)

func TestIsAllowedRelayer(t *testing.T) {
	testCases := []struct {
		name    string
		relayer sdk.AccAddress
		params  types.Params
		expPass bool
	}{
		{"success: valid relayer with default params", signer1, types.DefaultParams(), true},
		{"success: valid relayer with custom params", signer1, types.NewParams(ibctesting.TestAccAddress), true},
		{"success: valid relaeyr with multiple relayers in params", signer1, types.NewParams(signer2.String(), ibctesting.TestAccAddress), true},
		{"failure: invalid relayer with custom params", signer2, types.NewParams(ibctesting.TestAccAddress, signer3.String()), false},
	}

	for _, tc := range testCases {
		tc := tc
		require.Equal(t, tc.expPass, tc.params.IsAllowedRelayer(tc.relayer), tc.name)
	}
}

func TestValidateParams(t *testing.T) {
	tooManyRelayers := make([]string, types.MaxAllowedRelayersLength+1)
	for i, _ := range tooManyRelayers {
		tooManyRelayers[i] = ibctesting.TestAccAddress
	}
	testCases := []struct {
		name    string
		params  types.Params
		expPass bool
	}{
		{"default params", types.DefaultParams(), true},
		{"custom params", types.NewParams(ibctesting.TestAccAddress), true},
		{"multiple relayers", types.NewParams(ibctesting.TestAccAddress, signer2.String()), true},
		{"too many allowed relayers", types.NewParams(tooManyRelayers...), false},
		{"invalid relayer address", types.NewParams("invalidAddress"), false},
		{"invalid relayer address with valid ones", types.NewParams("invalidAddress", ibctesting.TestAccAddress), false},
	}

	for _, tc := range testCases {
		tc := tc

		err := tc.params.Validate()
		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}
