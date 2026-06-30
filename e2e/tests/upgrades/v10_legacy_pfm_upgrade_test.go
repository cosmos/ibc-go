//go:build !test_e2e

package upgrades

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/cosmos/interchaintest/v11/chain/cosmos"
	"github.com/cosmos/interchaintest/v11/ibc"
	test "github.com/cosmos/interchaintest/v11/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	packetforwardtypes "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
)

const (
	legacyV10IBCAppsImageTag        = "compat-v10-with-legacy-ibc-apps"
	legacyV10IBCAppsUpgradePlanName = "v11.1-legacy-ibc-apps"
)

// TestV10LegacyPFMUpgradeTestSuite upgrades a legacy v10 ibc-apps fixture chain to the current target image.
// It is intended for main/v11.1+ targets that include the v11.1-legacy-ibc-apps upgrade handler.
func TestV10LegacyPFMUpgradeTestSuite(t *testing.T) {
	testifysuite.Run(t, new(V10LegacyPFMUpgradeTestSuite))
}

type LegacyV10IBCAppsUpgradeTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *LegacyV10IBCAppsUpgradeTestSuite) SetupLegacyV10IBCAppsChains(chainCount int) {
	s.SetupChains(context.Background(), chainCount, nil, withLegacyV10IBCAppsOnChainB())
}

type V10LegacyPFMUpgradeTestSuite struct {
	LegacyV10IBCAppsUpgradeTestSuite
}

func (s *V10LegacyPFMUpgradeTestSuite) SetupSuite() {
	s.SetupLegacyV10IBCAppsChains(3)
}

