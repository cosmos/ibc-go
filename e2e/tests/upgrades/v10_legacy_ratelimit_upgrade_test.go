//go:build !test_e2e

package upgrades

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/interchaintest/v11/chain/cosmos"
	"github.com/cosmos/interchaintest/v11/ibc"
	test "github.com/cosmos/interchaintest/v11/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	ratelimitingtypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
)

// TestV10LegacyRateLimitUpgradeTestSuite upgrades a legacy v10 ibc-apps rate-limit fixture chain to the current target image.
// It is intended for main/v11.2+ targets that include the v11.2-legacy-ibc-apps upgrade handler.
func TestV10LegacyRateLimitUpgradeTestSuite(t *testing.T) {
	testifysuite.Run(t, new(V10LegacyRateLimitUpgradeTestSuite))
}

type V10LegacyRateLimitUpgradeTestSuite struct {
	LegacyV10IBCAppsUpgradeTestSuite
}

const legacyPendingSendPacketChannelLength = 16

func (s *V10LegacyRateLimitUpgradeTestSuite) SetupSuite() {
	s.SetupLegacyV10IBCAppsChains(2)
}

func (s *LegacyV10IBCAppsUpgradeTestSuite) submitLegacyRateLimitProposal(ctx context.Context, chain *cosmos.CosmosChain, proposer ibc.Wallet, denom, channelID string, maxPercentSend, maxPercentRecv int64, durationHours uint64) {
	authority, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chain)
	s.Require().NoError(err)

	msg := map[string]any{
		"@type":                "/ratelimit.v1.MsgAddRateLimit",
		"authority":            authority.String(),
		"denom":                denom,
		"channel_or_client_id": channelID,
		"max_percent_send":     sdkmath.NewInt(maxPercentSend).String(),
		"max_percent_recv":     sdkmath.NewInt(maxPercentRecv).String(),
		"duration_hours":       strconv.FormatUint(durationHours, 10),
	}
	msgBz, err := json.Marshal(msg)
	s.Require().NoError(err)

	proposal := cosmos.TxProposalv1{
		Messages: []json.RawMessage{msgBz},
		Deposit:  fmt.Sprintf("%d%s", testvalues.DefaultGovV1ProposalTokenAmount, chain.Config().Denom),
		Title:    "legacy v10 rate limit",
		Summary:  "configure legacy v10 ibc-apps rate limit",
	}

	s.submitAndPassLegacyGovV1Proposal(ctx, chain, proposer, proposal)

	stdout, _, err := chain.GetNode().ExecQuery(ctx, ratelimitingtypes.ModuleName, "rate-limit", channelID, "--denom", denom)
	s.Require().NoError(err)
	s.Require().Contains(string(stdout), denom)
}

func (s *V10LegacyRateLimitUpgradeTestSuite) legacyV2PendingSendPacketKey(channelID string, sequence uint64) []byte {
	key, err := ratelimitingtypes.PendingPacketKey(channelID, sequence)
	s.Require().NoError(err)
	return key
}

func (s *V10LegacyRateLimitUpgradeTestSuite) legacyPendingSendPacketKey(channelID string, sequence uint64) []byte {
	s.Require().LessOrEqual(len(channelID), legacyPendingSendPacketChannelLength)

	key := make([]byte, legacyPendingSendPacketChannelLength+8)
	copy(key, channelID)
	binary.BigEndian.PutUint64(key[legacyPendingSendPacketChannelLength:], sequence)

	return key
}

func (s *V10LegacyRateLimitUpgradeTestSuite) assertRateLimitPendingSendPacketExists(ctx context.Context, chain *cosmos.CosmosChain, key []byte) {
	s.Require().NotEmpty(s.rateLimitPendingSendPacketValue(ctx, chain, key))
}

func (s *V10LegacyRateLimitUpgradeTestSuite) assertNoRateLimitPendingSendPacket(ctx context.Context, chain *cosmos.CosmosChain, key []byte) {
	s.Require().Empty(s.rateLimitPendingSendPacketValue(ctx, chain, key))
}

func (s *V10LegacyRateLimitUpgradeTestSuite) rateLimitPendingSendPacketValue(ctx context.Context, chain *cosmos.CosmosChain, key []byte) []byte {
	fullKey := append(append([]byte{}, ratelimitingtypes.PendingSendPacketPrefix...), key...)
	res, err := chain.GetNode().Client.ABCIQuery(ctx, fmt.Sprintf("store/%s/key", ratelimitingtypes.StoreKey), fullKey)
	s.Require().NoError(err)
	s.Require().Zero(res.Response.Code, res.Response.Log)

	return res.Response.Value
}

