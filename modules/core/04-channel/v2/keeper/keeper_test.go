package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypes2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
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

			counterparty, found := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2.AliasV1Channel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			if tc.expPass {
				suite.Require().True(found)

				merklePath := commitmentv2types.NewMerklePath([]byte("ibc"), []byte(""))
				expCounterparty := channeltypes2.NewCounterparty(path.EndpointA.ClientID, path.EndpointB.ChannelID, merklePath)
				suite.Require().Equal(counterparty, expCounterparty)
			} else {
				suite.Require().False(found)
				suite.Require().Equal(counterparty, channeltypes2.Counterparty{})
			}
		})
	}
}
