//go:build !test_e2e

package upgrades

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	e2erelayer "github.com/cosmos/ibc-go/e2e/relayer"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	v7migrations "github.com/cosmos/ibc-go/v8/modules/core/02-client/migrations/v7"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

const (
	haltHeight         = uint64(100)
	blocksAfterUpgrade = uint64(10)
)

func TestUpgradeTestSuite(t *testing.T) {
	testCfg := testsuite.LoadConfig()
	if testCfg.UpgradeConfig.Tag == "" || testCfg.UpgradeConfig.PlanName == "" {
		t.Fatalf("%s and %s must be set when running an upgrade test", testsuite.ChainUpgradeTagEnv, testsuite.ChainUpgradePlanEnv)
	}

	testifysuite.Run(t, new(UpgradeTestSuite))
}

type UpgradeTestSuite struct {
	testsuite.E2ETestSuite
}

// UpgradeChain upgrades a chain to a specific version using the planName provided.
// The software upgrade proposal is broadcast by the provided wallet.
func (s *UpgradeTestSuite) UpgradeChain(ctx context.Context, chain *cosmos.CosmosChain, wallet ibc.Wallet, planName, currentVersion, upgradeVersion string) {
	plan := upgradetypes.Plan{
		Name:   planName,
		Height: int64(haltHeight),
		Info:   fmt.Sprintf("upgrade version test from %s to %s", currentVersion, upgradeVersion),
	}

	upgradeProposal := upgradetypes.NewSoftwareUpgradeProposal(fmt.Sprintf("upgrade from %s to %s", currentVersion, upgradeVersion), "upgrade chain E2E test", plan)
	s.ExecuteAndPassGovV1Beta1Proposal(ctx, chain, wallet, upgradeProposal)

	height, err := chain.Height(ctx)
	s.Require().NoError(err, "error fetching height before upgrade")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err = test.WaitForBlocks(timeoutCtx, int(haltHeight-height)+1, chain)
	s.Require().Error(err, "chain did not halt at halt height")

	err = chain.StopAllNodes(ctx)
	s.Require().NoError(err, "error stopping node(s)")

	repository := chain.Nodes()[0].Image.Repository
	chain.UpgradeVersion(ctx, s.DockerClient, repository, upgradeVersion)

	err = chain.StartAllNodes(ctx)
	s.Require().NoError(err, "error starting upgraded node(s)")

	// we are reinitializing the clients because we need to update the hostGRPCAddress after
	// the upgrade and subsequent restarting of nodes
	s.InitGRPCClients(chain)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err = test.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	s.Require().NoError(err, "chain did not produce blocks after upgrade")

	height, err = chain.Height(ctx)
	s.Require().NoError(err, "error fetching height after upgrade")

	s.Require().Greater(height, haltHeight, "height did not increment after upgrade")
}

func (s *UpgradeTestSuite) TestIBCChainUpgrade() {
	t := s.T()
	testCfg := testsuite.LoadConfig()

	ctx := context.Background()
	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, nil)
	chainA, chainB := s.GetChains()

	var (
		chainADenom    = chainA.Config().Denom
		chainBIBCToken = testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID) // IBC token sent to chainB

		chainBDenom    = chainB.Config().Denom
		chainAIBCToken = testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID) // IBC token sent to chainA
	)

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
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := s.QueryBalance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())

		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("upgrade chainA", func(t *testing.T) {
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), chainAUpgradeProposalWallet, testCfg.UpgradeConfig.PlanName, testCfg.ChainConfigs[0].Tag, testCfg.UpgradeConfig.Tag)
	})

	t.Run("restart relayer", func(t *testing.T) {
		s.StopRelayer(ctx, relayer)
		s.StartRelayer(relayer)
	})

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 2)
		actualBalance, err := s.QueryBalance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())

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

			actualBalance, err := s.QueryBalance(ctx, chainA, chainAAddress, chainAIBCToken.IBCDenom())

			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount
			s.Require().Equal(expected, actualBalance.Int64())
		})
	})
}

