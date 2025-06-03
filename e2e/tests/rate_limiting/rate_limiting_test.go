//go:build !test_e2e

package ratelimiting

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	ratelimitingtypes "github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	testifysuite "github.com/stretchr/testify/suite"
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

	relayer := s.CreateDefaultPaths(testName)
	s.StartRelayer(relayer, testName)

	chanAB := s.GetChainAToChainBChannel(testName)

	userA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	escrowAddrA := transfertypes.GetEscrowAddress(chanAB.PortID, chanAB.ChannelID)
	denomA := chainA.Config().Denom

	ibcTokenB := testsuite.GetIBCToken(denomA, chanAB.PortID, chanAB.ChannelID)

	// No rate limit set
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
	fmt.Printf("UserB :%s BalanceAfrer: %s\n", userB.FormattedAddress(), userBBalAfter)

	// Set Sending limit on chainA

	// No existing rate limit
	resp, err := query.GRPCQuery[ratelimitingtypes.QueryAllRateLimitsResponse](ctx, chainA, &ratelimitingtypes.QueryAllRateLimitsRequest{})
	s.Require().NoError(err)
	s.Require().Nil(resp)

	txResp = s.AddRateLimit(ctx, chainA, userA, userA.FormattedAddress(), denomA, chanAB.ChannelID, 10, 0, 1)
	s.AssertTxSuccess(txResp)
	// packet, err = ibctesting.ParseV1PacketFromEvents(txResp.Events)
	// s.Require().NoError(err)
	// s.Require().NotNil(packet)
	resp, err = query.GRPCQuery[ratelimitingtypes.QueryAllRateLimitsResponse](ctx, chainA, &ratelimitingtypes.QueryAllRateLimitsRequest{})
	s.NoError(err)
	fmt.Printf("Rate Limit: %+v\n", resp.RateLimits)
}
