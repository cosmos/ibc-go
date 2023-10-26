package wasm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/chain/polkadot"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

func TestGrandpaTestSuite(t *testing.T) {
	validateTestConfig()
	testifysuite.Run(t, new(GrandpaTestSuite))
}

type GrandpaTestSuite struct {
	testsuite.E2ETestSuite
}

// validateTestConfig ensures that the given test config is valid for this test suite.
func validateTestConfig() {
	tc := testsuite.LoadConfig()
	if tc.ActiveRelayer != "hyperspace" {
		panic(fmt.Errorf("hyperspace relayer must be specifed"))
	}
}

func getConfigOverrides() map[string]any {
	consensusOverrides := make(testutil.Toml)
	blockTime := 5 // seconds, parachain is 12 second blocks, don't make relayer work harder than needed
	blockT := (time.Duration(blockTime) * time.Second).String()
	consensusOverrides["timeout_commit"] = blockT
	consensusOverrides["timeout_propose"] = blockT

	configTomlOverrides := make(testutil.Toml)
	configTomlOverrides["consensus"] = consensusOverrides
	configTomlOverrides["log_level"] = "info"

	configFileOverrides := make(map[string]any)
	configFileOverrides["config/config.toml"] = configTomlOverrides
	return configFileOverrides
}

// TestMsgTransfer_Succeeds_GrandpaContract features
// * sets up a Polkadot parachain
// * sets up a Cosmos chain
// * sets up the Hyperspace relayer
// * Funds a user wallet on both chains
// * Pushes a wasm client contract to the Cosmos chain
// * create client, connection, and channel in relayer
// * start relayer
// * send transfer over ibc
func (s *GrandpaTestSuite) TestMsgTransfer_Succeeds_GrandpaContract() {
	t := s.T()

	ctx := context.Background()

	chainA, chainB := s.GetChains(func(options *testsuite.ChainOptions) {
		// configure chain A (polkadot)
		options.ChainASpec.ChainName = "composable"
		options.ChainASpec.Type = "polkadot"
		options.ChainASpec.ChainID = "rococo-local"
		options.ChainASpec.Name = "composable"
		options.ChainASpec.Images = []ibc.DockerImage{
			{
				Repository: "ghcr.io/misko9/polkadot-node",
				Version:    "local",
				UidGid:     "1000:1000",
			},
			{
				Repository: "ghcr.io/misko9/parachain-node",
				Version:    "latest",
				UidGid:     "1000:1000",
			},
		}
		options.ChainASpec.Bin = "polkadot"
		options.ChainASpec.Bech32Prefix = "composable"
		options.ChainASpec.Denom = "uDOT"
		options.ChainASpec.GasPrices = ""
		options.ChainASpec.GasAdjustment = 0
		options.ChainASpec.TrustingPeriod = ""
		options.ChainASpec.CoinType = "354"

		// these values are set by default for our cosmos chains, we need to explicitly remove them here.
		options.ChainASpec.ModifyGenesis = nil
		options.ChainASpec.ConfigFileOverrides = nil
		options.ChainASpec.EncodingConfig = nil

		// configure chain B (cosmos)
		options.ChainBSpec.ChainName = "simd" // Set chain name so that a suffix with a "dash" is not appended (required for hyperspace)
		options.ChainBSpec.Type = "cosmos"
		options.ChainBSpec.Name = "simd"
		options.ChainBSpec.ChainID = "simd"
		options.ChainBSpec.Images = []ibc.DockerImage{
			{
				Repository: "chatton/ibc-go-simd-wasm",
				Version:    "sdk50",
				UidGid:     "1000:1000",
			},
		}
		options.ChainBSpec.Bin = "simd"
		options.ChainBSpec.Bech32Prefix = "cosmos"

		// TODO: hyperspace relayer assumes a denom of "stake", hard code this here for now.
		options.ChainBSpec.Denom = "stake"
		options.ChainBSpec.GasPrices = "0.00stake"
		options.ChainBSpec.GasAdjustment = 1
		options.ChainBSpec.TrustingPeriod = "504h"
		options.ChainBSpec.CoinType = "118"

		options.ChainBSpec.ChainConfig.NoHostMount = false
		options.ChainBSpec.ConfigFileOverrides = getConfigOverrides()
		options.ChainBSpec.EncodingConfig = testsuite.SDKEncodingConfig()
	})

	polkadotChain := chainA.(*polkadot.PolkadotChain)
	cosmosChain := chainB.(*cosmos.CosmosChain)

	// we explicitly skip path creation as the contract needs to be uploaded before we can create clients.
	r := s.ConfigureRelayer(ctx, polkadotChain, cosmosChain, nil, func(options *interchaintest.InterchainBuildOptions) {
		options.SkipPathCreation = true
	})

	s.InitGRPCClients(cosmosChain)

	var err error
	cosmosWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	//
	file, err := os.Open("../data/ics10_grandpa_cw.wasm")
	s.Require().NoError(err)

	codeHash := s.PushNewWasmClientProposal(ctx, cosmosChain, cosmosWallet, file)

	s.Require().NotEmpty(codeHash, "codehash was empty but should not have been")

	eRep := s.GetRelayerExecReporter()

	// Set client contract hash in cosmos chain config
	err = r.SetClientContractHash(ctx, eRep, cosmosChain.Config(), codeHash)
	s.Require().NoError(err)

	// Ensure parachain has started (starts 1 session/epoch after relay chain)
	err = testutil.WaitForBlocks(ctx, 1, polkadotChain)
	s.Require().NoError(err, "polkadot chain failed to make blocks")

	// Fund users on both cosmos and parachain, mints Asset 1 for Alice
	fundAmount := int64(12_333_000_000_000)
	polkadotUser, cosmosUser := s.fundUsers(t, ctx, fundAmount, polkadotChain, cosmosChain)

	pathName := s.GetPathName(0)

	err = r.GeneratePath(ctx, eRep, cosmosChain.Config().ChainID, polkadotChain.Config().ChainID, pathName)
	s.Require().NoError(err)

	// Create new clients
	err = r.CreateClients(ctx, eRep, pathName, ibc.DefaultClientOpts())
	s.Require().NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, cosmosChain, polkadotChain) // these 1 block waits seem to be needed to reduce flakiness
	s.Require().NoError(err)

	// Create a new connection
	err = r.CreateConnections(ctx, eRep, pathName)
	s.Require().NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, cosmosChain, polkadotChain)
	s.Require().NoError(err)

	// Create a new channel & get channels from each chain
	err = r.CreateChannel(ctx, eRep, pathName, ibc.DefaultChannelOpts())
	s.Require().NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, cosmosChain, polkadotChain)
	s.Require().NoError(err)

	// Start relayer
	s.Require().NoError(r.StartRelayer(ctx, eRep, pathName))

	// Send 1.77 stake from cosmosUser to parachainUser
	amountToSend := int64(1_770_000)
	transfer := ibc.WalletAmount{
		Address: polkadotUser.FormattedAddress(),
		Denom:   cosmosChain.Config().Denom,
		Amount:  math.NewInt(amountToSend),
	}
	tx, err := cosmosChain.SendIBCTransfer(ctx, "channel-0", cosmosUser.KeyName(), transfer, ibc.TransferOptions{})
	s.Require().NoError(err)
	s.Require().NoError(tx.Validate()) // test source wallet has decreased funds
	err = testutil.WaitForBlocks(ctx, 15, cosmosChain, polkadotChain)
	s.Require().NoError(err)

	// Verify tokens arrived on parachain user
	parachainUserStake, err := polkadotChain.GetIbcBalance(ctx, string(polkadotUser.Address()), 2)
	s.Require().NoError(err)
	s.Require().Equal(amountToSend, parachainUserStake.Amount.Int64(), "parachain user's stake amount not expected after first tx")

	// Send 1.16 stake from parachainUser to cosmosUser
	amountToReflect := int64(1_160_000)
	reflectTransfer := ibc.WalletAmount{
		Address: cosmosUser.FormattedAddress(),
		Denom:   "2", // stake
		Amount:  math.NewInt(amountToReflect),
	}
	_, err = polkadotChain.SendIBCTransfer(ctx, "channel-0", polkadotUser.KeyName(), reflectTransfer, ibc.TransferOptions{})
	s.Require().NoError(err)

	// Send 1.88 "UNIT" from Alice to cosmosUser
	amountUnits := math.NewInt(1_880_000_000_000)
	unitTransfer := ibc.WalletAmount{
		Address: cosmosUser.FormattedAddress(),
		Denom:   "1", // UNIT
		Amount:  amountUnits,
	}
	_, err = polkadotChain.SendIBCTransfer(ctx, "channel-0", "alice", unitTransfer, ibc.TransferOptions{})
	s.Require().NoError(err)

	// Wait for MsgRecvPacket on cosmos chain
	finalStakeBal := math.NewInt(fundAmount - amountToSend + amountToReflect)
	err = cosmos.PollForBalance(ctx, cosmosChain, 20, ibc.WalletAmount{
		Address: cosmosUser.FormattedAddress(),
		Denom:   cosmosChain.Config().Denom,
		Amount:  finalStakeBal,
	})
	s.Require().NoError(err)

	// Wait for a new update state
	err = testutil.WaitForBlocks(ctx, 5, cosmosChain, polkadotChain)
	s.Require().NoError(err)

	// Verify cosmos user's final "stake" balance
	cosmosUserStakeBal, err := cosmosChain.GetBalance(ctx, cosmosUser.FormattedAddress(), cosmosChain.Config().Denom)
	s.Require().NoError(err)
	s.Require().True(cosmosUserStakeBal.Equal(finalStakeBal))

	// Verify cosmos user's final "unit" balance
	unitDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", "channel-0", "UNIT"))
	cosmosUserUnitBal, err := cosmosChain.GetBalance(ctx, cosmosUser.FormattedAddress(), unitDenomTrace.IBCDenom())
	s.Require().NoError(err)
	s.Require().True(cosmosUserUnitBal.Equal(amountUnits))

	// Verify parachain user's final "unit" balance (will be less than expected due gas costs for stake tx)
	parachainUserUnits, err := polkadotChain.GetIbcBalance(ctx, string(polkadotUser.Address()), 1)
	s.Require().NoError(err)
	s.Require().True(parachainUserUnits.Amount.LTE(math.NewInt(fundAmount)), "parachain user's final unit amount not expected")

	// Verify parachain user's final "stake" balance
	parachainUserStake, err = polkadotChain.GetIbcBalance(ctx, string(polkadotUser.Address()), 2)
	s.Require().NoError(err)
	s.Require().True(parachainUserStake.Amount.Equal(math.NewInt(amountToSend-amountToReflect)), "parachain user's final stake amount not expected")

}