func (s *UpgradeTestSuite) TestChainUpgrade() {
	t := s.T()

	ctx := context.Background()
	chain := s.SetupSingleChain(ctx)

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
		balance, err := s.QueryBalance(ctx, chain, userWalletAddr, chain.Config().Denom)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount * 2
		s.Require().Equal(expected, balance)
	})

	t.Run("upgrade chain", func(t *testing.T) {
		testCfg := testsuite.LoadConfig()
		proposerWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

		s.UpgradeChain(ctx, chain.(*cosmos.CosmosChain), proposerWallet, testCfg.UpgradeConfig.PlanName, testCfg.ChainConfigs[0].Tag, testCfg.UpgradeConfig.Tag)
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
		balance, err := s.QueryBalance(ctx, chain, userWalletAddr, chain.Config().Denom)
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
	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, nil)
	chainA, chainB := s.GetChains()

	var (
		chainADenom    = chainA.Config().Denom
		chainBIBCToken = testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID) // IBC token sent to chainB
	)

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
		status, err := s.QueryClientStatus(ctx, chainA, testvalues.TendermintClientID(0))
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		status, err = s.QueryClientStatus(ctx, chainA, testvalues.TendermintClientID(1))
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
		status, err := s.QueryClientStatus(ctx, chainA, testvalues.SolomachineClientID(2))
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
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := s.QueryBalance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	// create separate user specifically for the upgrade proposal to more easily verify starting
	// and end balances of the chainA users.
	chainAUpgradeProposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	t.Run("upgrade chainA", func(t *testing.T) {
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), chainAUpgradeProposalWallet, testCfg.UpgradeConfig.PlanName, testCfg.ChainConfigs[0].Tag, testCfg.UpgradeConfig.Tag)
	})

	// see this issue https://github.com/informalsystems/hermes/issues/3579
	// this restart is a temporary workaround to a limitation in hermes requiring a restart
	// in some cases after an upgrade.
	tc := testsuite.LoadConfig()
	if tc.GetActiveRelayerConfig().ID == e2erelayer.Hermes {
		s.RestartRelayer(ctx, relayer)
	}

	t.Run("check that the tendermint clients are active again after upgrade", func(t *testing.T) {
		status, err := s.QueryClientStatus(ctx, chainA, testvalues.TendermintClientID(0))
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		status, err = s.QueryClientStatus(ctx, chainA, testvalues.TendermintClientID(1))
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)
	})

	t.Run("IBC token transfer from chainA to chainB, to make sure the upgrade did not break the packet flow", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB.(*cosmos.CosmosChain)), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := s.QueryBalance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
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
	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, nil)
	chainA, chainB := s.GetChains()

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
		s.StartRelayer(relayer)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packet is relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := s.QueryBalance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA), "failed to wait for blocks")

	t.Run("upgrade chain", func(t *testing.T) {
		govProposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), govProposalWallet, testCfg.UpgradeConfig.PlanName, testCfg.ChainConfigs[0].Tag, testCfg.UpgradeConfig.Tag)
	})

	t.Run("ensure the localhost client is active and sentinel connection is stored in state", func(t *testing.T) {
		status, err := s.QueryClientStatus(ctx, chainA, exported.LocalhostClientID)
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		connectionEnd, err := s.QueryConnection(ctx, chainA, exported.LocalhostConnectionID)
		s.Require().NoError(err)
		s.Require().Equal(connectiontypes.OPEN, connectionEnd.State)
		s.Require().Equal(exported.LocalhostClientID, connectionEnd.ClientId)
		s.Require().Equal(exported.LocalhostClientID, connectionEnd.Counterparty.ClientId)
		s.Require().Equal(exported.LocalhostConnectionID, connectionEnd.Counterparty.ConnectionId)
	})

	t.Run("ensure escrow amount for native denom is stored in state", func(t *testing.T) {
		actualTotalEscrow, err := s.QueryTotalEscrowForDenom(ctx, chainA, chainADenom)
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
	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, nil)
	chainA, chainB := s.GetChains()

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
		s.StartRelayer(relayer)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	t.Run("packet is relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := s.QueryBalance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA), "failed to wait for blocks")

	t.Run("upgrade chain", func(t *testing.T) {
		govProposalWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
		s.UpgradeChain(ctx, chainB.(*cosmos.CosmosChain), govProposalWallet, testCfg.UpgradeConfig.PlanName, testCfg.ChainConfigs[0].Tag, testCfg.UpgradeConfig.Tag)
	})

	t.Run("update params", func(t *testing.T) {
		authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainB)
		s.Require().NoError(err)
		s.Require().NotNil(authority)

		msg := clienttypes.NewMsgUpdateParams(authority.String(), clienttypes.NewParams(exported.Tendermint, "some-client"))
		s.ExecuteAndPassGovV1Proposal(ctx, msg, chainB, chainBWallet)
	})

	t.Run("query params", func(t *testing.T) {
		clientParams, err := s.GetChainGRCPClients(chainB).ClientQueryClient.ClientParams(ctx, &clienttypes.QueryClientParamsRequest{})
		s.Require().NoError(err)

		allowedClients := clientParams.Params.AllowedClients

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

// ClientState queries the current ClientState by clientID
func (s *UpgradeTestSuite) ClientState(ctx context.Context, chain ibc.Chain, clientID string) (*clienttypes.QueryClientStateResponse, error) {
	queryClient := s.GetChainGRCPClients(chain).ClientQueryClient
	res, err := queryClient.ClientState(ctx, &clienttypes.QueryClientStateRequest{
		ClientId: clientID,
	})
	if err != nil {
		return res, err
	}

	return res, nil
}
