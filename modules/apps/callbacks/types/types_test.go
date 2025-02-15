package types_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// CallbacksTypesTestSuite defines the needed instances and methods to test callbacks
type CallbacksTypesTestSuite struct {
	suite.Suite

	coord *ibctesting.Coordinator

	chainA, chainB *ibctesting.TestChain

	path *ibctesting.Path
}

// SetupTest creates a coordinator with 2 test chains.
func (s *CallbacksTypesTestSuite) SetupTest() {
	s.coord = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coord.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coord.GetChain(ibctesting.GetChainID(2))
	s.path = ibctesting.NewPath(s.chainA, s.chainB)
}

func TestCallbacksTypesTestSuite(t *testing.T) {
	suite.Run(t, new(CallbacksTypesTestSuite))
}
