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
}

// SetupTest creates a coordinator with 1 test chain.
func (s *CallbacksTypesTestSuite) SetupSuite() {
	s.coord = ibctesting.NewCoordinator(s.T(), 1)
	s.chain = s.coord.GetChain(ibctesting.GetChainID(1))
}

func TestCallbacksTypesTestSuite(t *testing.T) {
	suite.Run(t, new(CallbacksTypesTestSuite))
}
