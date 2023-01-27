package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

const (
	denom              = "transfer/gaiachannel/atom"
	amount             = "100"
	largeAmount        = "18446744073709551616"                                                           // one greater than largest uint64 (^uint64(0))
	invalidLargeAmount = "115792089237316195423570985008687907853269984665640564039457584007913129639936" // 2^256
)

// TestFungibleTokenPacketDataValidateBasic tests ValidateBasic for FungibleTokenPacketData
func TestFungibleTokenPacketDataValidateBasic(t *testing.T) {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketData
		expPass    bool
	}{
		{"valid packet", types.NewFungibleTokenPacketData(denom, amount, addr1, addr2, ""), true},
		{"valid packet with memo", types.NewFungibleTokenPacketData(denom, amount, addr1, addr2, "memo"), true},
		{"valid packet with large amount", types.NewFungibleTokenPacketData(denom, largeAmount, addr1, addr2, ""), true},
		{"invalid denom", types.NewFungibleTokenPacketData("", amount, addr1, addr2, ""), false},
		{"invalid empty amount", types.NewFungibleTokenPacketData(denom, "", addr1, addr2, ""), false},
		{"invalid zero amount", types.NewFungibleTokenPacketData(denom, "0", addr1, addr2, ""), false},
		{"invalid negative amount", types.NewFungibleTokenPacketData(denom, "-1", addr1, addr2, ""), false},
		{"invalid large amount", types.NewFungibleTokenPacketData(denom, invalidLargeAmount, addr1, addr2, ""), false},
		{"missing sender address", types.NewFungibleTokenPacketData(denom, amount, emptyAddr, addr2, ""), false},
		{"missing recipient address", types.NewFungibleTokenPacketData(denom, amount, addr1, emptyAddr, ""), false},
	}

	for i, tc := range testCases {
		err := tc.packetData.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %v", i, err)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}
