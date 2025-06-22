package ibctesting_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type CoordinatorTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (s *CoordinatorTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestCoordinatorTestSuite(t *testing.T) {
	testifysuite.Run(t, new(CoordinatorTestSuite))
}

func (s *CoordinatorTestSuite) TestChainCodecRootResolveNotSet() {
	resolved, err := s.chainA.Codec.InterfaceRegistry().Resolve("/")
	s.Require().Error(err, "Root typeUrl should not be resolvable: %T", resolved)

	resolved, err = s.chainB.Codec.InterfaceRegistry().Resolve("/")
	s.Require().Error(err, "Root typeUrl should not be resolvable: %T", resolved)
}
