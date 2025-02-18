package clientv2_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestModuleTestSuite(t *testing.T) {
	testifysuite.Run(t, new(ModuleTestSuite))
}

type ModuleTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (suite *ModuleTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))
}
