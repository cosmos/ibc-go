package types_test

import (
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

func (suite *TypesTestSuite) TestFTPD() {
	packetData := types.FungibleTokenPacketData{
		Denom:    "uatom",
		Amount:   "1000000",
		Sender:   "sender",
		Receiver: "receiver",
		Memo:     "memo",
	}

	bz, err := types.EncodeABIFungibleTokenPacketData(&packetData)
	suite.Require().NoError(err)

	decodedPacketData, err := types.DecodeABIFungibleTokenPacketData(bz)
	suite.Require().NoError(err)

	suite.Require().Equal(packetData, *decodedPacketData)
}
