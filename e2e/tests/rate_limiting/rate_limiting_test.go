//go:build !test_e2e

package ratelimiting

import (
	"context"
	"testing"

	interchaintest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	ratelimitingtypes "github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
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

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	s.StartRelayer(relayer, testName)

	chanAB := s.GetChannelBetweenChains(testName, chainA, chainB)

	escrowAddrA := transfertypes.GetEscrowAddress(chanAB.PortID, chanAB.ChannelID)
	denomA := chainA.Config().Denom

	ibcTokenB := testsuite.GetIBCToken(denomA, chanAB.PortID, chanAB.ChannelID)

	t.Run("No rate limit set: transfer succeeds", func(_ *testing.T) {
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

		s.Require().NoError(testutil.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")
		s.AssertPacketRelayed(ctx, chainA, chanAB.PortID, chanAB.ChannelID, packet.Sequence)

		userABalAfter, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)

		// Balanced moved form userA to userB
		s.Require().Equal(userABalBefore-testvalues.IBCTransferAmount, userABalAfter)
		escrowBalA, err := query.Balance(ctx, chainA, escrowAddrA.String(), denomA)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, escrowBalA.Int64())

		userBBalAfter, err := query.Balance(ctx, chainB, userB.FormattedAddress(), ibcTokenB.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, userBBalAfter.Int64())
	})

	t.Run("Add outgoing rate limit on ChainA", func(_ *testing.T) {
		resp, err := query.GRPCQuery[ratelimitingtypes.QueryAllRateLimitsResponse](ctx, chainA, &ratelimitingtypes.QueryAllRateLimitsRequest{})
		s.Require().NoError(err)
		s.Require().Nil(resp.RateLimits)

		sendPercentage := int64(10)
		recvPercentage := int64(0)
		s.addRateLimit(ctx, chainA, userA, denomA, chanAB.ChannelID, authority.String(), sendPercentage, recvPercentage, 1)

		resp, err = query.GRPCQuery[ratelimitingtypes.QueryAllRateLimitsResponse](ctx, chainA, &ratelimitingtypes.QueryAllRateLimitsRequest{})
		s.Require().NoError(err)
		s.Require().Len(resp.RateLimits, 1)

		rateLimit := resp.RateLimits[0]
		s.Require().Equal(int64(0), rateLimit.Flow.Outflow.Int64())
		s.Require().Equal(int64(0), rateLimit.Flow.Inflow.Int64())
		s.Require().Equal(rateLimit.Quota.MaxPercentSend.Int64(), sendPercentage)
		s.Require().Equal(rateLimit.Quota.MaxPercentRecv.Int64(), recvPercentage)
		s.Require().Equal(uint64(1), rateLimit.Quota.DurationHours)
	})

	t.Run("Transfer updates the rate limit flow", func(_ *testing.T) {
		userABalBefore, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)

		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParseV1PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)

		s.Require().NoError(testutil.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")
		s.AssertPacketRelayed(ctx, chainA, chanAB.PortID, chanAB.ChannelID, packet.Sequence)

		userABalAfter, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)

		// Balanced moved form userA to userB
		s.Require().Equal(userABalBefore-testvalues.IBCTransferAmount, userABalAfter)
		userBBalAfter, err := query.Balance(ctx, chainB, userB.FormattedAddress(), ibcTokenB.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(2*testvalues.IBCTransferAmount, userBBalAfter.Int64())

		// Check the flow has been updated.
		rateLimit := s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		s.Require().NotNil(rateLimit)
		s.Require().Equal(testvalues.IBCTransferAmount, rateLimit.Flow.Outflow.Int64())
	})

	t.Run("Fill and exceed quota", func(_ *testing.T) {
		rateLim := s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		sendPercentage := rateLim.Quota.MaxPercentSend.Int64()

		// Create an account that can almost exhause the outflow limit.
		richKidAmt := rateLim.Flow.ChannelValue.MulRaw(sendPercentage).QuoRaw(100).Sub(rateLim.Flow.Outflow)
		richKid := interchaintest.GetAndFundTestUsers(t, ctx, "richkid", richKidAmt, chainA)[0]
		s.Require().NoError(testutil.WaitForBlocks(ctx, 4, chainA))

		sendCoin := sdk.NewCoin(denomA, richKidAmt)

		// Fill the quota
		txResp := s.Transfer(ctx, chainA, richKid, chanAB.PortID, chanAB.ChannelID, sendCoin, richKid.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(txResp)

		// Sending even 10denomA fails due to exceeding the quota
		sendCoin = sdk.NewInt64Coin(denomA, 10)
		txResp = s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, sendCoin, userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxFailure(txResp, ratelimitingtypes.ErrQuotaExceeded)
	})

	t.Run("Reset rate limit: transfer succeeds", func(_ *testing.T) {
		rateLimit := s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		sendPercentage := rateLimit.Quota.MaxPercentSend.Int64()
		recvPercentage := rateLimit.Quota.MaxPercentRecv.Int64()

		s.resetRateLimit(ctx, chainA, userA, denomA, chanAB.ChannelID, authority.String())

		rateLimit = s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		// Resetting only clears the flow. It does not change the quota
		s.Require().Zero(rateLimit.Flow.Outflow.Int64())
		s.Require().Equal(rateLimit.Quota.MaxPercentSend.Int64(), sendPercentage)
		s.Require().Equal(rateLimit.Quota.MaxPercentRecv.Int64(), recvPercentage)

		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(txResp)
	})

	t.Run("Set outflow quota to 0: transfer fails", func(_ *testing.T) {
		sendPercentage := int64(0)
		recvPercentage := int64(1)
		s.updateRateLimit(ctx, chainA, userA, denomA, chanAB.ChannelID, authority.String(), sendPercentage, recvPercentage)

		rateLimit := s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		s.Require().Equal(rateLimit.Quota.MaxPercentSend.Int64(), sendPercentage)
		s.Require().Equal(rateLimit.Quota.MaxPercentRecv.Int64(), recvPercentage)

		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxFailure(txResp, ratelimitingtypes.ErrQuotaExceeded)
	})

	t.Run("Remove rate limit -> transfer succeeds again", func(_ *testing.T) {
		s.removeRateLimit(ctx, chainA, userA, denomA, chanAB.ChannelID, authority.String())

		rateLimit := s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		s.Require().Nil(rateLimit)

		// Transfer works again
		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(txResp)
	})
}

