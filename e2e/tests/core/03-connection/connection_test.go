package connection

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v5/modules/apps/transfer/types"
	connectiontypes "github.com/cosmos/ibc-go/v5/modules/core/03-connection/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
)

func TestConnectionTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionTestSuite))
}

type ConnectionTestSuite struct {
	testsuite.E2ETestSuite
}

// QueryConnectionEnabledParam queries the on-chain connection enabled param for 03-connection
func (s *ConnectionTestSuite) QueryMaxExpectedTimePerBlockParam(ctx context.Context, chain ibc.Chain) uint64 {
	queryClient := s.GetChainGRCPClients(chain).ParamsQueryClient
	res, err := queryClient.Params(ctx, &paramsproposaltypes.QueryParamsRequest{
		Subspace: "ibc",
		Key:      string(connectiontypes.KeyMaxExpectedTimePerBlock),
	})
	s.Require().NoError(err)

	// removing additional strings that are used for amino
	delay := strings.ReplaceAll(res.Param.Value, "\"", "")
	time, err := strconv.ParseUint(delay, 10, 64)
	s.Require().NoError(err)

	return time
}

// TestMaxExpectedTimePerBlock tests changing the MaxExpectedTimePerBlock param using a governance proposal
func (s *ConnectionTestSuite) TestMaxExpectedTimePerBlock() {
	t := s.T()
	ctx := context.TODO()

	_, _ = s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()
	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("ensure delay is set to the default of 30 seconds", func(t *testing.T) {
		expectedDelay := fmt.Sprintf("\"%d\"", 30*time.Second)
		delay := fmt.Sprintf("\"%d\"", s.QueryMaxExpectedTimePerBlockParam(ctx, chainA))
		s.Require().Equal(expectedDelay, delay)
	})

	t.Run("change the delay to 60 seconds", func(t *testing.T) {
		delay := fmt.Sprintf("\"%d\"", 60*time.Second)
		changes := []paramsproposaltypes.ParamChange{
			paramsproposaltypes.NewParamChange("ibc", string(connectiontypes.KeyMaxExpectedTimePerBlock), delay),
		}

		proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
		s.ExecuteGovProposal(ctx, chainA, chainAWallet, proposal)
	})

	t.Run("validate the param was successfully changed", func(t *testing.T) {
		expectedDelay := fmt.Sprintf("\"%d\"", 60*time.Second)
		delay := s.QueryMaxExpectedTimePerBlockParam(ctx, chainA)
		s.Require().Equal(expectedDelay, fmt.Sprintf("\"%d\"", delay))
	})
}

// transferChannelOptions configures both of the chains to have non-incentivized transfer channels.
func transferChannelOptions() func(options *ibc.CreateChannelOptions) {
	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = transfertypes.Version
		opts.SourcePortName = transfertypes.PortID
		opts.DestPortName = transfertypes.PortID
	}
}
