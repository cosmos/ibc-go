package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

// KeeperTestSuite is a testing suite to test keeper functions.
type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

// TestKeeperTestSuite runs all the tests within this package.
func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

// SetupTest creates a coordinator with 2 test chains.
func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
}

// TODO: Remove, just testing the testing setup.
func (suite *KeeperTestSuite) TestCreateEurekaClients() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.SetupV2()

	// Assert counterparty set and creator deleted
	_, found := suite.chainA.App.GetPacketServer().ClientKeeper.GetCounterparty(suite.chainA.GetContext(), path.EndpointA.ClientID)
	suite.Require().True(found)

	// Assert counterparty set and creator deleted
	_, found = suite.chainB.App.GetPacketServer().ClientKeeper.GetCounterparty(suite.chainB.GetContext(), path.EndpointB.ClientID)
	suite.Require().True(found)
}
