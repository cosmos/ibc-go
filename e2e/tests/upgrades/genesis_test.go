//go:build !test_e2e

package upgrades

import (
	"context"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/strangelove-ventures/interchaintest/v8"
	cosmos "github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

type GenesisTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *GenesisTestSuite) TestIBCGenesis() {
	t := s.T()

	configFileOverrides := make(map[string]any)
	appTomlOverrides := make(test.Toml)

	appTomlOverrides["halt-height"] = haltHeight
	configFileOverrides["config/app.toml"] = appTomlOverrides
	chainOpts := func(options *testsuite.ChainOptions) {
		options.ChainASpec.ConfigFileOverrides = configFileOverrides
	}

	// create chains with specified chain configuration options
	chainA, chainB := s.GetChains(chainOpts)

	ctx := context.Background()
	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, nil)
	var (
		chainADenom    = chainA.Config().Denom
		chainBIBCToken = testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID) // IBC token sent to chainB

	)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("ics20: native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("ics20: tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	controllerAddress := controllerAccount.FormattedAddress()
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	var hostAccount string

	t.Run("ics27: broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		// explicitly set the version string because we don't want to use incentivized channels.
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAddress, version)

		txResp := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.AssertTxSuccess(txResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("ics20: packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := s.QueryBalance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())

		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	t.Run("ics27: verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = s.QueryInterchainAccount(ctx, chainA, controllerAddress, ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(hostAccount))

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")

	t.Run("Halt chain and export genesis", func(t *testing.T) {
		s.HaltChainAndExportGenesis(ctx, chainA.(*cosmos.CosmosChain), relayer, int64(haltHeight))
	})

	t.Run("ics20: native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("ics20: tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - 2*testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("ics27: interchain account executes a bank transfer on behalf of the corresponding owner account", func(t *testing.T) {
		t.Run("fund interchain account wallet", func(t *testing.T) {
			// fund the host account so it has some $$ to send
			err := chainB.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
				Address: hostAccount,
				Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
				Denom:   chainB.Config().Denom,
			})
			s.Require().NoError(err)
		})

		t.Run("broadcast MsgSendTx", func(t *testing.T) {
			// assemble bank transfer message from host account to user account on host chain
			msgSend := &banktypes.MsgSend{
				FromAddress: hostAccount,
				ToAddress:   chainBAccount.FormattedAddress(),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
			}

			cdc := testsuite.Codec()
			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(controllerAccount.FormattedAddress(), ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

			resp := s.BroadcastMessages(
				ctx,
				chainA,
				controllerAccount,
				msgSendTx,
			)

			s.AssertTxSuccess(resp)

			s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))
		})
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")
}

func (s *GenesisTestSuite) HaltChainAndExportGenesis(ctx context.Context, chain *cosmos.CosmosChain, relayer ibc.Relayer, haltHeight int64) {
	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err := test.WaitForBlocks(timeoutCtx, int(haltHeight), chain)
	s.Require().Error(err, "chain did not halt at halt height")

	err = chain.StopAllNodes(ctx)
	s.Require().NoError(err, "error stopping node(s)")

	state, err := chain.ExportState(ctx, haltHeight)
	s.Require().NoError(err)

	appTomlOverrides := make(test.Toml)

	appTomlOverrides["halt-height"] = 0

	for _, node := range chain.Nodes() {
		err := node.OverwriteGenesisFile(ctx, []byte(state))
		s.Require().NoError(err)
	}

	for _, node := range chain.Nodes() {
		err := test.ModifyTomlConfigFile(
			ctx,
			zap.NewExample(),
			node.DockerClient,
			node.TestName,
			node.VolumeName,
			"config/app.toml",
			appTomlOverrides,
		)
		s.Require().NoError(err)

		_, _, err = node.ExecBin(ctx, "comet", "unsafe-reset-all")
		s.Require().NoError(err)
	}

	err = chain.StartAllNodes(ctx)
	s.Require().NoError(err)

	// we are reinitializing the clients because we need to update the hostGRPCAddress after
	// the upgrade and subsequent restarting of nodes
	s.InitGRPCClients(chain)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err = test.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	s.Require().NoError(err, "chain did not produce blocks after halt")

	height, err := chain.Height(ctx)
	s.Require().NoError(err, "error fetching height after halt")

	s.Require().Greater(int64(height), haltHeight, "height did not increment after halt")
}
