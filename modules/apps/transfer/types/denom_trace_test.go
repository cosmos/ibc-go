package types_test

import (
	"testing"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/stretchr/testify/require"
)

func TestDenomTrace_IBCDenom(t *testing.T) {
	testCases := []struct {
		name     string
		trace    types.DenomTrace
		expDenom string
	}{
		{"base denom", types.DenomTrace{BaseDenom: "uatom"}, "uatom"},
		{"trace info", types.DenomTrace{BaseDenom: "uatom", Path: "transfer/channel-1"}, "ibc/C4CFF46FD6DE35CA4CF4CE031E643C8FDC9BA4B99AE598E9B0ED98FE3A2319F9"},
	}

	for _, tc := range testCases {
		tc := tc

		denom := tc.trace.IBCDenom()
		require.Equal(t, tc.expDenom, denom, tc.name)
	}
}
