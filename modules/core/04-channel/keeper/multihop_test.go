package keeper_test

import (
	"testing"

	ibctesting "github.com/cosmos/ibc-go/v6/testing"
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
func (s *MultihopTestSuite) SetupTest() {
	coord, paths := ibctesting.CreateLinkedChains(&s.Suite, 5)
	s.chanPath = paths.ToPathM()
	s.coord = &ibctesting.CoordinatorM{Coordinator: coord}
}

// A returns the one endpoint of the multihop channel.
func (s *MultihopTestSuite) A() *ibctesting.EndpointM {
	return s.chanPath.EndpointA
}

// Z returns the other endpoint of the multihop channel.
func (s *MultihopTestSuite) Z() *ibctesting.EndpointM {
	return s.chanPath.EndpointZ
}
