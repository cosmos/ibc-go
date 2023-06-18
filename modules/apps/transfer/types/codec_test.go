package types_test

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

// TestMustMarshalProtoJSON tests that the memo field is only emitted (marshalled) if it is populated
func (s *TypesTestSuite) TestMustMarshalProtoJSON() {
	memo := "memo"
	packetData := types.NewFungibleTokenPacketData(sdk.DefaultBondDenom, "1", s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), memo)

	bz := packetData.GetBytes()
	exists := strings.Contains(string(bz), memo)
	s.Require().True(exists)

	packetData.Memo = ""

	bz = packetData.GetBytes()
	exists = strings.Contains(string(bz), memo)
	s.Require().False(exists)
}
