package connection

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/strangelove-ventures/ibctest/v6/test"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	connectiontypes "github.com/cosmos/ibc-go/v6/modules/core/03-connection/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

func TestConnectionTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionTestSuite))
}

type ConnectionTestSuite struct {
	testsuite.E2ETestSuite
}

// QueryMaxExpectedTimePerBlockParam queries the on-chain max expected time per block param for 03-connection
func (s *ConnectionTestSuite) QueryMaxExpectedTimePerBlockParam(ctx context.Context, chain ibc.Chain) uint64 {
	queryClient := s.GetChainGRCPClients(chain).ParamsQueryClient
	res, err := queryClient.Params(ctx, &paramsproposaltypes.QueryParamsRequest{
		Subspace: host.ModuleName,
		Key:      string(connectiontypes.KeyMaxExpectedTimePerBlock),
	})
	s.Require().NoError(err)

	// removing additional strings that are used for amino
	delay := strings.ReplaceAll(res.Param.Value, "\"", "")
	time, err := strconv.ParseUint(delay, 10, 64)
	s.Require().NoError(err)

	return time
}

// TestMaxExpectedTimePerBlockParam tests changing the MaxExpectedTimePerBlock param using a governance proposal
func (s *ConnectionTestSuite) TestMaxExpectedTimePerBlockParam() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()

	chainBDenom := chainB.Config().Denom
	chainAIBCToken := testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.Bech32Address(chainB.Config().Bech32Prefix)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("ensure delay is set to the default of 30 seconds", func(t *testing.T) {
		expectedDelay := uint64(30 * time.Second)
		delay := s.QueryMaxExpectedTimePerBlockParam(ctx, chainA)
		s.Require().Equal(expectedDelay, delay)
	})

	t.Run("change the delay to 60 seconds", func(t *testing.T) {
		delay := fmt.Sprintf(`"%d"`, 1*time.Minute)
		changes := []paramsproposaltypes.ParamChange{
			paramsproposaltypes.NewParamChange(host.ModuleName, string(connectiontypes.KeyMaxExpectedTimePerBlock), delay),
		}

		proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
		s.ExecuteGovProposal(ctx, chainA, chainAWallet, proposal)
	})

	t.Run("validate the param was successfully changed", func(t *testing.T) {
		expectedDelay := uint64(1 * time.Minute)
		delay := s.QueryMaxExpectedTimePerBlockParam(ctx, chainA)
		s.Require().Equal(expectedDelay, delay)
	})

	t.Run("ensure packets can be received, send from chainB to chainA", func(t *testing.T) {
		t.Run("send tokens from chainB to chainA", func(t *testing.T) {
			transferTxResp, err := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
			s.Require().NoError(err)
			s.AssertValidTxResponse(transferTxResp)
		})

		t.Run("tokens are escrowed", func(t *testing.T) {
			actualBalance, err := s.GetChainBNativeBalance(ctx, chainBWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("start relayer", func(t *testing.T) {
			s.StartRelayer(relayer)
		})

		t.Run("packets are relayed", func(t *testing.T) {
			s.AssertPacketRelayed(ctx, chainA, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

			actualBalance, err := chainA.GetBalance(ctx, chainAAddress, chainAIBCToken.IBCDenom())
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("stop relayer", func(t *testing.T) {
			s.StopRelayer(ctx, relayer)
		})
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
