package upgrades

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	cosmos "github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	test "github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
)

type GenesisState map[string]json.RawMessage

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

type GenesisTestSuite struct {
	testsuite.E2ETestSuite
	cosmos.ChainNode
}

func (s *GenesisTestSuite) TestIBCGenesis() {
	t := s.T()

	configFileOverrides := make(map[string]any)
	appTomlOverrides := make(test.Toml)

	appTomlOverrides["halt-height"] = haltHeight
	configFileOverrides["config/app.toml"] = appTomlOverrides
	chainOpts := func(options *testsuite.ChainOptions) {
		options.ChainAConfig.ConfigFileOverrides = configFileOverrides
	}

	// create chains with specified chain configuration options
	chainA, chainB := s.GetChains(chainOpts)

	ctx := context.Background()
	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx)
	var (
		chainADenom    = chainA.Config().Denom
		chainBIBCToken = testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID) // IBC token sent to chainB

	)

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

		actualBalance, err := chainB.GetBalance(ctx, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")

	t.Run("Halt chain and export genesis", func(t *testing.T) {
		s.HaltChainAndExportGenesis(ctx, chainA, relayer, int64(haltHeight))
	})

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

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")
}

func (s *GenesisTestSuite) HaltChainAndExportGenesis(ctx context.Context, chain *cosmos.CosmosChain, relayer ibc.Relayer, haltHeight int64) {
	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err := test.WaitForBlocks(timeoutCtx, int(haltHeight), chain)
	s.Require().Error(err, "chain did not halt at halt height")

	err = chain.StopAllNodes(ctx)
	s.Require().NoError(err, "error stopping node(s)")

	// relayer should be stopped by now, but just in case
	s.StopRelayer(ctx, relayer)

	state, err := chain.ExportState(ctx, int64(haltHeight))
	s.Require().NoError(err)

	fmt.Println(state)
	// err = tmjson.Unmarshal([]byte(state), &genesisState)
	// s.Require().NoError(err)

	// genesisJson, err := tmjson.MarshalIndent(genesisState, "", "  ")
	// s.Require().NoError(err)

	newGenesisJson := strings.ReplaceAll(state, fmt.Sprintf("\"initial_height\":%d", 0), fmt.Sprintf("\"initial_height\":%d", haltHeight+2))

	chainAN, _ := s.GetChains()
	for _, node := range chainAN.Nodes() {
		err := node.OverwriteGenesisFile(ctx, []byte(newGenesisJson))
		s.Require().NoError(err)
	}
	fmt.Println([]byte(newGenesisJson))
	// for _, node := range chain.Validators {
	// 	err = node.UnsafeResetAll(ctx)
	// 	s.Require().NoError(err)
	// }

	node := chainAN.Validators[0]
	err = node.CreateNodeContainer(ctx)
	s.Require().NoError(err)
	err = node.StartContainer(ctx)
	s.Require().NoError(err)

	// we are reinitializing the clients because we need to update the hostGRPCAddress after
	// halt chain and subsequent restarting of nodes
	s.InitGRPCClients(chain)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err = test.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	s.Require().NoError(err, "chain did not produce blocks after halt")

	height, err := chain.Height(ctx)
	s.Require().NoError(err, "error fetching height after halt")

	s.Require().Greater(height, haltHeight, "height did not increment after halt")
}
