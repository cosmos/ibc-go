//go:build !test_e2e

package upgrades

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	interchaintest "github.com/strangelove-ventures/interchaintest/v9"
	"github.com/strangelove-ventures/interchaintest/v9/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v9/ibc"
	test "github.com/strangelove-ventures/interchaintest/v9/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	govtypes "cosmossdk.io/x/gov/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	e2erelayer "github.com/cosmos/ibc-go/e2e/relayer"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	feetypes "github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	v7migrations "github.com/cosmos/ibc-go/v9/modules/core/02-client/migrations/v7"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v9/modules/light-clients/06-solomachine"
	localhost "github.com/cosmos/ibc-go/v9/modules/light-clients/09-localhost"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
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

func (s *UpgradeTestSuite) CreateUpgradeTestPath(testName string) (ibc.Relayer, ibc.ChannelOutput) {
	return s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName), s.GetChainAChannelForTest(testName)
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

	msgSoftwareUpgrade := &upgradetypes.MsgSoftwareUpgrade{
		Plan:      plan,
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	}

	s.ExecuteAndPassGovV1Proposal(ctx, msgSoftwareUpgrade, chain, wallet)

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
}

func (s *UpgradeTestSuite) TestIBCChainUpgrade() {
	t := s.T()
	testCfg := testsuite.LoadConfig()

	ctx := context.Background()
	testName := t.Name()
	relayer, channelA := s.CreateUpgradeTestPath(testName)

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
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
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
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
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
			transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferCoins(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "", nil)
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
	s.CreateUpgradeTestPath(testName)

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

	relayer, channelA := s.CreateUpgradeTestPath(testName)

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
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
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
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB.(*cosmos.CosmosChain)), 0, "", nil)
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

	relayer, channelA := s.CreateUpgradeTestPath(testName)

	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("transfer native tokens from chainA to chainB", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB.(*cosmos.CosmosChain)), 0, "", nil)
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
		status, err := query.ClientStatus(ctx, chainA, exported.LocalhostClientID)
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		connectionResp, err := query.GRPCQuery[connectiontypes.QueryConnectionResponse](ctx, chainA, &connectiontypes.QueryConnectionRequest{ConnectionId: exported.LocalhostConnectionID})
		s.Require().NoError(err)
		s.Require().Equal(connectiontypes.OPEN, connectionResp.Connection.State)
		s.Require().Equal(exported.LocalhostClientID, connectionResp.Connection.ClientId)
		s.Require().Equal(exported.LocalhostClientID, connectionResp.Connection.Counterparty.ClientId)
		s.Require().Equal(exported.LocalhostConnectionID, connectionResp.Connection.Counterparty.ConnectionId)
	})

	t.Run("ensure escrow amount for native denom is stored in state", func(t *testing.T) {
		actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainA, chainADenom)
		s.Require().NoError(err)

		expectedTotalEscrow := sdk.NewCoin(chainADenom, sdkmath.NewInt(testvalues.IBCTransferAmount))
		s.Require().Equal(expectedTotalEscrow, actualTotalEscrow) // migration has run and total escrow amount has been set
	})

	t.Run("IBC token transfer from chainA to chainB, to make sure the upgrade did not break the packet flow", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
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

	relayer, channelA := s.CreateUpgradeTestPath(testName)

	chainA, chainB := s.GetChains()

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("transfer native tokens from chainA to chainB", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
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
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
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
	relayer := s.CreatePaths(ibc.DefaultClientOpts(), s.FeeTransferChannelOptions(), testName)

	channelA := s.GetChainAChannelForTest(testName)

	chainA, chainB := s.GetChains()
	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("transfer native tokens from chainA to chainB", func(t *testing.T) {
		txResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
		s.AssertTxSuccess(txResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("pay packet fee", func(t *testing.T) {
		t.Run("no packet fees in escrow", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		testFee := testvalues.DefaultFee(chainADenom)
		packetID := channeltypes.NewPacketID(channelA.PortID, channelA.ChannelID, 1)
		packetFee := feetypes.NewPacketFee(testFee, chainAWallet.FormattedAddress(), nil)

		t.Run("pay packet fee", func(t *testing.T) {
			txResp := s.PayPacketFeeAsync(ctx, chainA, chainAWallet, packetID, packetFee)
			s.AssertTxSuccess(txResp)
		})

		t.Run("query incentivized packets", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.Equal(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.Equal(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.Equal(testFee.TimeoutFee))
		})
	})

	t.Run("upgrade chain", func(t *testing.T) {
		testCfg := testsuite.LoadConfig()
		proposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), proposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	t.Run("29-fee migration partially refunds escrowed tokens", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		testFee := testvalues.DefaultFee(chainADenom)
		legacyTotal := testFee.RecvFee.Add(testFee.AckFee...).Add(testFee.TimeoutFee...)
		refundCoins := legacyTotal.Sub(testFee.Total()...) // Total() returns the denomwise max of (recvFee + ackFee, timeoutFee)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount - legacyTotal.AmountOf(chainADenom).Int64() + refundCoins.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)

		// query incentivised packets and assert calculated values are correct
		packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Len(packets, 1)
		actualFee := packets[0].PacketFees[0].Fee

		s.Require().True(actualFee.RecvFee.Equal(testFee.RecvFee))
		s.Require().True(actualFee.AckFee.Equal(testFee.AckFee))
		s.Require().True(actualFee.TimeoutFee.Equal(testFee.TimeoutFee))

		escrowBalance, err := query.Balance(ctx, chainA, authtypes.NewModuleAddress(feetypes.ModuleName).String(), chainADenom)
		s.Require().NoError(err)

		expected = testFee.Total().AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, escrowBalance.Int64())
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
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
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

func (s *UpgradeTestSuite) TestV8ToV8_1ChainUpgrade_FeeMiddlewareChannelUpgrade() {
	t := s.T()
	testCfg := testsuite.LoadConfig()
	ctx := context.Background()

	testName := t.Name()

	relayer, channelA := s.CreateUpgradeTestPath(testName)

	channelB := channelA.Counterparty

	chainA, chainB := s.GetChains()
	chainADenom := chainA.Config().Denom
	chainBDenom := chainB.Config().Denom
	chainAIBCToken := testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID)
	_ = chainAIBCToken
	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelB.PortID, channelB.ChannelID)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	var (
		chainARelayerWallet, chainBRelayerWallet ibc.Wallet
		relayerAStartingBalance                  int64
		testFee                                  = testvalues.DefaultFee(chainADenom)
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	// trying to create some inflight packets, although they might get relayed before the upgrade starts
	t.Run("create inflight transfer packets between chain A and chain B", func(t *testing.T) {
		chainBWalletAmount := ibc.WalletAmount{
			Address: chainBWallet.FormattedAddress(), // destination address
			Denom:   chainADenom,
			Amount:  sdkmath.NewInt(testvalues.IBCTransferAmount),
		}

		transferTxResp, err := chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName(), chainBWalletAmount, ibc.TransferOptions{})
		s.Require().NoError(err)
		s.Require().NoError(transferTxResp.Validate(), "chain-a ibc transfer tx is invalid")

		chainAwalletAmount := ibc.WalletAmount{
			Address: chainAWallet.FormattedAddress(), // destination address
			Denom:   chainBDenom,
			Amount:  sdkmath.NewInt(testvalues.IBCTransferAmount),
		}

		transferTxResp, err = chainB.SendIBCTransfer(ctx, channelB.ChannelID, chainBWallet.KeyName(), chainAwalletAmount, ibc.TransferOptions{})
		s.Require().NoError(err)
		s.Require().NoError(transferTxResp.Validate(), "chain-b ibc transfer tx is invalid")
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("upgrade chains", func(t *testing.T) {
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()

			t.Run("chain A", func(t *testing.T) {
				govProposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
				s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), govProposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
			})
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			t.Run("chain B", func(t *testing.T) {
				govProposalWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
				s.UpgradeChain(ctx, chainB.(*cosmos.CosmosChain), govProposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[1].Tag, testCfg.GetUpgradeConfig().Tag)
			})
		}()

		wg.Wait()
	})

	t.Run("query params", func(t *testing.T) {
		t.Run("on chain A", func(t *testing.T) {
			channelParamsResp, err := query.GRPCQuery[channeltypes.QueryChannelParamsResponse](ctx, chainA, &channeltypes.QueryChannelParamsRequest{})
			s.Require().NoError(err)

			upgradeTimeout := channelParamsResp.Params.UpgradeTimeout
			s.Require().Equal(clienttypes.ZeroHeight(), upgradeTimeout.Height)
			s.Require().Equal(uint64(time.Minute*10), upgradeTimeout.Timestamp)
		})

		t.Run("on chain B", func(t *testing.T) {
			channelParamsResp, err := query.GRPCQuery[channeltypes.QueryChannelParamsResponse](ctx, chainB, &channeltypes.QueryChannelParamsRequest{})
			s.Require().NoError(err)

			upgradeTimeout := channelParamsResp.Params.UpgradeTimeout
			s.Require().Equal(clienttypes.ZeroHeight(), upgradeTimeout.Height)
			s.Require().Equal(uint64(time.Minute*10), upgradeTimeout.Timestamp)
		})
	})

	t.Run("execute gov proposal to initiate channel upgrade", func(t *testing.T) {
		chA, err := query.Channel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)

		s.InitiateChannelUpgrade(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, s.CreateUpgradeFields(chA))
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")

	t.Run("packets are relayed between chain A and chain B", func(t *testing.T) {
		// packet from chain A to chain B
		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)
		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())

		// packet from chain B to chain A
		actualBalance, err = query.Balance(ctx, chainA, chainAAddress, chainAIBCToken.IBCDenom())
		s.Require().NoError(err)
		expected = testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	t.Run("verify channel A upgraded and is fee enabled", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)

		// check the channel version include the fee version
		version, err := feetypes.MetadataFromVersion(channel.Version)
		s.Require().NoError(err)
		s.Require().Equal(feetypes.Version, version.FeeVersion, "the channel version did not include ics29")

		// extra check
		feeEnabled, err := query.FeeEnabledChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(true, feeEnabled)
	})

	t.Run("verify channel B upgraded and is fee enabled", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainB, channelB.PortID, channelB.ChannelID)
		s.Require().NoError(err)

		// check the channel version include the fee version
		version, err := feetypes.MetadataFromVersion(channel.Version)
		s.Require().NoError(err)
		s.Require().Equal(feetypes.Version, version.FeeVersion, "the channel version did not include ics29")

		// extra check
		feeEnabled, err := query.FeeEnabledChannel(ctx, chainB, channelB.PortID, channelB.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(true, feeEnabled)
	})

	t.Run("prune packet acknowledgements", func(t *testing.T) {
		// there should be one ack for the packet that we sent before the upgrade
		acks, err := query.PacketAcknowledgements(ctx, chainA, channelA.PortID, channelA.ChannelID, []uint64{})
		s.Require().NoError(err)
		s.Require().Len(acks, 1)
		s.Require().Equal(uint64(1), acks[0].Sequence)

		pruneAcksTxResponse := s.PruneAcknowledgements(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, uint64(1))
		s.AssertTxSuccess(pruneAcksTxResponse)

		// after pruning there should not be any acks
		acks, err = query.PacketAcknowledgements(ctx, chainA, channelA.PortID, channelA.ChannelID, []uint64{})
		s.Require().NoError(err)
		s.Require().Empty(acks)
	})

	t.Run("stop relayer", func(t *testing.T) {
		s.StopRelayer(ctx, relayer)
	})

	t.Run("recover relayer wallets", func(t *testing.T) {
		_, _, err := s.RecoverRelayerWallets(ctx, relayer, testName)
		s.Require().NoError(err)

		chainARelayerWallet, chainBRelayerWallet, err = s.GetRelayerWallets(relayer)
		s.Require().NoError(err)

		relayerAStartingBalance, err = s.GetChainANativeBalance(ctx, chainARelayerWallet)
		s.Require().NoError(err)
		t.Logf("relayer A user starting with balance: %d", relayerAStartingBalance)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("register and verify counterparty payee", func(t *testing.T) {
		_, chainBRelayerUser := s.GetRelayerUsers(ctx, testName)
		resp := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.FormattedAddress(), chainARelayerWallet.FormattedAddress())
		s.AssertTxSuccess(resp)

		address, err := query.CounterPartyPayee(ctx, chainB, chainBRelayerWallet.FormattedAddress(), channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.FormattedAddress(), address)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("send incentivized transfer packet", func(t *testing.T) {
		// before adding fees for the packet, there should not be incentivized packets
		packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)

		transferAmount := testvalues.DefaultTransferAmount(chainA.Config().Denom)

		msgPayPacketFee := feetypes.NewMsgPayPacketFee(testFee, channelA.PortID, channelA.ChannelID, chainAWallet.FormattedAddress(), nil)
		msgTransfer := testsuite.GetMsgTransfer(
			channelA.PortID,
			channelA.ChannelID,
			channelA.Version, // upgrade adds fee middleware, but keeps transfer version
			sdk.NewCoins(transferAmount),
			chainAWallet.FormattedAddress(),
			chainBWallet.FormattedAddress(),
			s.GetTimeoutHeight(ctx, chainB),
			0,
			"",
			nil,
		)
		resp := s.BroadcastMessages(ctx, chainA, chainAWallet, msgPayPacketFee, msgTransfer)
		s.AssertTxSuccess(resp)
	})

	t.Run("escrow fees equal (ack fee + recv fee)", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		// walletA has done two IBC transfers of value testvalues.IBCTransferAmount since the start of the test.
		expected := testvalues.StartingTokenAmount - (2 * testvalues.IBCTransferAmount) - testFee.AckFee.AmountOf(chainADenom).Int64() - testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})

	t.Run("tokens are received by walletB", func(t *testing.T) {
		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		// walletB has received two IBC transfers of value testvalues.IBCTransferAmount since the start of the test.
		expected := 2 * testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	t.Run("relayerA is paid ack and recv fee", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainARelayerWallet)
		s.Require().NoError(err)

		expected := relayerAStartingBalance + testFee.AckFee.AmountOf(chainADenom).Int64() + testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})
}

func (s *UpgradeTestSuite) TestV8ToV9ChainUpgrade() {
	t := s.T()
	testCfg := testsuite.LoadConfig()
	ctx := context.Background()

	testName := t.Name()

	relayer, channelA := s.CreateUpgradeTestPath(testName)

	chainA, chainB := s.GetChains()
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
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
		s.AssertTxSuccess(transferTxResp)

		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	t.Run("transfer native tokens from chainB to chainA", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferCoins(chainBDenom), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA.(*cosmos.CosmosChain)), 0, "", nil)
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
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
		s.AssertTxSuccess(transferTxResp)

		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 2)

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount * 2
		s.Require().Equal(expected, actualBalance.Int64())
	})
}

