//go:build !test_e2e

package connection

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/interchaintest/v10/ibc"
	test "github.com/cosmos/interchaintest/v10/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// compatibility:from_version: v7.10.0
func TestConnectionTestSuite(t *testing.T) {
	testifysuite.Run(t, new(ConnectionTestSuite))
}

type ConnectionTestSuite struct {
	testsuite.E2ETestSuite
}

// SetupSuite sets up chains for the current test suite
func (s *ConnectionTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil)
}

// QueryMaxExpectedTimePerBlockParam queries the on-chain max expected time per block param for 03-connection
func (s *ConnectionTestSuite) QueryMaxExpectedTimePerBlockParam(ctx context.Context, chain ibc.Chain) uint64 {
	if testvalues.SelfParamsFeatureReleases.IsSupported(chain.Config().Images[0].Version) {
		res, err := query.GRPCQuery[connectiontypes.QueryConnectionParamsResponse](ctx, chain, &connectiontypes.QueryConnectionParamsRequest{})
		s.Require().NoError(err)

		return res.Params.MaxExpectedTimePerBlock
	}
	res, err := query.GRPCQuery[paramsproposaltypes.QueryParamsResponse](ctx, chain, &paramsproposaltypes.QueryParamsRequest{
		Subspace: ibcexported.ModuleName,
		Key:      string(connectiontypes.KeyMaxExpectedTimePerBlock),
	})
	s.Require().NoError(err)

	// removing additional strings that are used for amino
	delay := strings.ReplaceAll(res.Param.Value, "\"", "")
	// convert to uint64
	uinttime, err := strconv.ParseUint(delay, 10, 64)
	s.Require().NoError(err)

	return uinttime
}

// TestMaxExpectedTimePerBlockParam tests changing the MaxExpectedTimePerBlock param using a governance proposal
func (s *ConnectionTestSuite) TestMaxExpectedTimePerBlockParam() {
	t := s.T()
	ctx := context.TODO()
	testName := t.Name()

	chainA, chainB := s.GetChains()
	chainAVersion := chainA.Config().Images[0].Version

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainBDenom := chainB.Config().Denom
	chainAIBCToken := testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("ensure delay is set to the default of 30 seconds", func(t *testing.T) {
		delay := s.QueryMaxExpectedTimePerBlockParam(ctx, chainA)
		s.Require().Equal(uint64(connectiontypes.DefaultTimePerBlock), delay)
	})

	t.Run("change the delay to 60 seconds", func(t *testing.T) {
		delay := uint64(1 * time.Minute)
		if testvalues.SelfParamsFeatureReleases.IsSupported(chainAVersion) {
			authority, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
			s.Require().NoError(err)
			s.Require().NotNil(authority)

			msg := connectiontypes.NewMsgUpdateParams(authority.String(), connectiontypes.NewParams(delay))
			s.ExecuteAndPassGovV1Proposal(ctx, msg, chainA, chainAWallet)
		} else {
			changes := []paramsproposaltypes.ParamChange{
				paramsproposaltypes.NewParamChange(ibcexported.ModuleName, string(connectiontypes.KeyMaxExpectedTimePerBlock), fmt.Sprintf(`"%d"`, delay)),
			}

			proposal := paramsproposaltypes.NewParameterChangeProposal(ibctesting.Title, ibctesting.Description, changes)
			s.ExecuteAndPassGovV1Beta1Proposal(ctx, chainA, chainAWallet, proposal)
		}
	})

	t.Run("validate the param was successfully changed", func(t *testing.T) {
		expectedDelay := uint64(1 * time.Minute)
		delay := s.QueryMaxExpectedTimePerBlockParam(ctx, chainA)
		s.Require().Equal(expectedDelay, delay)
	})

	t.Run("ensure packets can be received, send from chainB to chainA", func(t *testing.T) {
		t.Run("send tokens from chainB to chainA", func(t *testing.T) {
			transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
			s.AssertTxSuccess(transferTxResp)
		})

		t.Run("tokens are escrowed", func(t *testing.T) {
			actualBalance, err := s.GetChainBNativeBalance(ctx, chainBWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("start relayer", func(t *testing.T) {
			s.StartRelayer(relayer, testName)
		})

		t.Run("packets are relayed", func(t *testing.T) {
			s.AssertPacketRelayed(ctx, chainA, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

			actualBalance, err := query.Balance(ctx, chainA, chainAAddress, chainAIBCToken.IBCDenom())

			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance.Int64())
		})

		t.Run("stop relayer", func(t *testing.T) {
			s.StopRelayer(ctx, relayer)
		})
	})
}
