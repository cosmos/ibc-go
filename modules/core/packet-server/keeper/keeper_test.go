package keeper_test

import (
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

const testClientID = "tendermint-0"

func (suite *KeeperTestSuite) TestSetCounterparty() {
	merklePathPrefix := commitmenttypes.NewMerklePath([]byte("ibc"), []byte(""))
	counterparty := types.Counterparty{
		ClientId:         testClientID,
		MerklePathPrefix: merklePathPrefix,
	}
	suite.chainA.App.GetIBCKeeper().PacketServerKeeper.SetCounterparty(suite.chainA.GetContext(), testClientID, counterparty)

	retrievedCounterparty, found := suite.chainA.App.GetIBCKeeper().PacketServerKeeper.GetCounterparty(suite.chainA.GetContext(), testClientID)
	suite.Require().True(found, "GetCounterparty does not return counterparty")
	suite.Require().Equal(counterparty, retrievedCounterparty, "Counterparty retrieved not equal")

	retrievedCounterparty, found = suite.chainA.App.GetIBCKeeper().PacketServerKeeper.GetCounterparty(suite.chainA.GetContext(), "client-0")
	suite.Require().False(found, "GetCounterparty unexpectedly returned a counterparty")
	suite.Require().Equal(types.Counterparty{}, retrievedCounterparty, "Counterparty retrieved not empty")
}
