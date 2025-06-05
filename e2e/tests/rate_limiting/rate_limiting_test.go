//go:build !test_e2e

package ratelimiting

import (
	"context"
	"strings"
	"testing"
	"time"

	ibc "github.com/strangelove-ventures/interchaintest/v8/ibc"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	ratelimitingtypes "github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type RateLimTestSuite struct {
	testsuite.E2ETestSuite
}

func TestRateLimitSuite(t *testing.T) {
	testifysuite.Run(t, new(RateLimTestSuite))
}

func (s *RateLimTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil, func(options *testsuite.ChainOptions) {
		options.RelayerCount = 1
	})
}

func (s *RateLimTestSuite) TestRateLimit() {
	t := s.T()
	ctx := context.TODO()
	testName := t.Name()

	chainA, chainB := s.GetChains()

	userA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	authority, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
	s.Require().NoError(err)
	s.Require().NotNil(authority)

	relayer := s.CreateDefaultPaths(testName)
	s.StartRelayer(relayer, testName)

	chanAB := s.GetChainAToChainBChannel(testName)

	escrowAddrA := transfertypes.GetEscrowAddress(chanAB.PortID, chanAB.ChannelID)
	denomA := chainA.Config().Denom

	ibcTokenB := testsuite.GetIBCToken(denomA, chanAB.PortID, chanAB.ChannelID)

	t.Run("No rate limit set: Tranfer succeed", func(_ *testing.T) {
		userABalBefore, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)
		userBBalBefore, err := query.Balance(ctx, chainB, userB.FormattedAddress(), ibcTokenB.IBCDenom())
		s.Require().NoError(err)
		s.Require().Zero(userBBalBefore.Int64())

		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParseV1PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)

		s.Require().Eventually(func() bool {
			_, err := query.GRPCQuery[chantypes.QueryPacketCommitmentResponse](ctx, chainA, &chantypes.QueryPacketCommitmentRequest{
				PortId:    chanAB.PortID,
				ChannelId: chanAB.ChannelID,
				Sequence:  packet.Sequence,
			})
			return err != nil && strings.Contains(err.Error(), "packet commitment hash not found")
		}, time.Second*70, time.Second)

		userABalAfter, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)

		// Balanced moved form useA to userB
		s.Require().Equal(userABalBefore-testvalues.IBCTransferAmount, userABalAfter)
		escrowBalA, err := query.Balance(ctx, chainA, escrowAddrA.String(), denomA)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, escrowBalA.Int64())

		userBBalAfter, err := query.Balance(ctx, chainB, userB.FormattedAddress(), ibcTokenB.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, userBBalAfter.Int64())
	})

	t.Run("Add Outgoing Ratelimit on ChainA", func(_ *testing.T) {
		resp, err := query.GRPCQuery[ratelimitingtypes.QueryAllRateLimitsResponse](ctx, chainA, &ratelimitingtypes.QueryAllRateLimitsRequest{})
		s.Require().NoError(err)
		s.Require().Nil(resp.RateLimits)

		txResp := s.addRateLimit(ctx, chainA, userA, userA.FormattedAddress(), denomA, chanAB.ChannelID, 10, 0, 1)
		s.AssertTxSuccess(txResp)

		resp, err = query.GRPCQuery[ratelimitingtypes.QueryAllRateLimitsResponse](ctx, chainA, &ratelimitingtypes.QueryAllRateLimitsRequest{})
		s.Require().NoError(err)
		s.Require().Len(resp.RateLimits, 1)

		// Make transfer again and see the flow has been updated.
		userABalBefore, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)

		txResp = s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParseV1PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)

		s.Require().Eventually(func() bool {
			_, err := query.GRPCQuery[chantypes.QueryPacketCommitmentResponse](ctx, chainA, &chantypes.QueryPacketCommitmentRequest{
				PortId:    chanAB.PortID,
				ChannelId: chanAB.ChannelID,
				Sequence:  packet.Sequence,
			})
			return err != nil && strings.Contains(err.Error(), "packet commitment hash not found")
		}, time.Second*70, time.Second)

		userABalAfter, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)

		// Balanced moved form useA to userB
		s.Require().Equal(userABalBefore-testvalues.IBCTransferAmount, userABalAfter)
		userBBalAfter, err := query.Balance(ctx, chainB, userB.FormattedAddress(), ibcTokenB.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(2*testvalues.IBCTransferAmount, userBBalAfter.Int64())

		// Check the flow has been updated.
		rateLimit := s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		s.Require().NotNil(rateLimit)

		s.Require().Equal(rateLimit.Flow.Outflow.Int64(), testvalues.IBCTransferAmount)
	})

	t.Run("Reset RateLimit: Set outgoing to 0 -> Transfet Failed", func(_ *testing.T) {
		txResp := s.resetRateLimit(ctx, chainA, userA, userA.FormattedAddress(), denomA, chanAB.ChannelID)
		s.AssertTxSuccess(txResp)

		rateLimit := s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		s.Require().Zero(rateLimit.Flow.Outflow.Int64())

		txResp = s.updateRateLimit(ctx, chainA, userA, userA.FormattedAddress(), denomA, chanAB.ChannelID)
		s.AssertTxSuccess(txResp)

		txResp = s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxFailure(txResp, ratelimitingtypes.ErrQuotaExceeded)
	})
}

func (s *RateLimTestSuite) rateLimit(ctx context.Context, chain ibc.Chain, denom, chanID string) ratelimitingtypes.RateLimit {
	respRateLim, err := query.GRPCQuery[ratelimitingtypes.QueryRateLimitResponse](ctx, chain, &ratelimitingtypes.QueryRateLimitRequest{
		Denom:             denom,
		ChannelOrClientId: chanID,
	})
	s.Require().NoError(err)
	return *respRateLim.RateLimit
}

func (s *RateLimTestSuite) addRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, sender, denom, chanID string, sendPercent, recvPercent, duration int64) sdk.TxResponse {
	msg := &ratelimitingtypes.MsgAddRateLimit{
		Signer:            sender,
		Denom:             denom,
		ChannelOrClientId: chanID,
		MaxPercentSend:    sdkmath.NewInt(sendPercent),
		MaxPercentRecv:    sdkmath.NewInt(recvPercent),
		DurationHours:     uint64(duration),
	}
	return s.BroadcastMessages(ctx, chain, user, msg)
}

func (s *RateLimTestSuite) resetRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, sender, denom, chanID string) sdk.TxResponse {
	msg := &ratelimitingtypes.MsgResetRateLimit{
		Signer:            sender,
		Denom:             denom,
		ChannelOrClientId: chanID,
	}
	return s.BroadcastMessages(ctx, chain, user, msg)
}

func (s *RateLimTestSuite) updateRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, sender, denom, chanID string) sdk.TxResponse {
	msg := &ratelimitingtypes.MsgUpdateRateLimit{
		Signer:            sender,
		Denom:             denom,
		ChannelOrClientId: chanID,
		MaxPercentSend:    sdkmath.ZeroInt(), // From 10% to 0%
		MaxPercentRecv:    sdkmath.OneInt(),  // One of Send or Receive needs to be > 0
		DurationHours:     1,
	}
	return s.BroadcastMessages(ctx, chain, user, msg)
}
