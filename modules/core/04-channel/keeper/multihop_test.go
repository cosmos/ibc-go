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

	// A is the first endpoint in a multihop channel
	A ibctesting.EndpointM
	// Z is the first endpoint in a multihop channel
	Z ibctesting.EndpointM
}

// SetupTest is run before each test method in the suite
func (s *MultihopTestSuite) SetupTest() {
	s.coordinator, s.paths = ibctesting.CreateLinkedChains(&s.Suite, 5)
	s.A, s.Z = ibctesting.NewEndpointMFromLinkedPaths(s.paths)
}

// TestMultihopMultihopTestSuite runs all multihop related tests.
func TestMultihopTestSuite(t *testing.T) {
	suite.Run(t, new(MultihopTestSuite))
}
