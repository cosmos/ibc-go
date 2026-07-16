//go:build !test_e2e

package ratelimiting

import (
	"context"
	"testing"

	interchaintest "github.com/cosmos/interchaintest/v11"
	"github.com/cosmos/interchaintest/v11/ibc"
	"github.com/cosmos/interchaintest/v11/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	pfmtypes "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/types"
	ratelimitingtypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
)

type RateLimTestSuite struct {
	testsuite.E2ETestSuite
}

func TestRateLimitSuite(t *testing.T) {
	testifysuite.Run(t, new(RateLimTestSuite))
}

func (s *RateLimTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 3, nil)
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

	relayer := s.GetRelayerForTest(testName)
	s.CreatePath(ctx, relayer, chainA, chainB, ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	s.StartRelayer(relayer, testName)
	defer s.StopRelayer(ctx, relayer)

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

		respRateLim, err := query.GRPCQuery[ratelimitingtypes.QueryRateLimitResponse](ctx, chainA, &ratelimitingtypes.QueryRateLimitRequest{
			Denom:             denomA,
			ChannelOrClientId: chanAB.ChannelID,
		})
		s.Require().NoError(err)
		s.Require().Nil(respRateLim.RateLimit)

		// Transfer works again
		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(txResp)
	})
}

func (s *RateLimTestSuite) TestRateLimitWithPFM() {
	t := s.T()
	ctx := context.TODO()
	testName := t.Name()

	chains := s.GetAllChains()
	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	userA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	userC := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)

	authorityB, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chainB)
	s.Require().NoError(err)
	s.Require().NotNil(authorityB)

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)

	chanAB := s.GetChannelBetweenChains(testName, chainA, chainB)
	chanBC := s.GetChannelBetweenChains(testName, chainB, chainC)

	denomA := chainA.Config().Denom
	transferAmount := testvalues.DefaultTransferAmount(denomA)
	seedAmount := sdk.NewInt64Coin(denomA, 10*testvalues.IBCTransferAmount)
	ibcTokenB := testsuite.GetIBCToken(denomA, chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID)
	ibcTokenC := testsuite.GetIBCToken(ibcTokenB.Path(), chanBC.Counterparty.PortID, chanBC.Counterparty.ChannelID)

	seedTxResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, seedAmount, userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
	s.AssertTxSuccess(seedTxResp)
	s.flushPacketsOnChannel(ctx, relayer, chainA, chainB, chanAB.ChannelID)
	s.flushPacketsOnChannel(ctx, relayer, chainA, chainB, chanAB.ChannelID)

	userBBalance, err := query.Balance(ctx, chainB, userB.FormattedAddress(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)
	s.Require().Equal(seedAmount.Amount.Int64(), userBBalance.Int64())

	s.addRateLimit(ctx, chainB, userB, ibcTokenB.IBCDenom(), chanAB.Counterparty.ChannelID, authorityB.String(), 100, 100, 1)

	t.Run("successful async ack keeps inflow", func(_ *testing.T) {
		inflowBefore := s.rateLimit(ctx, chainB, ibcTokenB.IBCDenom(), chanAB.Counterparty.ChannelID).Flow.Inflow.Int64()

		memo := s.forwardMemo(userC.FormattedAddress(), chanBC)
		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, transferAmount, userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, memo)
		s.AssertTxSuccess(txResp)

		s.flushPacketsOnChannel(ctx, relayer, chainA, chainB, chanAB.ChannelID)
		s.Require().Equal(inflowBefore+testvalues.IBCTransferAmount, s.rateLimit(ctx, chainB, ibcTokenB.IBCDenom(), chanAB.Counterparty.ChannelID).Flow.Inflow.Int64())

		s.flushPacketsOnChannel(ctx, relayer, chainB, chainC, chanBC.ChannelID)
		s.flushPacketsOnChannel(ctx, relayer, chainB, chainC, chanBC.ChannelID)
		s.Require().Equal(inflowBefore+testvalues.IBCTransferAmount, s.rateLimit(ctx, chainB, ibcTokenB.IBCDenom(), chanAB.Counterparty.ChannelID).Flow.Inflow.Int64())

		s.flushPacketsOnChannel(ctx, relayer, chainA, chainB, chanAB.ChannelID)

		balanceC, err := query.Balance(ctx, chainC, userC.FormattedAddress(), ibcTokenC.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, balanceC.Int64())
	})

	t.Run("failing async ack undoes inflow", func(_ *testing.T) {
		inflowBefore := s.rateLimit(ctx, chainB, ibcTokenB.IBCDenom(), chanAB.Counterparty.ChannelID).Flow.Inflow.Int64()
		balanceABefore, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)

		memo := s.forwardMemo("invalid-receiver", chanBC)
		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, transferAmount, userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, memo)
		s.AssertTxSuccess(txResp)

		s.flushPacketsOnChannel(ctx, relayer, chainA, chainB, chanAB.ChannelID)
		s.Require().Equal(inflowBefore+testvalues.IBCTransferAmount, s.rateLimit(ctx, chainB, ibcTokenB.IBCDenom(), chanAB.Counterparty.ChannelID).Flow.Inflow.Int64())

		s.flushPacketsOnChannel(ctx, relayer, chainB, chainC, chanBC.ChannelID)
		s.flushPacketsOnChannel(ctx, relayer, chainB, chainC, chanBC.ChannelID)
		s.Require().Equal(inflowBefore, s.rateLimit(ctx, chainB, ibcTokenB.IBCDenom(), chanAB.Counterparty.ChannelID).Flow.Inflow.Int64())

		s.flushPacketsOnChannel(ctx, relayer, chainA, chainB, chanAB.ChannelID)

		balanceAAfter, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)
		s.Require().Equal(balanceABefore, balanceAAfter)
	})
}

func (s *RateLimTestSuite) rateLimit(ctx context.Context, chain ibc.Chain, denom, chanID string) *ratelimitingtypes.RateLimit {
	respRateLim, err := query.GRPCQuery[ratelimitingtypes.QueryRateLimitResponse](ctx, chain, &ratelimitingtypes.QueryRateLimitRequest{
		Denom:             denom,
		ChannelOrClientId: chanID,
	})
	s.Require().NoError(err)
	s.Require().NotNil(respRateLim.RateLimit, "rate limit not found for denom %s and channel ID %s", denom, chanID)
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

func (s *RateLimTestSuite) forwardMemo(receiver string, channel ibc.ChannelOutput) string {
	metadata := pfmtypes.PacketMetadata{
		Forward: pfmtypes.ForwardMetadata{
			Receiver: receiver,
			Channel:  channel.ChannelID,
			Port:     channel.PortID,
		},
	}

	memo, err := metadata.ToMemo()
	s.Require().NoError(err)
	return memo
}

func (s *RateLimTestSuite) flushPacketsOnChannel(ctx context.Context, relayer ibc.Relayer, chainA, chainB ibc.Chain, channelID string) {
	err := relayer.Flush(ctx, s.GetRelayerExecReporter(), s.GetPathByChains(chainA, chainB), channelID)
	s.Require().NoError(err)
	s.Require().NoError(testutil.WaitForBlocks(ctx, 1, chainA, chainB))
}
