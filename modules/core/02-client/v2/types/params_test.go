package types_test

import (
	"crypto/sha256"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/stretchr/testify/require"
)

func TestIsAllowedRelayer(t *testing.T) {
	signer1 := sdk.MustAccAddressFromBech32(ibctesting.TestAccAddress)
	hash2 := sha256.Sum256([]byte("signer2"))
	signer2 := sdk.AccAddress(hash2[:])
	hash3 := sha256.Sum256([]byte("signer3"))
	signer3 := sdk.AccAddress(hash3[:])
	testCases := []struct {
		name    string
		relayer sdk.AccAddress
		params  types.Params
		expPass bool
	}{
		{"success: valid relayer with default params", signer1, types.DefaultParams(), true},
		{"success: valid relayer with custom params", signer1, types.NewParams(signer1), true},
		{"success: valid relaeyr with multiple relayers in params", signer1, types.NewParams(signer2, signer1), true},
		{"failure: invalid relayer with custom params", signer2, types.NewParams(signer1, signer3), false},
	}

	for _, tc := range testCases {
		tc := tc
		require.Equal(t, tc.expPass, tc.params.IsAllowedRelayer(tc.relayer), tc.name)
	}
}
