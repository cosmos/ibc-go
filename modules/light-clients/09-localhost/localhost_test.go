package localhost_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type LocalhostTestSuite struct {
	testifysuite.Suite

	coordinator ibctesting.Coordinator
	chain       *ibctesting.TestChain
}

func (s *LocalhostTestSuite) SetupTest() {
	s.coordinator = *ibctesting.NewCoordinator(s.T(), 1)
	s.chain = s.coordinator.GetChain(ibctesting.GetChainID(1))
}

func TestLocalhostTestSuite(t *testing.T) {
	testifysuite.Run(t, new(LocalhostTestSuite))
}
