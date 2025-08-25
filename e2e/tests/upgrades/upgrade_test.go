//go:build !test_e2e

package upgrades

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	interchaintest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	test "github.com/cosmos/interchaintest/v10/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	e2erelayer "github.com/cosmos/ibc-go/e2e/relayer"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	v7migrations "github.com/cosmos/ibc-go/v10/modules/core/02-client/migrations/v7"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	localhost "github.com/cosmos/ibc-go/v10/modules/light-clients/09-localhost"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const (
	haltHeightOffset   = int64(30)
	blocksAfterUpgrade = uint64(10)
)

func TestUpgradeTestSuite(t *testing.T) {
	testCfg := testsuite.LoadConfig()
	if testCfg.UpgradePlanName == "" {
		t.Fatalf("%s must be set when running an upgrade test", testsuite.ChainUpgradePlanEnv)
	}

	testifysuite.Run(t, new(UpgradeTestSuite))
}

type UpgradeTestSuite struct {
	testsuite.E2ETestSuite
}

// SetupSuite sets up chains for the current test suite
func (s *UpgradeTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil)
}

// UpgradeChain upgrades a chain to a specific version using the planName provided.
// The software upgrade proposal is broadcast by the provided wallet.
func (s *UpgradeTestSuite) UpgradeChain(ctx context.Context, chain *cosmos.CosmosChain, wallet ibc.Wallet, planName, currentVersion, upgradeVersion string) {
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

		s.ExecuteAndPassGovV1Proposal(ctx, msgSoftwareUpgrade, chain, wallet)
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

func (s *UpgradeTestSuite) TestIBCChainUpgrade() {
	t := s.T()
	testCfg := testsuite.LoadConfig()

	ctx := context.Background()
	testName := t.Name()

	chainA, chainB := s.GetChains()

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainADenom := chainA.Config().Denom
	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID) // IBC token sent to chainB

	chainBDenom := chainB.Config().Denom
	chainAIBCToken := testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID) // IBC token sent to chainA

	// create separate user specifically for the upgrade proposal to more easily verify starting
	// and end balances of the chainA users.
	chainAUpgradeProposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())

		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("upgrade chainA", func(t *testing.T) {
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), chainAUpgradeProposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	t.Run("restart relayer", func(t *testing.T) {
		s.StopRelayer(ctx, relayer)
		s.StartRelayer(relayer, testName)
	})

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 2)
		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())

		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount * 2
		s.Require().Equal(expected, actualBalance.Int64())
	})

	t.Run("ensure packets can be received, send from chainB to chainA", func(t *testing.T) {
		t.Run("send from chainB to chainA", func(t *testing.T) {
			transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
			s.AssertTxSuccess(transferTxResp)
		})

		s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

		t.Run("packets are relayed", func(t *testing.T) {
			s.AssertPacketRelayed(ctx, chainA, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

			actualBalance, err := query.Balance(ctx, chainA, chainAAddress, chainAIBCToken.IBCDenom())

			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance.Int64())
		})
	})
}

func (s *UpgradeTestSuite) TestChainUpgrade() {
	t := s.T()

	ctx := context.Background()

	testName := t.Name()
	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)

	// TODO(chatton): this test is still creating a relayer and a channel, but it is not using them.
	chain := s.GetAllChains()[0]

	userWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userWalletAddr := userWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chain), "failed to wait for blocks")

	t.Run("send funds to test wallet", func(t *testing.T) {
		err := chain.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
			Address: userWalletAddr,
			Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
			Denom:   chain.Config().Denom,
		})
		s.Require().NoError(err)
	})

	t.Run("verify tokens sent", func(t *testing.T) {
		balance, err := query.Balance(ctx, chain, userWalletAddr, chain.Config().Denom)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount * 2
		s.Require().Equal(expected, balance)
	})

	t.Run("upgrade chain", func(t *testing.T) {
		testCfg := testsuite.LoadConfig()
		proposerWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

		s.UpgradeChain(ctx, chain.(*cosmos.CosmosChain), proposerWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	t.Run("send funds to test wallet", func(t *testing.T) {
		err := chain.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
			Address: userWalletAddr,
			Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
			Denom:   chain.Config().Denom,
		})
		s.Require().NoError(err)
	})

	t.Run("verify tokens sent", func(t *testing.T) {
		balance, err := query.Balance(ctx, chain, userWalletAddr, chain.Config().Denom)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount * 3
		s.Require().Equal(expected, balance)
	})
}

