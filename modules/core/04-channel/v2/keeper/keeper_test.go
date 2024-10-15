package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypes2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	commitmentv2types "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))
}

func (suite *KeeperTestSuite) TestAliasV1Channel() {
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"failure: channel not found",
			func() {
				path.EndpointA.ChannelID = ""
			},
			false,
		},
		{
			"failure: channel not OPEN",
			func() {
				path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.TRYOPEN })
			},
			false,
		},
		{
			"failure: channel is ORDERED",
			func() {
				path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.Ordering = types.ORDERED })
			},
			false,
		},
		{
			"failure: connection not found",
			func() {
				path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.ConnectionHops = []string{ibctesting.InvalidID} })
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// create a previously existing path on chainA to change the identifiers
			// between the path between chainA and chainB
			path1 := ibctesting.NewPath(suite.chainA, suite.chainC)
			path1.Setup()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			tc.malleate()

			channel, found := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2.AliasV1Channel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			if tc.expPass {
				suite.Require().True(found)

				merklePath := commitmentv2types.NewMerklePath([]byte("ibc"), []byte(""))
				expChannel := channeltypes2.NewChannel(path.EndpointA.ClientID, path.EndpointB.ChannelID, merklePath)
				suite.Require().Equal(channel, expChannel)
			} else {
				suite.Require().False(found)
				suite.Require().Equal(channel, channeltypes2.Channel{})
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSetChannel() {
	merklePathPrefix := commitmenttypes.NewMerklePath([]byte("ibc"), []byte(""))
	channel := channeltypes2.Channel{
		ClientId:         ibctesting.FirstClientID,
		MerklePathPrefix: merklePathPrefix,
	}
	suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetChannel(suite.chainA.GetContext(), ibctesting.FirstChannelID, channel)

	retrievedChannel, found := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetChannel(suite.chainA.GetContext(), ibctesting.FirstChannelID)
	suite.Require().True(found, "GetChannel does not return channel")
	suite.Require().Equal(channel, retrievedChannel, "Channel retrieved not equal")

	// No channel stored under other channel identifier.
	retrievedChannel, found = suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetChannel(suite.chainA.GetContext(), ibctesting.SecondChannelID)
	suite.Require().False(found, "GetChannel unexpectedly returned a channel")
	suite.Require().Equal(channeltypes2.Channel{}, retrievedChannel, "Channel retrieved not empty")
}

func (suite *KeeperTestSuite) TestSetCreator() {
	expectedCreator := "test-creator"

	// Set the creator for the client
	suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetCreator(suite.chainA.GetContext(), ibctesting.FirstChannelID, expectedCreator)

	// Retrieve the creator from the store
	retrievedCreator, found := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetCreator(suite.chainA.GetContext(), ibctesting.FirstChannelID)

	// Verify that the retrieved creator matches the expected creator
	suite.Require().True(found, "GetCreator did not return stored creator")
	suite.Require().Equal(expectedCreator, retrievedCreator, "Creator is not retrieved correctly")

	// Verify that the creator is deleted from the store
	suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeleteCreator(suite.chainA.GetContext(), ibctesting.FirstChannelID)
	retrievedCreator, found = suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetCreator(suite.chainA.GetContext(), ibctesting.FirstChannelID)
	suite.Require().False(found, "GetCreator unexpectedly returned a creator")
	suite.Require().Empty(retrievedCreator, "Creator is not empty")

	// Verify non stored creator is not found
	retrievedCreator, found = suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetCreator(suite.chainA.GetContext(), ibctesting.SecondChannelID)
	suite.Require().False(found, "GetCreator unexpectedly returned a creator")
	suite.Require().Empty(retrievedCreator, "Creator is not empty")
}
