package channel_test

import (
	"testing"

	ibctesting "github.com/cosmos/ibc-go/testing"
	"github.com/stretchr/testify/suite"
)

type ChannelTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	path *ibctesting.Path
}

// SetupTest creates a coordinator with 2 test chains.
func (suite *ChannelTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.Setup(suite.path)
}

// TestChannelTestSuite runs all the tests within this package.
func TestChannelTestSuite(t *testing.T) {
	suite.Run(t, new(ChannelTestSuite))
}