// TestV6ToV7ChainUpgrade will test that an upgrade from a v6 ibc-go binary to a v7 ibc-go binary is successful
// and that the automatic migrations associated with the 02-client module are performed. Namely that the solo machine
// proto definition is migrated in state from the v2 to v3 definition. This is checked by creating a solo machine client
// before the upgrade and asserting that its TypeURL has been changed after the upgrade. The test also ensure packets
// can be sent before and after the upgrade without issue
func (s *UpgradeTestSuite) TestV6ToV7ChainUpgrade() {
	t := s.T()
	testCfg := testsuite.LoadConfig()

	ctx := context.Background()
	testName := t.Name()

	chainA, chainB := s.GetChains()

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainADenom := chainA.Config().Denom
	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID) // IBC token sent to chainB

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	// create second tendermint client
	createClientOptions := ibc.CreateClientOptions{
		TrustingPeriod: ibctesting.TrustingPeriod.String(),
	}

	s.SetupClients(ctx, relayer, createClientOptions)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("check that both tendermint clients are active", func(t *testing.T) {
		status, err := query.ClientStatus(ctx, chainA, testvalues.TendermintClientID(0))
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		status, err = query.ClientStatus(ctx, chainA, testvalues.TendermintClientID(1))
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)
	})

	// create solo machine client using the solomachine implementation from ibctesting
	// TODO: the solomachine clientID should be updated when after fix of this issue: https://github.com/cosmos/ibc-go/issues/2907
	solo := ibctesting.NewSolomachine(t, testsuite.Codec(), "solomachine", "testing", 1)

	legacyConsensusState := &v7migrations.ConsensusState{
		PublicKey:   solo.ConsensusState().PublicKey,
		Diversifier: solo.ConsensusState().Diversifier,
		Timestamp:   solo.ConsensusState().Timestamp,
	}

	legacyClientState := &v7migrations.ClientState{
		Sequence:                 solo.ClientState().Sequence,
		IsFrozen:                 solo.ClientState().IsFrozen,
		ConsensusState:           legacyConsensusState,
		AllowUpdateAfterProposal: true,
	}

	msgCreateSoloMachineClient, err := clienttypes.NewMsgCreateClient(legacyClientState, legacyConsensusState, chainAAddress)
	s.Require().NoError(err)

	resp := s.BroadcastMessages(
		ctx,
		chainA.(*cosmos.CosmosChain),
		chainAWallet,
		msgCreateSoloMachineClient,
	)

	s.AssertTxSuccess(resp)

	t.Run("check that the solomachine is now active and that the clientstate is a pre-upgrade v2 solomachine clientstate", func(t *testing.T) {
		status, err := query.ClientStatus(ctx, chainA, testvalues.SolomachineClientID(2))
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		res, err := s.ClientState(ctx, chainA, testvalues.SolomachineClientID(2))
		s.Require().NoError(err)
		s.Require().Equal(fmt.Sprint("/", proto.MessageName(&v7migrations.ClientState{})), res.ClientState.TypeUrl)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	// create separate user specifically for the upgrade proposal to more easily verify starting
	// and end balances of the chainA users.
	chainAUpgradeProposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	t.Run("upgrade chainA", func(t *testing.T) {
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), chainAUpgradeProposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	// see this issue https://github.com/informalsystems/hermes/issues/3579
	// this restart is a temporary workaround to a limitation in hermes requiring a restart
	// in some cases after an upgrade.
	tc := testsuite.LoadConfig()
	if tc.GetActiveRelayerConfig().ID == e2erelayer.Hermes {
		s.RestartRelayer(ctx, relayer, testName)
	}

	t.Run("check that the tendermint clients are active again after upgrade", func(t *testing.T) {
		status, err := query.ClientStatus(ctx, chainA, testvalues.TendermintClientID(0))
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		status, err = query.ClientStatus(ctx, chainA, testvalues.TendermintClientID(1))
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)
	})

	t.Run("IBC token transfer from chainA to chainB, to make sure the upgrade did not break the packet flow", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB.(*cosmos.CosmosChain)), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount * 2
		s.Require().Equal(expected, actualBalance.Int64())
	})

	t.Run("check that the v2 solo machine clientstate has been updated to the v3 solo machine clientstate", func(t *testing.T) {
		res, err := s.ClientState(ctx, chainA, testvalues.SolomachineClientID(2))
		s.Require().NoError(err)
		s.Require().Equal(fmt.Sprint("/", proto.MessageName(&solomachine.ClientState{})), res.ClientState.TypeUrl)
	})
}

