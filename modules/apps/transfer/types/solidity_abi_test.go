package types_test

import (
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

func (s *TypesTestSuite) TestFTPD() {
	packetData := types.FungibleTokenPacketData{
		Denom:    "uatom",
		Amount:   "1000000",
		Sender:   "sender",
		Receiver: "receiver",
		Memo:     "memo",
	}

	bz, err := types.EncodeABIFungibleTokenPacketData(&packetData)
	s.Require().NoError(err)

	decodedPacketData, err := types.DecodeABIFungibleTokenPacketData(bz)
	s.Require().NoError(err)

	s.Require().Equal(packetData, *decodedPacketData)
}
