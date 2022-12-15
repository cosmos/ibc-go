package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// MultihopTestSuite is a testing suite to test keeper functions.
type MultihopTestSuite struct {
	suite.Suite
}

// TestMultihopMultihopTestSuite runs all multihop related tests.
func TestMultihopTestSuite(t *testing.T) {
	suite.Run(t, new(MultihopTestSuite))
}
