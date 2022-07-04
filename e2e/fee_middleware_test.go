package e2e

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestFeeMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(FeeMiddlewareTestSuite))
}

type FeeMiddlewareTestSuite struct {
	suite.Suite
}

func (s *FeeMiddlewareTestSuite) TestPlaceholder() {
	s.T().Logf("Placeholder test")
	s.Require().True(true)
}
