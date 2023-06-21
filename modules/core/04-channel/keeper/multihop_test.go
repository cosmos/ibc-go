package keeper_test

import (
	"testing"

	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/stretchr/testify/suite"
)

// TestMultihopMultihopTestSuite runs all multihop related tests.
func TestMultihopTestSuite(t *testing.T) {
	suite.Run(t, new(MultihopTestSuite))
}

// MultihopTestSuite is a testing suite to test keeper functions.
type MultihopTestSuite struct {
	suite.Suite
	// multihop channel path
	chanPath *ibctesting.PathM
	coord    *ibctesting.CoordinatorM
}

// SetupTest is run before each test method in the suite
// No IBC connections or channels are created.
func (suite *MultihopTestSuite) SetupTest(numChains int) {
	coord, paths := ibctesting.CreateLinkedChains(&suite.Suite, numChains)
	suite.chanPath = paths.ToPathM()
	suite.coord = &ibctesting.CoordinatorM{Coordinator: coord}
}

// SetupConnections creates connections between each pair of chains in the multihop path.
func (s *MultihopTestSuite) SetupConnections() {
	s.coord.SetupConnections(s.chanPath)
}

// SetupConnections creates connections between each pair of chains in the multihop path.
func (s *MultihopTestSuite) SetupAllButTheSpecifiedConnection(index int) {
	s.coord.SetupAllButTheSpecifiedConnection(s.chanPath, index)
}

// SetupChannels create a multihop channel after creating all its preprequisites in order, ie. clients, connections.
func (s *MultihopTestSuite) SetupChannels() {
	s.coord.SetupChannels(s.chanPath)
}

// A returns the one endpoint of the multihop channel.
func (s *MultihopTestSuite) A() *ibctesting.EndpointM {
	return s.chanPath.EndpointA
}

// Z returns the other endpoint of the multihop channel.
func (s *MultihopTestSuite) Z() *ibctesting.EndpointM {
	return s.chanPath.EndpointZ
}