func (s *V10LegacyRateLimitUpgradeTestSuite) rateLimit(ctx context.Context, chain ibc.Chain, denom, channelID string) *ratelimitingtypes.RateLimit {
	resp, err := query.GRPCQuery[ratelimitingtypes.QueryRateLimitResponse](ctx, chain, &ratelimitingtypes.QueryRateLimitRequest{
		Denom:             denom,
		ChannelOrClientId: channelID,
	})
	s.Require().NoError(err)
	return resp.RateLimit
}

func (s *V10LegacyRateLimitUpgradeTestSuite) TestV10LegacyRateLimitUpgradeClearsPendingSendPackets() {
	ctx := context.Background()
	testName := s.T().Name()

	chains := s.GetAllChains()
	chainA := chains[0]
	chainB := chains[1]
	cosmosChainB, ok := chainB.(*cosmos.CosmosChain)
	s.Require().True(ok)

	relayer := s.GetRelayerForTest(testName)
	s.CreatePath(ctx, relayer, chainB, chainA, ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)

	chanBA := s.GetChannelBetweenChains(testName, chainB, chainA)
	pathBA := s.GetPathByChains(chainB, chainA)

	senderB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	receiverA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	upgradeProposer := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	denomB := chainB.Config().Denom
	timeoutAmount := sdkmath.NewInt(100_000)

	s.submitLegacyRateLimitProposal(ctx, cosmosChainB, upgradeProposer, denomB, chanBA.ChannelID, 100, 100, 1)

	timeoutTx, err := chainB.SendIBCTransfer(ctx, chanBA.ChannelID, senderB.KeyName(), ibc.WalletAmount{
		Address: receiverA.FormattedAddress(),
		Denom:   denomB,
		Amount:  timeoutAmount,
	}, ibc.TransferOptions{
		Timeout: &ibc.IBCTimeout{NanoSeconds: uint64((5 * time.Second).Nanoseconds())},
	})
	s.Require().NoError(err)
	legacyV2PendingSendKey := s.legacyV2PendingSendPacketKey(chanBA.ChannelID, timeoutTx.Packet.Sequence)
	legacyPendingSendKey := s.legacyPendingSendPacketKey(chanBA.ChannelID, timeoutTx.Packet.Sequence)
	s.assertRateLimitPendingSendPacketExists(ctx, cosmosChainB, legacyPendingSendKey)
	s.assertNoRateLimitPendingSendPacket(ctx, cosmosChainB, legacyV2PendingSendKey)

	s.UpgradeChain(
		ctx,
		cosmosChainB,
		upgradeProposer,
		legacyV10IBCAppsUpgradePlanName,
		chainB.Config().Images[0].Version,
		chainA.Config().Images[0].Version,
	)

	rateLimit := s.rateLimit(ctx, chainB, denomB, chanBA.ChannelID)
	s.Require().NotNil(rateLimit)
	s.Require().Equal(timeoutAmount, rateLimit.Flow.Outflow)
	s.assertNoRateLimitPendingSendPacket(ctx, cosmosChainB, legacyPendingSendKey)
	s.assertNoRateLimitPendingSendPacket(ctx, cosmosChainB, legacyV2PendingSendKey)

	bHeightBeforeFlush, err := chainB.Height(ctx)
	s.Require().NoError(err)

	err = relayer.Flush(ctx, s.GetRelayerExecReporter(), pathBA, chanBA.ChannelID)
	s.Require().NoError(err)
	s.Require().NoError(test.WaitForBlocks(ctx, 2, chainA, chainB))

	bHeightAfterFlush, err := chainB.Height(ctx)
	s.Require().NoError(err)

	_, err = cosmos.PollForMessage[*chantypes.MsgTimeout](ctx, cosmosChainB, chainB.Config().EncodingConfig.InterfaceRegistry, bHeightBeforeFlush, bHeightAfterFlush+30, nil)
	s.Require().NoError(err)

	rateLimit = s.rateLimit(ctx, chainB, denomB, chanBA.ChannelID)
	s.Require().NotNil(rateLimit)
	s.Require().Equal(timeoutAmount, rateLimit.Flow.Outflow)

	escrowAddrBA := transfertypes.GetEscrowAddress(chanBA.PortID, chanBA.ChannelID)
	escrowBABalance, err := query.Balance(ctx, chainB, escrowAddrBA.String(), denomB)
	s.Require().NoError(err)
	s.Require().True(escrowBABalance.IsZero())

	s.assertNoRateLimitPendingSendPacket(ctx, cosmosChainB, legacyPendingSendKey)
	s.assertNoRateLimitPendingSendPacket(ctx, cosmosChainB, legacyV2PendingSendKey)
}
