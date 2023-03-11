package upgrades

import (
	"context"
	"os"
	"time"
	// "os"
	"encoding/json"
	"testing"

	"github.com/cosmos/ibc-go/e2e/testconfig"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	tmjson "github.com/tendermint/tendermint/libs/json"

	// interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	cosmos "github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	// "github.com/strangelove-ventures/interchaintest/v7/ibc"
	test "github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/suite"

	// "go.uber.org/zap/zaptest"
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
	configFileOverrides["halt-height"] = haltHeight

	testconfig.DefaultChainOptions().ChainAConfig.ConfigFileOverrides = appTomlOverrides

	ctx := context.Background()
	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	var (
		chainADenom    = chainA.Config().Denom
		// chainBIBCToken = testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID) // IBC token sent to chainB

		// chainBDenom    = chainB.Config().Denom
		//chainAIBCToken = testsuite.GetIBCToken(chainBDenom, channelA.PortID, channelA.ChannelID) // IBC token sent to chainA
	)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp, err := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.Require().NoError(err)
		s.AssertValidTxResponse(transferTxResp)
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

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	t.Run("Halt chain and export genesis", func(t *testing.T) {
		s.HaltChainAndExportGenesis(ctx, chainA, int64(haltHeight))
	})

}

func (s *GenesisTestSuite) HaltChainAndExportGenesis(ctx context.Context, chain *cosmos.CosmosChain, haltHeight int64) {
	var genesisState GenesisState
	
	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	height, err := chain.Height(ctx)
	s.Require().NoError(err, "error fetching height before halt")
	
	err = test.WaitForBlocks(timeoutCtx, int(haltHeight - int64(height)) +1, chain)
	s.Require().NoError(err)
	err = chain.StopAllNodes(ctx)
	s.Require().NoError(err, "error stopping node(s)")

	state, err := chain.ExportState(ctx, int64(haltHeight))
	println("check data: ", state)
	s.Require().NoError(err)
	err = tmjson.Unmarshal([]byte(state), &genesisState)
	s.Require().NoError(err)
	genesisJson, err := tmjson.MarshalIndent(genesisState, "", "  ")
	s.Require().NoError(err)

	err = WriteFile("genesis.json", genesisJson)
	s.Require().NoError(err)

}

func WriteFile(path string, body []byte) error {
	_, err := os.Create(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o600)
}
