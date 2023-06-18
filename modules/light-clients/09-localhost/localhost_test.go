package localhost_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type LocalhostTestSuite struct {
	suite.Suite

	coordinator ibctesting.Coordinator
	chain       *ibctesting.TestChain
}

func (suite *LocalhostTestSuite) SetupTest() {
	suite.coordinator = *ibctesting.NewCoordinator(suite.T(), 1)
	suite.chain = suite.coordinator.GetChain(ibctesting.GetChainID(1))
}

func TestLocalhostTestSuite(t *testing.T) {
	suite.Run(t, new(LocalhostTestSuite))
}
