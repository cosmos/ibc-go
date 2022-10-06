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

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.Bech32Address(chainB.Config().Bech32Prefix)

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

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp, err := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0)
		s.Require().NoError(err)
		s.AssertValidTxResponse(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
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