func (s *UpgradeTestSuite) TestV7ToV7_1ChainUpgrade() {
	t := s.T()
	testCfg := testsuite.LoadConfig()

	ctx := context.Background()
	testName := t.Name()

	chainA, chainB := s.GetChains()

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("transfer native tokens from chainA to chainB", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB.(*cosmos.CosmosChain)), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packet is relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA), "failed to wait for blocks")

	t.Run("upgrade chain", func(t *testing.T) {
		govProposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), govProposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	t.Run("ensure the localhost client is active and sentinel connection is stored in state", func(t *testing.T) {
		localhostClientID := exported.LocalhostClientID
		if !testvalues.LocalhostWithDashFeatureReleases.IsSupported(chainA.Config().Images[0].Version) {
			localhostClientID = exported.Localhost
		}
		status, err := query.ClientStatus(ctx, chainA, localhostClientID)
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		connectionResp, err := query.GRPCQuery[connectiontypes.QueryConnectionResponse](ctx, chainA, &connectiontypes.QueryConnectionRequest{ConnectionId: exported.LocalhostConnectionID})
		s.Require().NoError(err)
		s.Require().Equal(connectiontypes.OPEN, connectionResp.Connection.State)
		s.Require().Equal(localhostClientID, connectionResp.Connection.ClientId)
		s.Require().Equal(localhostClientID, connectionResp.Connection.Counterparty.ClientId)
		s.Require().Equal(exported.LocalhostConnectionID, connectionResp.Connection.Counterparty.ConnectionId)
	})

	t.Run("ensure escrow amount for native denom is stored in state", func(t *testing.T) {
		actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainA, chainADenom)
		s.Require().NoError(err)

		expectedTotalEscrow := sdk.NewCoin(chainADenom, sdkmath.NewInt(testvalues.IBCTransferAmount))
		s.Require().Equal(expectedTotalEscrow, actualTotalEscrow) // migration has run and total escrow amount has been set
	})

	t.Run("IBC token transfer from chainA to chainB, to make sure the upgrade did not break the packet flow", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount * 2
		s.Require().Equal(expected, actualBalance.Int64())
	})
}

func (s *UpgradeTestSuite) TestV7ToV8ChainUpgrade() {
	t := s.T()
	testCfg := testsuite.LoadConfig()

	ctx := context.Background()
	testName := t.Name()

	chainA, chainB := s.GetChains()

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("transfer native tokens from chainA to chainB", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packet is relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA), "failed to wait for blocks")

	t.Run("upgrade chain", func(t *testing.T) {
		govProposalWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
		s.UpgradeChain(ctx, chainB.(*cosmos.CosmosChain), govProposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	t.Run("update params", func(t *testing.T) {
		authority, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chainB)
		s.Require().NoError(err)
		s.Require().NotNil(authority)

		msg := clienttypes.NewMsgUpdateParams(authority.String(), clienttypes.NewParams(exported.Tendermint, "some-client"))
		s.ExecuteAndPassGovV1Proposal(ctx, msg, chainB, chainBWallet)
	})

	t.Run("query params", func(t *testing.T) {
		clientParamsResp, err := query.GRPCQuery[clienttypes.QueryClientParamsResponse](ctx, chainB, &clienttypes.QueryClientParamsRequest{})
		s.Require().NoError(err)

		allowedClients := clientParamsResp.Params.AllowedClients

		s.Require().Len(allowedClients, 2)
		s.Require().Contains(allowedClients, exported.Tendermint)
		s.Require().Contains(allowedClients, "some-client")
	})

	t.Run("query human readable ibc denom", func(t *testing.T) {
		s.AssertHumanReadableDenom(ctx, chainB, chainADenom, channelA)
	})

	t.Run("IBC token transfer from chainA to chainB, to make sure the upgrade did not break the packet flow", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount * 2
		s.Require().Equal(expected, actualBalance.Int64())
	})
}

