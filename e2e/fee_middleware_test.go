package e2e

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/stretchr/testify/suite"

	"e2e/testsuite"
)

func TestFeeMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(FeeMiddlewareTestSuite))
}

type FeeMiddlewareTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *FeeMiddlewareTestSuite) TestPlaceholder() {
	ctx := context.Background()
	r := s.SetupChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
	s.T().Run("start relayer", func(t *testing.T) {
		s.StartRelayer(r)
	})
}

// feeMiddlewareChannelOptions configures both of the chains to have fee middleware enabled.
func feeMiddlewareChannelOptions() func(options *ibc.CreateChannelOptions) {
	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}"
		opts.DestPortName = "transfer"
		opts.SourcePortName = "transfer"
	}
}
