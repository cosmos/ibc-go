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

const (
	heightDelta = uint64(20)
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
		// configure chain A
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

		options.ChainASpec.ModifyGenesis = nil
		options.ChainASpec.ConfigFileOverrides = nil
		options.ChainASpec.EncodingConfig = nil

		// configure chain B
		options.ChainBSpec.ChainName = "simd" // Set chain name so that a suffix with a "dash" is not appended (required for hyperspace)
		options.ChainBSpec.Type = "cosmos"
		options.ChainBSpec.Name = "simd"
		options.ChainBSpec.ChainID = "simd"
		options.ChainBSpec.Images = []ibc.DockerImage{
			{
				Repository: "chatton/ibc-go-simd-wasm",
				Version:    "latest",
				UidGid:     "1000:1000",
			},
		}
		options.ChainBSpec.Bin = "simd"
		options.ChainBSpec.Bech32Prefix = "cosmos"
		options.ChainBSpec.Denom = "stake"
		options.ChainBSpec.GasPrices = "0.00stake"
		options.ChainBSpec.GasAdjustment = 1
		options.ChainBSpec.TrustingPeriod = "504h"
		options.ChainBSpec.CoinType = "118"

		options.ChainBSpec.ChainConfig.NoHostMount = false
		options.ChainBSpec.ConfigFileOverrides = getConfigOverrides()
		//options.ChainBSpec.ModifyGenesis = modifyGenesisShortProposals(votingPeriod, maxDepositPeriod)
		//options.ChainBSpec.EncodingConfig = nil
	})

	polkadotChain := chainA.(*polkadot.PolkadotChain)
	cosmosChain := chainB.(*cosmos.CosmosChain)

	// we explicitly skip path creation as the contract needs to be uploaded before we can create clients.
	r := s.ConfigureRelayer(ctx, polkadotChain, cosmosChain, nil, func(options *interchaintest.InterchainBuildOptions) {
		options.SkipPathCreation = true
	})

	s.InitGRPCClients(polkadotChain)
	s.InitGRPCClients(cosmosChain)


	cosmosWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	file, err := os.Open("../data/ics10_grandpa_cw.wasm")
	s.Require().NoError(err)

	codeHash := s.PushNewWasmClientProposal(ctx, cosmosChain, cosmosWallet, file)

	// Create a proposal, vote, and wait for it to pass. Return code hash for relayer.
	//codeHash := s.pushWasmContractViaGov(t, ctx, cosmosChain)
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

type GetCodeQueryMsgResponse struct {
	Data []byte `json:"data"`
}

// PushNewWasmClientProposal submits a new wasm client governance proposal to the chain
func (s *GrandpaTestSuite) PushNewWasmClientProposal(ctx context.Context, chain *cosmos.CosmosChain, wallet ibc.Wallet, proposalContent io.Reader) string {
	content, err := io.ReadAll(proposalContent)
	s.Require().NoError(err)

	codeHashByte32 := sha256.Sum256(content)
	codeHash := hex.EncodeToString(codeHashByte32[:])
	content, err = testutil.GzipIt(content)
	s.Require().NoError(err)
	message := wasmtypes.MsgStoreCode{
		Signer:      authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		WasmByteCode: content,
	}

	s.ExecuteAndPassGovV1Proposal(ctx, &message, chain, wallet)

	var getCodeQueryMsgRsp GetCodeQueryMsgResponse
	err = chain.QueryClientContractCode(ctx, codeHash, &getCodeQueryMsgRsp)
	codeHashByte32 = sha256.Sum256(getCodeQueryMsgRsp.Data)
	codeHash2 := hex.EncodeToString(codeHashByte32[:])
	s.Require().NoError(err)
	s.Require().NotEmpty(getCodeQueryMsgRsp.Data)
	s.Require().Equal(codeHash, codeHash2)

	return codeHash
}

func (s *GrandpaTestSuite) pushWasmContractViaGov(t *testing.T, ctx context.Context, cosmosChain *cosmos.CosmosChain) string {
	// Set up cosmos user for pushing new wasm code msg via governance
	fundAmountForGov := int64(10_000_000_000)
	contractUsers := interchaintest.GetAndFundTestUsers(t, ctx, "default", int64(fundAmountForGov), cosmosChain)
	contractUser := contractUsers[0]

	contractUserBalInitial, err := cosmosChain.GetBalance(ctx, contractUser.FormattedAddress(), cosmosChain.Config().Denom)
	s.Require().NoError(err, "error fetching initial balance of contract user")
	s.Require().Equal(math.NewInt(fundAmountForGov).Int64(), contractUserBalInitial.Int64(), "initial balance of contract user not expected")

	proposal := cosmos.TxProposalv1{
		Metadata: "none",
		Deposit:  "500000000" + cosmosChain.Config().Denom, // greater than min deposit
		Title:    "Grandpa Contract",
		Summary:  "new grandpa contract",
	}

	proposalTx, codeHash, err := cosmosChain.PushNewWasmClientProposal(ctx, contractUser.KeyName(), "../data/ics10_grandpa_cw.wasm", proposal)
	s.Require().NoError(err, "error submitting new wasm contract proposal tx")

	height, err := cosmosChain.Height(ctx)
	s.Require().NoError(err, "error fetching height before submit upgrade proposal")

	err = cosmosChain.VoteOnProposalAllValidators(ctx, proposalTx.ProposalID, cosmos.ProposalVoteYes)
	s.Require().NoError(err, "failed to submit votes")

	_, err = cosmos.PollForProposalStatus(ctx, cosmosChain, height, height+heightDelta, proposalTx.ProposalID, cosmos.ProposalStatusPassed)
	s.Require().NoError(err, "proposal status did not change to passed in expected number of blocks")

	err = testutil.WaitForBlocks(ctx, 1, cosmosChain)
	s.Require().NoError(err)

	var getCodeQueryMsgRsp GetCodeQueryMsgResponse
	err = cosmosChain.QueryClientContractCode(ctx, codeHash, &getCodeQueryMsgRsp)
	codeHashByte32 := sha256.Sum256(getCodeQueryMsgRsp.Data)
	codeHash2 := hex.EncodeToString(codeHashByte32[:])
	s.Require().NoError(err)
	s.Require().NotEmpty(getCodeQueryMsgRsp.Data)
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

//func modifyGenesisShortProposals(votingPeriod string, maxDepositPeriod string) func(ibc.ChainConfig, []byte) ([]byte, error) {
//	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
//		g := make(map[string]interface{})
//		if err := json.Unmarshal(genbz, &g); err != nil {
//			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
//		}
//		if err := dyno.Set(g, votingPeriod, "app_state", "gov", "params", "voting_period"); err != nil {
//			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
//		}
//		if err := dyno.Set(g, maxDepositPeriod, "app_state", "gov", "params", "max_deposit_period"); err != nil {
//			return nil, fmt.Errorf("failed to set max deposit period in genesis json: %w", err)
//		}
//		if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", "params", "min_deposit", 0, "denom"); err != nil {
//			return nil, fmt.Errorf("failed to set min deposit in genesis json: %w", err)
//		}
//		out, err := json.Marshal(g)
//		if err != nil {
//			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
//		}
//		return out, nil
//	}
//}