// UpgradeChain upgrades a chain to a specific version using the planName provided.
// The software upgrade proposal is broadcast by the provided wallet.
func (s *LegacyV10IBCAppsUpgradeTestSuite) UpgradeChain(ctx context.Context, chain *cosmos.CosmosChain, wallet ibc.Wallet, planName, currentVersion, upgradeVersion string) {
	height, err := chain.GetNode().Height(ctx)
	s.Require().NoError(err, "error fetching height before upgrade")

	haltHeight := height + haltHeightOffset
	plan := upgradetypes.Plan{
		Name:   planName,
		Height: haltHeight,
		Info:   fmt.Sprintf("upgrade version test from %s to %s", currentVersion, upgradeVersion),
	}

	if testvalues.GovV1MessagesFeatureReleases.IsSupported(chain.Config().Images[0].Version) {
		msgSoftwareUpgrade := &upgradetypes.MsgSoftwareUpgrade{
			Plan:      plan,
			Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		}

		msgBz, err := chain.Config().EncodingConfig.Codec.MarshalInterfaceJSON(msgSoftwareUpgrade)
		s.Require().NoError(err)

		proposal := cosmos.TxProposalv1{
			Messages: []json.RawMessage{msgBz},
			Deposit:  fmt.Sprintf("%d%s", testvalues.DefaultGovV1ProposalTokenAmount, chain.Config().Denom),
			Title:    fmt.Sprintf("upgrade from %s to %s", currentVersion, upgradeVersion),
			Summary:  "upgrade chain E2E test",
		}
		s.submitAndPassGovV1Proposal(ctx, chain, wallet, proposal)
	} else {
		upgradeProposal := upgradetypes.NewSoftwareUpgradeProposal(fmt.Sprintf("upgrade from %s to %s", currentVersion, upgradeVersion), "upgrade chain E2E test", plan)
		s.ExecuteAndPassGovV1Beta1Proposal(ctx, chain, wallet, upgradeProposal)
	}

	err = test.WaitForCondition(time.Minute*2, time.Second*2, func() (bool, error) {
		status, err := chain.GetNode().Client.Status(ctx)
		if err != nil {
			return false, err
		}
		return status.SyncInfo.LatestBlockHeight >= haltHeight, nil
	})
	s.Require().NoError(err, "failed to wait for chain to halt")

	var allNodes []test.ChainHeighter
	for _, node := range chain.Nodes() {
		allNodes = append(allNodes, node)
	}

	err = test.WaitForInSync(ctx, chain, allNodes...)
	s.Require().NoError(err, "error waiting for node(s) to sync")

	err = chain.StopAllNodes(ctx)
	s.Require().NoError(err, "error stopping node(s)")

	repository := chain.Nodes()[0].Image.Repository
	chain.UpgradeVersion(ctx, s.DockerClient, repository, upgradeVersion)

	err = chain.StartAllNodes(ctx)
	s.Require().NoError(err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err = test.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	s.Require().NoError(err, "chain did not produce blocks after upgrade")

	height, err = chain.Height(ctx)
	s.Require().NoError(err, "error fetching height after upgrade")

	s.Require().Greater(height, haltHeight, "height did not increment after upgrade")

	// In case the query paths have changed after the upgrade, we need to repopulate them
	err = query.PopulateQueryReqToPath(ctx, chain)
	s.Require().NoError(err, "error populating query paths after upgrade")
}

func (s *LegacyV10IBCAppsUpgradeTestSuite) submitAndPassGovV1Proposal(ctx context.Context, chain *cosmos.CosmosChain, proposer ibc.Wallet, proposal cosmos.TxProposalv1) uint64 {
	proposalID := s.nextGovV1ProposalID(ctx, chain)
	s.submitGovV1Proposal(ctx, chain, proposer, proposal, proposalID)

	s.Require().NoError(chain.VoteOnProposalAllValidators(ctx, proposalID, cosmos.ProposalVoteYes))
	s.waitForGovV1ProposalToPass(ctx, chain, proposalID)

	return proposalID
}

func (s *LegacyV10IBCAppsUpgradeTestSuite) submitGovV1Proposal(ctx context.Context, chain *cosmos.CosmosChain, proposer ibc.Wallet, proposal cosmos.TxProposalv1, proposalID uint64) {
	proposalJSON, err := json.MarshalIndent(proposal, "", " ")
	s.Require().NoError(err)

	proposalFile := fmt.Sprintf("proposal-%d.json", proposalID)
	err = chain.GetNode().WriteFile(ctx, proposalJSON, proposalFile)
	s.Require().NoError(err)

	stdout, _, err := chain.GetNode().Exec(ctx, chain.GetNode().TxCommand(
		proposer.KeyName(),
		"gov", "submit-proposal", path.Join(chain.GetNode().HomeDir(), proposalFile), "--gas", "auto",
	), chain.Config().Env)
	s.Require().NoError(err)

	var tx cosmos.CosmosTx
	s.Require().NoError(json.Unmarshal(stdout, &tx))
	s.Require().Zero(tx.Code, tx.RawLog)
	s.Require().NoError(test.WaitForBlocks(ctx, 2, chain))

	err = test.WaitForCondition(time.Minute, time.Second, func() (bool, error) {
		proposalResp, err := query.GRPCQuery[govtypesv1.QueryProposalResponse](ctx, chain, &govtypesv1.QueryProposalRequest{
			ProposalId: proposalID,
		})
		if err != nil {
			return false, nil
		}

		return proposalResp.Proposal.Status == govtypesv1.StatusVotingPeriod, nil
	})
	s.Require().NoError(err)
}

func (s *LegacyV10IBCAppsUpgradeTestSuite) nextGovV1ProposalID(ctx context.Context, chain ibc.Chain) uint64 {
	proposalsResp, err := query.GRPCQuery[govtypesv1.QueryProposalsResponse](ctx, chain, &govtypesv1.QueryProposalsRequest{})
	s.Require().NoError(err)

	var maxProposalID uint64
	for _, proposal := range proposalsResp.Proposals {
		if proposal.Id > maxProposalID {
			maxProposalID = proposal.Id
		}
	}

	return maxProposalID + 1
}

func (s *LegacyV10IBCAppsUpgradeTestSuite) waitForGovV1ProposalToPass(ctx context.Context, chain ibc.Chain, proposalID uint64) {
	var govProposal *govtypesv1.Proposal
	err := test.WaitForCondition(testvalues.VotingPeriod+time.Minute, 2*time.Second, func() (bool, error) {
		proposalResp, err := query.GRPCQuery[govtypesv1.QueryProposalResponse](ctx, chain, &govtypesv1.QueryProposalRequest{
			ProposalId: proposalID,
		})
		if err != nil {
			return false, err
		}

		govProposal = proposalResp.Proposal
		return govProposal.Status == govtypesv1.StatusPassed, nil
	})

	failedReason := ""
	if govProposal != nil {
		failedReason = govProposal.FailedReason
	}
	s.Require().NoError(err, failedReason)
}

func withLegacyV10IBCAppsOnChainB() testsuite.ChainOptionConfiguration {
	return func(options *testsuite.ChainOptions) {
		validators := 1
		fullNodes := 0

		options.ChainSpecs[1].ChainConfig.Images[0].Version = legacyV10IBCAppsImageTag
		options.ChainSpecs[1].NumValidators = &validators
		options.ChainSpecs[1].NumFullNodes = &fullNodes
	}
}

func buildForwardMemo(receiver, channel string, timeout time.Duration, retries *uint8) (string, error) {
	return packetforwardtypes.PacketMetadata{
		Forward: packetforwardtypes.ForwardMetadata{
			Receiver: receiver,
			Port:     transfertypes.PortID,
			Channel:  channel,
			Timeout:  timeout,
			Retries:  retries,
		},
	}.ToMemo()
}

func (s *V10LegacyPFMUpgradeTestSuite) assertNoPFMInFlightPackets(ctx context.Context, chain *cosmos.CosmosChain) {
	height, err := chain.Height(ctx)
	s.Require().NoError(err)

	err = chain.StopAllNodes(ctx)
	s.Require().NoError(err)

	state, err := chain.ExportState(ctx, height)
	s.Require().NoError(err)

	var exportedState struct {
		AppState map[string]json.RawMessage `json:"app_state"`
	}
	s.Require().NoError(json.Unmarshal([]byte(state), &exportedState))

	moduleState, ok := exportedState.AppState[packetforwardtypes.ModuleName]
	s.Require().True(ok, "packet forward middleware app state missing from exported state")

	var pfmGenesis packetforwardtypes.GenesisState
	s.Require().NoError(json.Unmarshal(moduleState, &pfmGenesis))
	s.Require().Empty(pfmGenesis.InFlightPackets)
}

func (s *V10LegacyPFMUpgradeTestSuite) TestV10LegacyPFMUpgradePreservesInFlightPackets() {
	ctx := context.Background()
	testName := s.T().Name()

	chains := s.GetAllChains()
	chainA := chains[0]
	chainB := chains[1]
	chainC := chains[2]

	relayer := s.GetRelayerForTest(testName)
	s.CreatePath(ctx, relayer, chainA, chainB, ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	s.CreatePath(ctx, relayer, chainB, chainC, ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)

	chanAB := s.GetChannelBetweenChains(testName, chainA, chainB)
	chanBC := s.GetChannelBetweenChains(testName, chainB, chainC)
	pathAB := s.GetPathByChains(chainA, chainB)
	pathBA := s.GetPathByChains(chainB, chainA)
	pathBC := s.GetPathByChains(chainB, chainC)

	senderA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	receiverB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	receiverC := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)

	successAmount := sdkmath.NewInt(100_000)
	timeoutAmount := sdkmath.NewInt(200_000)
	totalForwarded := successAmount.Add(timeoutAmount)
	retries := uint8(0)
	escrowAddrAB := transfertypes.GetEscrowAddress(chanAB.PortID, chanAB.ChannelID)
	escrowAddrBC := transfertypes.GetEscrowAddress(chanBC.PortID, chanBC.ChannelID)
	ibcTokenB := testsuite.GetIBCToken(chainA.Config().Denom, chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID)
	ibcTokenC := testsuite.GetIBCToken(ibcTokenB.Path(), chanBC.Counterparty.PortID, chanBC.Counterparty.ChannelID)

	// Stage each A -> B packet separately so we can prove B actually received it and
	// avoid Hermes batching both recvs into a single oversized tx.
	bHeightBeforeTimeoutStage, err := chainB.Height(ctx)
	s.Require().NoError(err)

	timeoutMemo, err := buildForwardMemo(receiverC.FormattedAddress(), chanBC.ChannelID, 10*time.Second, &retries)
	s.Require().NoError(err)

	timeoutTx, err := chainA.SendIBCTransfer(ctx, chanAB.ChannelID, senderA.KeyName(), ibc.WalletAmount{
		Address: receiverB.FormattedAddress(),
		Denom:   chainA.Config().Denom,
		Amount:  timeoutAmount,
	}, ibc.TransferOptions{Memo: timeoutMemo})
	s.Require().NoError(err)

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB))

	err = relayer.Flush(ctx, s.GetRelayerExecReporter(), pathAB, chanAB.ChannelID)
	s.Require().NoError(err)
	s.Require().NoError(test.WaitForBlocks(ctx, 2, chainA, chainB))

	bHeightAfterTimeoutStage, err := chainB.Height(ctx)
	s.Require().NoError(err)

	_, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, chainB.(*cosmos.CosmosChain), chainB.Config().EncodingConfig.InterfaceRegistry, bHeightBeforeTimeoutStage, bHeightAfterTimeoutStage+20, nil)
	s.Require().NoError(err)

	userABalance, err := query.Balance(ctx, chainA, senderA.FormattedAddress(), chainA.Config().Denom)
	s.Require().NoError(err)
	s.Require().Equal(testvalues.StartingTokenAmount-timeoutAmount.Int64(), userABalance.Int64())

	receiverBBalance, err := query.Balance(ctx, chainB, receiverB.FormattedAddress(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)
	s.Require().Zero(receiverBBalance.Int64())

	receiverCBalance, err := query.Balance(ctx, chainC, receiverC.FormattedAddress(), ibcTokenC.IBCDenom())
	s.Require().NoError(err)
	s.Require().Zero(receiverCBalance.Int64())

	escrowABBalance, err := query.Balance(ctx, chainA, escrowAddrAB.String(), chainA.Config().Denom)
	s.Require().NoError(err)
	s.Require().Equal(timeoutAmount, escrowABBalance)

	escrowBCBalance, err := query.Balance(ctx, chainB, escrowAddrBC.String(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)
	s.Require().Equal(timeoutAmount, escrowBCBalance)

	_, err = query.GRPCQuery[chantypes.QueryPacketCommitmentResponse](ctx, chainA, &chantypes.QueryPacketCommitmentRequest{
		PortId:    chanAB.PortID,
		ChannelId: chanAB.ChannelID,
		Sequence:  timeoutTx.Packet.Sequence,
	})
	s.Require().NoError(err)

	time.Sleep(12 * time.Second)

	bHeightBeforeSuccessStage, err := chainB.Height(ctx)
	s.Require().NoError(err)

	successMemo, err := buildForwardMemo(receiverC.FormattedAddress(), chanBC.ChannelID, 0, nil)
	s.Require().NoError(err)

	successTx, err := chainA.SendIBCTransfer(ctx, chanAB.ChannelID, senderA.KeyName(), ibc.WalletAmount{
		Address: receiverB.FormattedAddress(),
		Denom:   chainA.Config().Denom,
		Amount:  successAmount,
	}, ibc.TransferOptions{Memo: successMemo})
	s.Require().NoError(err)

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB))

	err = relayer.Flush(ctx, s.GetRelayerExecReporter(), pathAB, chanAB.ChannelID)
	s.Require().NoError(err)
	s.Require().NoError(test.WaitForBlocks(ctx, 2, chainA, chainB))

	bHeightAfterSuccessStage, err := chainB.Height(ctx)
	s.Require().NoError(err)

	_, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, chainB.(*cosmos.CosmosChain), chainB.Config().EncodingConfig.InterfaceRegistry, bHeightBeforeSuccessStage, bHeightAfterSuccessStage+20, nil)
	s.Require().NoError(err)

	userABalance, err = query.Balance(ctx, chainA, senderA.FormattedAddress(), chainA.Config().Denom)
	s.Require().NoError(err)
	s.Require().Equal(testvalues.StartingTokenAmount-totalForwarded.Int64(), userABalance.Int64())

	receiverBBalance, err = query.Balance(ctx, chainB, receiverB.FormattedAddress(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)
	s.Require().Zero(receiverBBalance.Int64())

	receiverCBalance, err = query.Balance(ctx, chainC, receiverC.FormattedAddress(), ibcTokenC.IBCDenom())
	s.Require().NoError(err)
	s.Require().Zero(receiverCBalance.Int64())

	escrowABBalance, err = query.Balance(ctx, chainA, escrowAddrAB.String(), chainA.Config().Denom)
	s.Require().NoError(err)
	s.Require().Equal(totalForwarded, escrowABBalance)

	escrowBCBalance, err = query.Balance(ctx, chainB, escrowAddrBC.String(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)
	s.Require().Equal(totalForwarded, escrowBCBalance)

	_, err = query.GRPCQuery[chantypes.QueryPacketCommitmentResponse](ctx, chainA, &chantypes.QueryPacketCommitmentRequest{
		PortId:    chanAB.PortID,
		ChannelId: chanAB.ChannelID,
		Sequence:  successTx.Packet.Sequence,
	})
	s.Require().NoError(err)

	_, err = query.GRPCQuery[chantypes.QueryPacketCommitmentResponse](ctx, chainA, &chantypes.QueryPacketCommitmentRequest{
		PortId:    chanAB.PortID,
		ChannelId: chanAB.ChannelID,
		Sequence:  timeoutTx.Packet.Sequence,
	})
	s.Require().NoError(err)

	upgradeProposer := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	s.UpgradeChain(
		ctx,
		chainB.(*cosmos.CosmosChain),
		upgradeProposer,
		legacyV10IBCAppsUpgradePlanName,
		chainB.Config().Images[0].Version,
		chainA.Config().Images[0].Version,
	)

	// Resolve the timed out and successful forwards in explicit steps so the test
	// observes the actual packet lifecycle instead of relying on Hermes start.
	bHeightBeforeBarrierFlush, err := chainB.Height(ctx)
	s.Require().NoError(err)

	cHeightBeforeBarrierFlush, err := chainC.Height(ctx)
	s.Require().NoError(err)

	err = relayer.Flush(ctx, s.GetRelayerExecReporter(), pathBC, chanBC.ChannelID)
	s.Require().NoError(err)
	s.Require().NoError(test.WaitForBlocks(ctx, 2, chainB, chainC))

	bHeightAfterBarrierFlush, err := chainB.Height(ctx)
	s.Require().NoError(err)

	cHeightAfterBarrierFlush, err := chainC.Height(ctx)
	s.Require().NoError(err)

	_, err = cosmos.PollForMessage[*chantypes.MsgTimeout](ctx, chainB.(*cosmos.CosmosChain), chainB.Config().EncodingConfig.InterfaceRegistry, bHeightBeforeBarrierFlush, bHeightAfterBarrierFlush+30, nil)
	s.Require().NoError(err)

	_, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, chainC.(*cosmos.CosmosChain), chainC.Config().EncodingConfig.InterfaceRegistry, cHeightBeforeBarrierFlush, cHeightAfterBarrierFlush+30, nil)
	s.Require().NoError(err)

	aHeightBeforeAckFlush, err := chainA.Height(ctx)
	s.Require().NoError(err)

	err = relayer.Flush(ctx, s.GetRelayerExecReporter(), pathBA, chanAB.Counterparty.ChannelID)
	s.Require().NoError(err)
	s.Require().NoError(test.WaitForBlocks(ctx, 2, chainA, chainB))

	aHeightAfterAckFlush, err := chainA.Height(ctx)
	s.Require().NoError(err)

	_, err = test.PollForAck(ctx, chainA, aHeightBeforeAckFlush, aHeightAfterAckFlush+30, timeoutTx.Packet)
	s.Require().NoError(err)

	_, err = test.PollForAck(ctx, chainA, aHeightBeforeAckFlush, aHeightAfterAckFlush+30, successTx.Packet)
	s.Require().NoError(err)

	userABalance, err = query.Balance(ctx, chainA, senderA.FormattedAddress(), chainA.Config().Denom)
	s.Require().NoError(err)
	s.Require().Equal(testvalues.StartingTokenAmount-successAmount.Int64(), userABalance.Int64())

	receiverBBalance, err = query.Balance(ctx, chainB, receiverB.FormattedAddress(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)
	s.Require().Zero(receiverBBalance.Int64())

	receiverCBalance, err = query.Balance(ctx, chainC, receiverC.FormattedAddress(), ibcTokenC.IBCDenom())
	s.Require().NoError(err)
	s.Require().Equal(successAmount, receiverCBalance)

	escrowBCBalance, err = query.Balance(ctx, chainB, escrowAddrBC.String(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)
	s.Require().Equal(successAmount, escrowBCBalance)

	escrowABBalance, err = query.Balance(ctx, chainA, escrowAddrAB.String(), chainA.Config().Denom)
	s.Require().NoError(err)
	s.Require().Equal(successAmount, escrowABBalance)

	s.assertNoPFMInFlightPackets(ctx, chainB.(*cosmos.CosmosChain))
}