func (s *UpgradeTestSuite) TestV8ToV8_1ChainUpgrade() {
	t := s.T()
	ctx := context.Background()

	testName := t.Name()
	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)

	chainA, chainB := s.GetChains()

	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)
	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("transfer native tokens from chainA to chainB", func(t *testing.T) {
		txResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(txResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("upgrade chain", func(t *testing.T) {
		testCfg := testsuite.LoadConfig()
		proposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), proposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packet is relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA), "failed to wait for blocks")

	t.Run("IBC token transfer from chainA to chainB, to make sure the upgrade did not break the packet flow", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount * 2
		s.Require().Equal(expected, actualBalance.Int64())
	})
}

func (s *UpgradeTestSuite) TestV8ToV10ChainUpgrade() {
	t := s.T()
	testCfg := testsuite.LoadConfig()
	ctx := context.Background()

	testName := t.Name()

	chainA, chainB := s.GetChains()

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainADenom := chainA.Config().Denom
	chainBDenom := chainB.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	chainAIBCToken := testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID)
	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("transfer native tokens from chainA to chainB", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)

		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	t.Run("transfer native tokens from chainB to chainA", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA.(*cosmos.CosmosChain)), 0, "")
		s.AssertTxSuccess(transferTxResp)

		s.AssertPacketRelayed(ctx, chainA, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainA, chainAAddress, chainAIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA), "failed to wait for blocks")

	t.Run("stop relayer", func(t *testing.T) {
		s.StopRelayer(ctx, relayer)
	})

	t.Run("upgrade chain", func(t *testing.T) {
		govProposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), govProposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("query denoms after upgrade", func(t *testing.T) {
		resp, err := query.TransferDenoms(ctx, chainA)
		s.Require().NoError(err)
		s.Require().Len(resp.Denoms, 1)
		s.Require().Equal(chainAIBCToken, resp.Denoms[0])
	})

	t.Run("IBC token transfer from chainA to chainB, to make sure the upgrade did not break the packet flow", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)

		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 2)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount * 2
		s.Require().Equal(expected, actualBalance.Int64())
	})
}

