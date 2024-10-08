package keeper_test

import (
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
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

	// Counterparty not yet stored for another client.
	retrievedCounterparty, found = suite.chainA.App.GetIBCKeeper().PacketServerKeeper.GetCounterparty(suite.chainA.GetContext(), ibctesting.SecondClientID)
	suite.Require().False(found, "GetCounterparty unexpectedly returned a counterparty")
	suite.Require().Equal(types.Counterparty{}, retrievedCounterparty, "Counterparty retrieved not empty")
}

func (suite *KeeperTestSuite) TestSetCreator() {
	clientID := ibctesting.FirstClientID
	expectedCreator := "test-creator"

	// Set the creator for the client
	suite.chainA.App.GetIBCKeeper().PacketServerKeeper.SetCreator(suite.chainA.GetContext(), clientID, expectedCreator)

	// Retrieve the creator from the store
	retrievedCreator, found := suite.chainA.App.GetIBCKeeper().PacketServerKeeper.GetCreator(suite.chainA.GetContext(), clientID)

	// Verify that the retrieved creator matches the expected creator
	suite.Require().True(found, "GetCreator did not return stored creator")
	suite.Require().Equal(expectedCreator, retrievedCreator, "Creator is not retrieved correctly")

	// Verify non stored creator is not found
	retrievedCreator, found = suite.chainA.App.GetIBCKeeper().PacketServerKeeper.GetCreator(suite.chainA.GetContext(), ibctesting.SecondClientID)
	suite.Require().False(found, "GetCreator unexpectedly returned a creator")
	suite.Require().Empty(retrievedCreator, "Creator is not empty")

	// Verify that the creator is deleted from the store
	suite.chainA.App.GetIBCKeeper().PacketServerKeeper.DeleteCreator(suite.chainA.GetContext(), clientID)
	retrievedCreator, found = suite.chainA.App.GetIBCKeeper().PacketServerKeeper.GetCreator(suite.chainA.GetContext(), clientID)
	suite.Require().False(found, "GetCreator unexpectedly returned a creator")
	suite.Require().Empty(retrievedCreator, "Creator is not empty")
}