// extractCodeHashFromGzippedContent takes a gzipped wasm contract and returns the codehash.
func (s *GrandpaTestSuite) extractCodeHashFromGzippedContent(zippedContent []byte) string {
	content, err := wasmtypes.Uncompress(zippedContent, wasmtypes.MaxWasmByteSize())
	s.Require().NoError(err)

	codeHashByte32 := sha256.Sum256(content)
	return hex.EncodeToString(codeHashByte32[:])
}

// PushNewWasmClientProposal submits a new wasm client governance proposal to the chain.
func (s *GrandpaTestSuite) PushNewWasmClientProposal(ctx context.Context, chain *cosmos.CosmosChain, wallet ibc.Wallet, proposalContentReader io.Reader) string {
	zippedContent, err := io.ReadAll(proposalContentReader)
	s.Require().NoError(err)

	codeHash := s.extractCodeHashFromGzippedContent(zippedContent)

	s.Require().NoError(err)
	message := wasmtypes.MsgStoreCode{
		Signer:       authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		WasmByteCode: zippedContent,
	}

	s.ExecuteAndPassGovV1Proposal(ctx, &message, chain, wallet)

	codeBz, err := s.QueryWasmCode(ctx, chain, codeHash)

	codeHashByte32 := sha256.Sum256(codeBz)
	codeHash2 := hex.EncodeToString(codeHashByte32[:])
	s.Require().Equal(codeHash, codeHash2)

	return codeHash
}