func (s *RateLimTestSuite) rateLimit(ctx context.Context, chain ibc.Chain, denom, chanID string) *ratelimitingtypes.RateLimit {
	respRateLim, err := query.GRPCQuery[ratelimitingtypes.QueryRateLimitResponse](ctx, chain, &ratelimitingtypes.QueryRateLimitRequest{
		Denom:             denom,
		ChannelOrClientId: chanID,
	})
	s.Require().NoError(err)
	return respRateLim.RateLimit
}

func (s *RateLimTestSuite) addRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, denom, chanID, authority string, sendPercent, recvPercent, duration int64) {
	msg := &ratelimitingtypes.MsgAddRateLimit{
		Signer:            authority,
		Denom:             denom,
		ChannelOrClientId: chanID,
		MaxPercentSend:    sdkmath.NewInt(sendPercent),
		MaxPercentRecv:    sdkmath.NewInt(recvPercent),
		DurationHours:     uint64(duration),
	}
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, user)
}

func (s *RateLimTestSuite) resetRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, denom, chanID, authority string) {
	msg := &ratelimitingtypes.MsgResetRateLimit{
		Signer:            authority,
		Denom:             denom,
		ChannelOrClientId: chanID,
	}
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, user)
}

func (s *RateLimTestSuite) updateRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, denom, chanID, authority string, sendPercent, recvPercent int64) {
	msg := &ratelimitingtypes.MsgUpdateRateLimit{
		Signer:            authority,
		Denom:             denom,
		ChannelOrClientId: chanID,
		MaxPercentSend:    sdkmath.NewInt(sendPercent),
		MaxPercentRecv:    sdkmath.NewInt(recvPercent),
		DurationHours:     1,
	}
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, user)
}

func (s *RateLimTestSuite) removeRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, denom, chanID, authority string) {
	msg := &ratelimitingtypes.MsgRemoveRateLimit{
		Signer:            authority,
		Denom:             denom,
		ChannelOrClientId: chanID,
	}
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, user)
}
