package types_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// CallbacksTestSuite defines the needed instances and methods to test callbacks
type CallbacksTypesTestSuite struct {
	suite.Suite

	coord *ibctesting.Coordinator

	chain *ibctesting.TestChain

	chainA, chainB *ibctesting.TestChain

	path *ibctesting.Path
}

// SetupTest creates a coordinator with 1 test chain.
func (s *CallbacksTypesTestSuite) SetupTest() {
	s.coord = ibctesting.NewCoordinator(s.T(), 3)
	s.chain = s.coord.GetChain(ibctesting.GetChainID(1))
	s.chainA = s.coord.GetChain(ibctesting.GetChainID(2))
	s.chainB = s.coord.GetChain(ibctesting.GetChainID(3))
	s.path = ibctesting.NewPath(s.chainA, s.chainB)
}

func TestCallbacksTypesTestSuite(t *testing.T) {
	suite.Run(t, new(CallbacksTypesTestSuite))
}