func (s *GrandpaTestSuite) fundUsers(t *testing.T, ctx context.Context, fundAmount int64, polkadotChain ibc.Chain, cosmosChain ibc.Chain) (ibc.Wallet, ibc.Wallet) {
	users := interchaintest.GetAndFundTestUsers(t, ctx, "user", fundAmount, polkadotChain, cosmosChain)
	polkadotUser, cosmosUser := users[0], users[1]
	err := testutil.WaitForBlocks(ctx, 2, polkadotChain, cosmosChain) // Only waiting 1 block is flaky for parachain
	s.Require().NoError(err, "cosmos or polkadot chain failed to make blocks")

	// Check balances are correct
	amount := math.NewInt(fundAmount)
	polkadotUserAmount, err := polkadotChain.GetBalance(ctx, polkadotUser.FormattedAddress(), polkadotChain.Config().Denom)
	s.Require().NoError(err)
	s.Require().True(polkadotUserAmount.Equal(amount), "Initial polkadot user amount not expected")

	parachainUserAmount, err := polkadotChain.GetBalance(ctx, polkadotUser.FormattedAddress(), "")
	s.Require().NoError(err)
	s.Require().True(parachainUserAmount.Equal(amount), "Initial parachain user amount not expected")

	cosmosUserAmount, err := cosmosChain.GetBalance(ctx, cosmosUser.FormattedAddress(), cosmosChain.Config().Denom)
	s.Require().NoError(err)
	s.Require().True(cosmosUserAmount.Equal(amount), "Initial cosmos user amount not expected")

	return polkadotUser, cosmosUser
}
