package localhost_test

import (
	"testing"

	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/stretchr/testify/suite"
)

type LocalhostTestSuite struct {
	suite.Suite

	coordinator ibctesting.Coordinator
	chain       *ibctesting.TestChain
}

func (s *LocalhostTestSuite) SetupTest() {
	s.coordinator = *ibctesting.NewCoordinator(s.T(), 1)
	s.chain = s.coordinator.GetChain(ibctesting.GetChainID(1))
}

func TestLocalhostTestSuite(t *testing.T) {
	suite.Run(t, new(LocalhostTestSuite))
}
