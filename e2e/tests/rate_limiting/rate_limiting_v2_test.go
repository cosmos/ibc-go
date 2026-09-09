//go:build !test_e2e

package ratelimiting

import (
	"context"
	"testing"
	"time"

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
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

type RateLimV2TestSuite struct {
	testsuite.E2ETestSuite
}

func TestRateLimitV2Suite(t *testing.T) {
	testifysuite.Run(t, new(RateLimV2TestSuite))
}

func (s *RateLimV2TestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil, func(options *testsuite.ChainOptions) {
		options.RelayerCount = 1
	})
}

func (s *RateLimV2TestSuite) TestRateLimitV2HappyPath() {
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

	denomA := chainA.Config().Denom

	// Test IBC v2 transfer with rate limiting
	t.Run("IBC v2 transfer with rate limiting: happy path", func(_ *testing.T) {
		// Set up a rate limit for the channel
		sendPercentage := int64(10) // 10% of total supply
		recvPercentage := int64(10) // 10% of total supply
		s.addRateLimit(ctx, chainA, userA, denomA, chanAB.ChannelID, authority.String(), sendPercentage, recvPercentage, 1)

		// Verify rate limit was set
		rateLimit := s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		s.Require().NotNil(rateLimit)
		s.Require().Equal(rateLimit.Quota.MaxPercentSend.Int64(), sendPercentage)
		s.Require().Equal(rateLimit.Quota.MaxPercentRecv.Int64(), recvPercentage)

		// Create IBC v2 transfer using aliasing
		transferAmount := testvalues.DefaultTransferAmount(denomA)

		// Create IBC v2 transfer message with UseAliasing = true
		timeoutTimestamp := uint64(time.Now().Add(time.Hour).UnixNano())
		msgTransfer := transfertypes.NewMsgTransferWithEncoding(
			chanAB.PortID,
			chanAB.ChannelID,
			transferAmount,
			userA.FormattedAddress(),
			userB.FormattedAddress(),
			clienttypes.Height{}, // IBC v2 requires timeoutHeight to be zero
			timeoutTimestamp,
			"",      // memo
			"proto", // encoding
			true,    // useAliasing - this makes it IBC v2
		)

		// Broadcast the IBC v2 transfer
		txResp := s.BroadcastMessages(ctx, chainA, userA, msgTransfer)
		s.AssertTxSuccess(txResp)

		// Verify the transfer was successful by checking balances
		s.Require().NoError(testutil.WaitForBlocks(ctx, 2, chainA, chainB))

		// Check that the rate limit flow was updated
		rateLimitAfter := s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		s.Require().NotNil(rateLimitAfter)
		s.Require().Equal(transferAmount.Amount.Int64(), rateLimitAfter.Flow.Outflow.Int64())
	})

	t.Run("IBC v2 transfer exceeds rate limit: should fail", func(_ *testing.T) {
		// Reset the rate limit to a very small amount
		sendPercentage := int64(1) // 1% of total supply
		recvPercentage := int64(1) // 1% of total supply
		s.updateRateLimit(ctx, chainA, userA, denomA, chanAB.ChannelID, authority.String(), sendPercentage, recvPercentage)

		// Verify rate limit was updated
		rateLimit := s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		s.Require().NotNil(rateLimit)
		s.Require().Equal(rateLimit.Quota.MaxPercentSend.Int64(), sendPercentage)

		// Try to send a large amount that should exceed the rate limit
		largeAmount := sdk.NewInt64Coin(denomA, 1000000) // Large amount

		timeoutTimestamp := uint64(time.Now().Add(time.Hour).UnixNano())
		msgTransfer := transfertypes.NewMsgTransferWithEncoding(
			chanAB.PortID,
			chanAB.ChannelID,
			largeAmount,
			userA.FormattedAddress(),
			userB.FormattedAddress(),
			clienttypes.Height{}, // IBC v2 requires timeoutHeight to be zero
			timeoutTimestamp,
			"",      // memo
			"proto", // encoding
			true,    // useAliasing - this makes it IBC v2
		)

		// This transfer should fail due to rate limiting
		txResp := s.BroadcastMessages(ctx, chainA, userA, msgTransfer)
		s.AssertTxFailure(txResp, ratelimitingtypes.ErrQuotaExceeded)
	})

	t.Run("IBC v2 transfer after rate limit reset: should succeed", func(_ *testing.T) {
		// Reset the rate limit flow
		s.resetRateLimit(ctx, chainA, userA, denomA, chanAB.ChannelID, authority.String())

		// Verify flow was reset
		rateLimit := s.rateLimit(ctx, chainA, denomA, chanAB.ChannelID)
		s.Require().NotNil(rateLimit)
		s.Require().Zero(rateLimit.Flow.Outflow.Int64())

		// Now the transfer should succeed again
		transferAmount := testvalues.DefaultTransferAmount(denomA)

		timeoutTimestamp := uint64(time.Now().Add(time.Hour).UnixNano())
		msgTransfer := transfertypes.NewMsgTransferWithEncoding(
			chanAB.PortID,
			chanAB.ChannelID,
			transferAmount,
			userA.FormattedAddress(),
			userB.FormattedAddress(),
			clienttypes.Height{}, // IBC v2 requires timeoutHeight to be zero
			timeoutTimestamp,
			"",      // memo
			"proto", // encoding
			true,    // useAliasing - this makes it IBC v2
		)

		txResp := s.BroadcastMessages(ctx, chainA, userA, msgTransfer)
		s.AssertTxSuccess(txResp)
	})
}

// Helper methods (reused from the original rate limiting test)
func (s *RateLimV2TestSuite) rateLimit(ctx context.Context, chain ibc.Chain, denom, chanID string) *ratelimitingtypes.RateLimit {
	respRateLim, err := query.GRPCQuery[ratelimitingtypes.QueryRateLimitResponse](ctx, chain, &ratelimitingtypes.QueryRateLimitRequest{
		Denom:             denom,
		ChannelOrClientId: chanID,
	})
	s.Require().NoError(err)
	return respRateLim.RateLimit
}

func (s *RateLimV2TestSuite) addRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, denom, chanID, authority string, sendPercent, recvPercent, duration int64) {
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

func (s *RateLimV2TestSuite) resetRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, denom, chanID, authority string) {
	msg := &ratelimitingtypes.MsgResetRateLimit{
		Signer:            authority,
		Denom:             denom,
		ChannelOrClientId: chanID,
	}
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, user)
}

func (s *RateLimV2TestSuite) updateRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, denom, chanID, authority string, sendPercent, recvPercent int64) {
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
