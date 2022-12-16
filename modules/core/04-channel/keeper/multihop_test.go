package keeper_test

import (
	"testing"

	ibctesting "github.com/cosmos/ibc-go/v6/testing"
	"github.com/stretchr/testify/suite"
)

// MultihopTestSuite is a testing suite to test keeper functions.
type MultihopTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator
	paths       ibctesting.LinkedPaths
}

// SetupTest is run before each test method in the suite
func (s *MultihopTestSuite) SetupTest() {
	s.coordinator, s.paths = ibctesting.CreateLinkedChains(&s.Suite, 5)
}

// TestMultihopMultihopTestSuite runs all multihop related tests.
func TestMultihopTestSuite(t *testing.T) {
	suite.Run(t, new(MultihopTestSuite))
}
