package e2e

import (
	"os"
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
	image, ok := os.LookupEnv("SIMD_IMAGE")
	s.Require().True(ok)
	s.T().Logf("SIMD_IMAGE=%s", image)

	s.T().Logf("Placeholder test")
	s.Require().True(true)
}