func (s *UpgradeTestSuite) TestV8ToV9ChainUpgrade_Localhost() {
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
		txResp := s.Transfer(ctx, chainA, userAWallet, transfertypes.PortID, srcChannelID, testvalues.DefaultTransferCoins(chainADenom), userAWallet.FormattedAddress(), userBWallet.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "", nil)
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParsePacketFromEvents(txResp.Events)
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
		status, err := query.ClientStatus(ctx, chainA, exported.LocalhostClientID)
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		state, err := s.ClientState(ctx, chainA, exported.LocalhostClientID)
		s.Require().NoError(err)
		s.Require().NotNil(state)
	})

	t.Run("upgrade chain", func(t *testing.T) {
		govProposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), govProposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	t.Run("localhost does not exist in state after upgrade", func(t *testing.T) {
		status, err := query.ClientStatus(ctx, chainA, exported.LocalhostClientID)
		s.Require().NoError(err)
		s.Require().Equal(exported.Active.String(), status)

		state, err := s.ClientState(ctx, chainA, exported.LocalhostClientID)
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
		transferCoins := testvalues.DefaultTransferCoins(ibcToken.IBCDenom())
		txResp := s.Transfer(ctx, chainA, userBWallet, transfertypes.PortID, dstChannelID, transferCoins, userBWallet.FormattedAddress(), userAWallet.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "", nil)
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParsePacketFromEvents(txResp.Events)
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

func (s *UpgradeTestSuite) TestV8ToV9ChainUpgrade_ICS20v2ChannelUpgrade() {
	t := s.T()
	testCfg := testsuite.LoadConfig()
	ctx := context.Background()

	testName := t.Name()

	relayer, channelA := s.CreateUpgradeTestPath(testName)

	chainA, chainB := s.GetChains()
	chainADenom := chainA.Config().Denom
	chainBDenom := chainB.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("transfer native tokens from chainA to chainB", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "", nil)
		s.AssertTxSuccess(transferTxResp)

		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	t.Run("verify channel version before upgrade", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(transfertypes.V1, channel.Version, "the channel version is not ics20-1")
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA), "failed to wait for blocks")

	t.Run("upgrade chain", func(t *testing.T) {
		govProposalWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
		s.UpgradeChain(ctx, chainA.(*cosmos.CosmosChain), govProposalWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	t.Run("upgrade channel to ics20-2", func(t *testing.T) {
		chA, err := query.Channel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)

		upgradeFields := channeltypes.NewUpgradeFields(chA.Ordering, chA.ConnectionHops, transfertypes.V2)
		s.InitiateChannelUpgrade(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, upgradeFields)
	})

	t.Run("verify channel A upgraded and transfer version is ics20-2", func(t *testing.T) {
		err := test.WaitForCondition(time.Minute*3, time.Second*2, func() (bool, error) {
			channel, err := query.Channel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			if err != nil {
				return false, err
			}

			return channel.Version == transfertypes.V2, nil
		})
		s.Require().NoError(err, "failed to wait for channel to be upgraded to ics20-2")
	})

	t.Run("multi-denom IBC token transfer from chainB to chainA, to make sure the upgrade did not break the packet flow", func(t *testing.T) {
		transferCoins := []sdk.Coin{
			testvalues.DefaultTransferAmount(chainBIBCToken.IBCDenom()),
			testvalues.DefaultTransferAmount(chainBDenom),
		}

		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, transferCoins, chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "", nil)
		s.AssertTxSuccess(transferTxResp)

		s.AssertPacketRelayed(ctx, chainB, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 2)

		actualNativeBalance, err := chainA.GetBalance(ctx, chainAAddress, chainADenom)
		s.Require().NoError(err)
		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualNativeBalance.Int64())

		chainAIBCToken := testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID)
		actualIBCTokenBalance, err := query.Balance(ctx, chainA, chainAAddress, chainAIBCToken.IBCDenom())
		s.Require().NoError(err)
		expected = testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualIBCTokenBalance.Int64())
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