func (s *UpgradeTestSuite) TestV8ToV10ChainUpgrade_Localhost() {
	t := s.T()
	testCfg := testsuite.LoadConfig()
	ctx := context.Background()

	chainA, chainB := s.GetChains()
	chainADenom := chainA.Config().Denom

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userBWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	var (
		srcChannelID string
		dstChannelID string
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("open localhost channel", func(t *testing.T) {
		var (
			msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse
			msgChanOpenTryRes  channeltypes.MsgChannelOpenTryResponse
		)

		msgChanOpenInit := channeltypes.NewMsgChannelOpenInit(
			transfertypes.PortID, transfertypes.V1,
			channeltypes.UNORDERED, []string{exported.LocalhostConnectionID},
			transfertypes.PortID, rlyWallet.FormattedAddress(),
		)
		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenInit)
		s.AssertTxSuccess(txResp)
		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenInitRes))
		srcChannelID = msgChanOpenInitRes.ChannelId

		msgChanOpenTry := channeltypes.NewMsgChannelOpenTry(
			transfertypes.PortID, transfertypes.V1,
			channeltypes.UNORDERED, []string{exported.LocalhostConnectionID},
			transfertypes.PortID, srcChannelID,
			transfertypes.V1, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)
		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenTry)
		s.AssertTxSuccess(txResp)
		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenTryRes))
		dstChannelID = msgChanOpenTryRes.ChannelId

		msgChanOpenAck := channeltypes.NewMsgChannelOpenAck(
			transfertypes.PortID, srcChannelID,
			dstChannelID, transfertypes.V1,
			localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)
		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenAck)
		s.AssertTxSuccess(txResp)

		msgChanOpenConfirm := channeltypes.NewMsgChannelOpenConfirm(
			transfertypes.PortID, dstChannelID,
			localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress(),
		)
		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenConfirm)
		s.AssertTxSuccess(txResp)
	})

	t.Run("ibc transfer over localhost", func(t *testing.T) {
		txResp := s.Transfer(ctx, chainA, userAWallet, transfertypes.PortID, srcChannelID, testvalues.DefaultTransferAmount(chainADenom), userAWallet.FormattedAddress(), userBWallet.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParseV1PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)

		msgRecvPacket := channeltypes.NewMsgRecvPacket(packet, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())

		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgRecvPacket)
		s.AssertTxSuccess(txResp)

		ack, err := ibctesting.ParseAckFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(ack)

		msgAcknowledgement := channeltypes.NewMsgAcknowledgement(packet, ack, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())
		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgAcknowledgement)
		s.AssertTxSuccess(txResp)

		s.AssertPacketRelayed(ctx, chainA, transfertypes.PortID, srcChannelID, 1)
		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, dstChannelID)
		actualBalance, err := query.Balance(ctx, chainA, userBWallet.FormattedAddress(), ibcToken.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, actualBalance.Int64())
	})

	t.Run("localhost exists in state before upgrade", func(t *testing.T) {
		localhostClientID := exported.LocalhostClientID
		if !testvalues.LocalhostWithDashFeatureReleases.IsSupported(chainA.Config().Images[0].Version) {
			localhostClientID = exported.Localhost
		}

		status, err := query.ClientStatus(ctx, chainA, localhostClientID)
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		state, err := s.ClientState(ctx, chainA, localhostClientID)
		s.Require().NoError(err)
		s.Require().NotNil(state)
	})

	t.Run("upgrade chain", func(t *testing.T) {
		govProposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), govProposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	t.Run("localhost does not exist in state after upgrade", func(t *testing.T) {
		localhostClientID := exported.LocalhostClientID
		if !testvalues.LocalhostWithDashFeatureReleases.IsSupported(chainA.Config().Images[0].Version) {
			localhostClientID = exported.Localhost
		}

		status, err := query.ClientStatus(ctx, chainA, localhostClientID)
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		state, err := s.ClientState(ctx, chainA, localhostClientID)
		s.Require().Error(err)
		s.Require().Nil(state)
	})

	t.Run("query localhost transfer channel ends after upgrade", func(t *testing.T) {
		channelEndA, err := query.Channel(ctx, chainA, transfertypes.PortID, srcChannelID)
		s.Require().NoError(err)
		s.Require().NotNil(channelEndA)

		channelEndB, err := query.Channel(ctx, chainA, transfertypes.PortID, dstChannelID)
		s.Require().NoError(err)
		s.Require().NotNil(channelEndB)

		s.Require().Equal(channelEndA.ConnectionHops, channelEndB.ConnectionHops)
	})

	t.Run("ibc transfer back over localhost after upgrade", func(t *testing.T) {
		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, dstChannelID)
		transferCoins := testvalues.DefaultTransferAmount(ibcToken.IBCDenom())
		txResp := s.Transfer(ctx, chainA, userBWallet, transfertypes.PortID, dstChannelID, transferCoins, userBWallet.FormattedAddress(), userAWallet.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParseV1PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)

		msgRecvPacket := channeltypes.NewMsgRecvPacket(packet, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())

		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgRecvPacket)
		s.AssertTxSuccess(txResp)

		ack, err := ibctesting.ParseAckFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(ack)

		msgAcknowledgement := channeltypes.NewMsgAcknowledgement(packet, ack, localhost.SentinelProof, clienttypes.ZeroHeight(), rlyWallet.FormattedAddress())
		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgAcknowledgement)
		s.AssertTxSuccess(txResp)

		s.AssertPacketRelayed(ctx, chainA, transfertypes.PortID, dstChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainA, userAWallet.FormattedAddress(), chainADenom)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, actualBalance.Int64())
	})
}

// ClientState queries the current ClientState by clientID
func (*UpgradeTestSuite) ClientState(ctx context.Context, chain ibc.Chain, clientID string) (*clienttypes.QueryClientStateResponse, error) {
	res, err := query.GRPCQuery[clienttypes.QueryClientStateResponse](ctx, chain, &clienttypes.QueryClientStateRequest{ClientId: clientID})
	if err != nil {
		return res, err
	}

	return res, nil
}
