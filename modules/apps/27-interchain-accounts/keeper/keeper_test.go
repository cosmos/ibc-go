package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

func NewICAPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = types.PortID
	path.EndpointB.ChannelConfig.PortID = types.PortID

	return path
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestIsBound() {
	isBound := suite.chainA.GetSimApp().ICAKeeper.IsBound(suite.chainA.GetContext(), types.PortID)
	suite.Require().True(isBound)
}

func (suite *KeeperTestSuite) TestGetPort() {
	port := suite.chainA.GetSimApp().ICAKeeper.GetPort(suite.chainA.GetContext())
	suite.Require().Equal(types.PortID, port)
}
