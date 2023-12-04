//go:build !test_e2e

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

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/chain/polkadot"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/math"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

const (
	composable    = "composable"
	simd          = "simd"
	wasmSimdImage = "ghcr.io/cosmos/ibc-go-wasm-simd"

	defaultWasmClientID = "08-wasm-0"
)

func TestGrandpaTestSuite(t *testing.T) {
	// this test suite only works with the hyperspace relayer, for now hard code this here.
	// this will enforce that the hyperspace relayer is used in CI.
	t.Setenv(testsuite.RelayerIDEnv, "hyperspace")

	// TODO: this value should be passed in via the config file / CI, not hard coded in the test.
	// This configuration can be handled in https://github.com/cosmos/ibc-go/issues/4697
	if testsuite.IsCI() && !testsuite.IsFork() {
		t.Setenv(testsuite.ChainImageEnv, wasmSimdImage)
	}

	// wasm tests require a longer voting period to account for the time it takes to upload a contract.
	testvalues.VotingPeriod = time.Minute * 5

	validateTestConfig()
	testifysuite.Run(t, new(GrandpaTestSuite))
}

type GrandpaTestSuite struct {
	testsuite.E2ETestSuite
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
	ctx := context.Background()
	t := s.T()

	chainA, chainB := s.GetGrandpaTestChains()

	polkadotChain := chainA.(*polkadot.PolkadotChain)
	cosmosChain := chainB.(*cosmos.CosmosChain)

	// we explicitly skip path creation as the contract needs to be uploaded before we can create clients.
	r := s.ConfigureRelayer(ctx, polkadotChain, cosmosChain, nil, func(options *interchaintest.InterchainBuildOptions) {
		options.SkipPathCreation = true
	})

	s.InitGRPCClients(cosmosChain)

	cosmosWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	file, err := os.Open("contracts/ics10_grandpa_cw.wasm.gz")
	s.Require().NoError(err)

	checksum := s.PushNewWasmClientProposal(ctx, cosmosChain, cosmosWallet, file)

	s.Require().NotEmpty(checksum, "checksum was empty but should not have been")

	eRep := s.GetRelayerExecReporter()

	// Set client contract hash in cosmos chain config
	err = r.SetClientContractHash(ctx, eRep, cosmosChain.Config(), checksum)
	s.Require().NoError(err)

	// Ensure parachain has started (starts 1 session/epoch after relay chain)
	err = testutil.WaitForBlocks(ctx, 1, polkadotChain)
	s.Require().NoError(err, "polkadot chain failed to make blocks")

	// Fund users on both cosmos and parachain, mints Asset 1 for Alice
	fundAmount := int64(12_333_000_000_000)
	polkadotUser, cosmosUser := s.fundUsers(ctx, fundAmount, polkadotChain, cosmosChain)

	// TODO: this can be refactored to broadcast a MsgTransfer instead of CLI.
	// https://github.com/cosmos/ibc-go/issues/4963
	amountToSend := int64(1_770_000)
	transfer := ibc.WalletAmount{
		Address: polkadotUser.FormattedAddress(),
		Denom:   cosmosChain.Config().Denom,
		Amount:  math.NewInt(amountToSend),
	}

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

	t.Run("send successful IBC transfer from Cosmos to Polkadot parachain", func(t *testing.T) {
		// Send 1.77 stake from cosmosUser to parachainUser
		tx, err := cosmosChain.SendIBCTransfer(ctx, "channel-0", cosmosUser.KeyName(), transfer, ibc.TransferOptions{})
		s.Require().NoError(tx.Validate(), "source ibc transfer tx is invalid")
		s.Require().NoError(err)
		// verify token balance for cosmos user has decreased
		balance, err := cosmosChain.GetBalance(ctx, cosmosUser.FormattedAddress(), cosmosChain.Config().Denom)
		s.Require().NoError(err)
		s.Require().Equal(balance, math.NewInt(fundAmount-amountToSend), "unexpected cosmos user balance after first tx")
		err = testutil.WaitForBlocks(ctx, 15, cosmosChain, polkadotChain)
		s.Require().NoError(err)

		// Verify tokens arrived on parachain user
		parachainUserStake, err := polkadotChain.GetIbcBalance(ctx, string(polkadotUser.Address()), 2)
		s.Require().NoError(err)
		s.Require().Equal(amountToSend, parachainUserStake.Amount.Int64(), "unexpected parachain user balance after first tx")
	})

	t.Run("send two successful IBC transfers from Polkadot parachain to Cosmos, first with ibc denom, second with parachain denom", func(t *testing.T) {
		// Send 1.16 stake from parachainUser to cosmosUser
		amountToReflect := int64(1_160_000)
		reflectTransfer := ibc.WalletAmount{
			Address: cosmosUser.FormattedAddress(),
			Denom:   "2", // stake
			Amount:  math.NewInt(amountToReflect),
		}
		_, err := polkadotChain.SendIBCTransfer(ctx, "channel-0", polkadotUser.KeyName(), reflectTransfer, ibc.TransferOptions{})
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
		parachainUserStake, err := polkadotChain.GetIbcBalance(ctx, string(polkadotUser.Address()), 2)
		s.Require().NoError(err)
		s.Require().True(parachainUserStake.Amount.Equal(math.NewInt(amountToSend-amountToReflect)), "parachain user's final stake amount not expected")
	})
}

// TestMsgTransfer_TimesOut_GrandpaContract
// sets up cosmos and polkadot chains, hyperspace relayer, and funds users on both chains
// * sends transfer over ibc channel, this transfer should timeout
func (s *GrandpaTestSuite) TestMsgTransfer_TimesOut_GrandpaContract() {
	ctx := context.Background()
	t := s.T()

	chainA, chainB := s.GetGrandpaTestChains()

	polkadotChain := chainA.(*polkadot.PolkadotChain)
	cosmosChain := chainB.(*cosmos.CosmosChain)

	// we explicitly skip path creation as the contract needs to be uploaded before we can create clients.
	r := s.ConfigureRelayer(ctx, polkadotChain, cosmosChain, nil, func(options *interchaintest.InterchainBuildOptions) {
		options.SkipPathCreation = true
	})

	s.InitGRPCClients(cosmosChain)

	cosmosWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	file, err := os.Open("contracts/ics10_grandpa_cw.wasm.gz")
	s.Require().NoError(err)

	checksum := s.PushNewWasmClientProposal(ctx, cosmosChain, cosmosWallet, file)

	s.Require().NotEmpty(checksum, "checksum was empty but should not have been")

	eRep := s.GetRelayerExecReporter()

	// Set client contract hash in cosmos chain config
	err = r.SetClientContractHash(ctx, eRep, cosmosChain.Config(), checksum)
	s.Require().NoError(err)

	// Ensure parachain has started (starts 1 session/epoch after relay chain)
	err = testutil.WaitForBlocks(ctx, 1, polkadotChain)
	s.Require().NoError(err, "polkadot chain failed to make blocks")

	// Fund users on both cosmos and parachain, mints Asset 1 for Alice
	fundAmount := int64(12_333_000_000_000)
	polkadotUser, cosmosUser := s.fundUsers(ctx, fundAmount, polkadotChain, cosmosChain)

	// TODO: this can be refactored to broadcast a MsgTransfer instead of CLI.
	// https://github.com/cosmos/ibc-go/issues/4963
	amountToSend := int64(1_770_000)
	transfer := ibc.WalletAmount{
		Address: polkadotUser.FormattedAddress(),
		Denom:   cosmosChain.Config().Denom,
		Amount:  math.NewInt(amountToSend),
	}

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

	t.Run("IBC transfer from Cosmos chain to Polkadot parachain times out", func(t *testing.T) {
		// Stop relayer
		s.Require().NoError(r.StopRelayer(ctx, s.GetRelayerExecReporter()))

		tx, err := cosmosChain.SendIBCTransfer(ctx, "channel-0", cosmosUser.KeyName(), transfer, ibc.TransferOptions{Timeout: testvalues.ImmediatelyTimeout()})
		s.Require().NoError(err)
		s.Require().NoError(tx.Validate(), "source ibc transfer tx is invalid")
		time.Sleep(time.Nanosecond * 1) // want it to timeout immediately

		// check that tokens are escrowed
		actualBalance, err := cosmosChain.GetBalance(ctx, cosmosUser.FormattedAddress(), cosmosChain.Config().Denom)
		s.Require().NoError(err)
		expected := fundAmount - amountToSend
		s.Require().Equal(expected, actualBalance.Int64())

		// start relayer
		s.Require().NoError(r.StartRelayer(ctx, s.GetRelayerExecReporter(), s.GetPathName(0)))
		err = testutil.WaitForBlocks(ctx, 15, polkadotChain, cosmosChain)
		s.Require().NoError(err)

		// ensure that receiver on parachain did not receive any tokens
		receiverBalance, err := polkadotChain.GetIbcBalance(ctx, polkadotUser.FormattedAddress(), 2)
		s.Require().NoError(err)
		s.Require().Equal(int64(0), receiverBalance.Amount.Int64())

		// check that tokens have been refunded to sender address
		senderBalance, err := cosmosChain.GetBalance(ctx, cosmosUser.FormattedAddress(), cosmosChain.Config().Denom)
		s.Require().NoError(err)
		s.Require().Equal(fundAmount, senderBalance.Int64())
	})
}

// TestMsgMigrateContract_Success_GrandpaContract features
// * sets up a Polkadot parachain
// * sets up a Cosmos chain
// * sets up the Hyperspace relayer
// * Funds a user wallet on both chains
// * Pushes a wasm client contract to the Cosmos chain
// * create client in relayer
// * Pushes a new wasm client contract to the Cosmos chain
// * Migrates the wasm client contract
func (s *GrandpaTestSuite) TestMsgMigrateContract_Success_GrandpaContract() {
	ctx := context.Background()

	chainA, chainB := s.GetGrandpaTestChains()

	polkadotChain := chainA.(*polkadot.PolkadotChain)
	cosmosChain := chainB.(*cosmos.CosmosChain)

	// we explicitly skip path creation as the contract needs to be uploaded before we can create clients.
	r := s.ConfigureRelayer(ctx, polkadotChain, cosmosChain, nil, func(options *interchaintest.InterchainBuildOptions) {
		options.SkipPathCreation = true
	})

	s.InitGRPCClients(cosmosChain)

	cosmosWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	file, err := os.Open("contracts/ics10_grandpa_cw.wasm.gz")
	s.Require().NoError(err)

	checksum := s.PushNewWasmClientProposal(ctx, cosmosChain, cosmosWallet, file)

	s.Require().NotEmpty(checksum, "checksum was empty but should not have been")

	eRep := s.GetRelayerExecReporter()

	// Set client contract hash in cosmos chain config
	err = r.SetClientContractHash(ctx, eRep, cosmosChain.Config(), checksum)
	s.Require().NoError(err)

	// Ensure parachain has started (starts 1 session/epoch after relay chain)
	err = testutil.WaitForBlocks(ctx, 1, polkadotChain)
	s.Require().NoError(err, "polkadot chain failed to make blocks")

	pathName := s.GetPathName(0)

	err = r.GeneratePath(ctx, eRep, cosmosChain.Config().ChainID, polkadotChain.Config().ChainID, pathName)
	s.Require().NoError(err)

	// Create new clients
	err = r.CreateClients(ctx, eRep, pathName, ibc.DefaultClientOpts())
	s.Require().NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, cosmosChain, polkadotChain) // these 1 block waits seem to be needed to reduce flakiness
	s.Require().NoError(err)

	// Do not start relayer

	// This contract is a dummy contract that will always succeed migration.
	// Other entry points are unimplemented.
	migrateFile, err := os.Open("contracts/migrate_success.wasm.gz")
	s.Require().NoError(err)

	// First Store the code
	newChecksum := s.PushNewWasmClientProposal(ctx, cosmosChain, cosmosWallet, migrateFile)
	s.Require().NotEmpty(newChecksum, "checksum was empty but should not have been")

	newChecksumBz, err := hex.DecodeString(newChecksum)
	s.Require().NoError(err)

	// Attempt to migrate the contract
	message := wasmtypes.NewMsgMigrateContract(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		defaultWasmClientID,
		newChecksumBz,
		[]byte("{}"),
	)

	s.ExecuteAndPassGovV1Proposal(ctx, message, cosmosChain, cosmosWallet)

	clientState, err := s.QueryClientState(ctx, cosmosChain, defaultWasmClientID)
	s.Require().NoError(err)

	wasmClientState, ok := clientState.(*wasmtypes.ClientState)
	s.Require().True(ok)

	s.Require().Equal(newChecksumBz, wasmClientState.Checksum)
}

// TestMsgMigrateContract_ContractError_GrandpaContract features
// * sets up a Polkadot parachain
// * sets up a Cosmos chain
// * sets up the Hyperspace relayer
// * Funds a user wallet on both chains
// * Pushes a wasm client contract to the Cosmos chain
// * create client in relayer
// * Pushes a new wasm client contract to the Cosmos chain
// * Migrates the wasm client contract with a contract that will always fail migration
func (s *GrandpaTestSuite) TestMsgMigrateContract_ContractError_GrandpaContract() {
	ctx := context.Background()

	chainA, chainB := s.GetGrandpaTestChains()

	polkadotChain := chainA.(*polkadot.PolkadotChain)
	cosmosChain := chainB.(*cosmos.CosmosChain)

	// we explicitly skip path creation as the contract needs to be uploaded before we can create clients.
	r := s.ConfigureRelayer(ctx, polkadotChain, cosmosChain, nil, func(options *interchaintest.InterchainBuildOptions) {
		options.SkipPathCreation = true
	})

	s.InitGRPCClients(cosmosChain)

	cosmosWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	file, err := os.Open("contracts/ics10_grandpa_cw.wasm.gz")
	s.Require().NoError(err)
	checksum := s.PushNewWasmClientProposal(ctx, cosmosChain, cosmosWallet, file)

	s.Require().NotEmpty(checksum, "checksum was empty but should not have been")

	eRep := s.GetRelayerExecReporter()

	// Set client contract hash in cosmos chain config
	err = r.SetClientContractHash(ctx, eRep, cosmosChain.Config(), checksum)
	s.Require().NoError(err)

	// Ensure parachain has started (starts 1 session/epoch after relay chain)
	err = testutil.WaitForBlocks(ctx, 1, polkadotChain)
	s.Require().NoError(err, "polkadot chain failed to make blocks")

	pathName := s.GetPathName(0)

	err = r.GeneratePath(ctx, eRep, cosmosChain.Config().ChainID, polkadotChain.Config().ChainID, pathName)
	s.Require().NoError(err)

	// Create new clients
	err = r.CreateClients(ctx, eRep, pathName, ibc.DefaultClientOpts())
	s.Require().NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, cosmosChain, polkadotChain) // these 1 block waits seem to be needed to reduce flakiness
	s.Require().NoError(err)

	// Do not start the relayer

	// This contract is a dummy contract that will always fail migration.
	// Other entry points are unimplemented.
	migrateFile, err := os.Open("contracts/migrate_error.wasm.gz")
	s.Require().NoError(err)

	// First Store the code
	newChecksum := s.PushNewWasmClientProposal(ctx, cosmosChain, cosmosWallet, migrateFile)
	s.Require().NotEmpty(newChecksum, "checksum was empty but should not have been")

	newChecksumBz, err := hex.DecodeString(newChecksum)
	s.Require().NoError(err)

	// Attempt to migrate the contract
	message := wasmtypes.NewMsgMigrateContract(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		defaultWasmClientID,
		newChecksumBz,
		[]byte("{}"),
	)

	err = s.ExecuteGovV1Proposal(ctx, message, cosmosChain, cosmosWallet)
	// This is the error string that is returned from the contract
	s.Require().ErrorContains(err, "migration not supported")
}

// TestRecoverClient_Succeeds_GrandpaContract features:
// * setup cosmos and polkadot substrates nodes
// * funds test user wallets on both chains
// * stores a wasm client contract on the cosmos chain
// * creates a subject client using the hyperspace relayer
// * waits the expiry period and asserts the subject client status has expired
// * creates a substitute client using the hyperspace relayer
// * executes a gov proposal to recover the expired client
// * asserts the status of the subject client has been restored to active
// NOTE: The testcase features a modified grandpa client contract compiled as:
// - ics10_grandpa_cw_expiry.wasm.gz
// This contract modifies the unbonding period to 1600s with the trusting period being calculated as (unbonding period / 3).
func (s *GrandpaTestSuite) TestRecoverClient_Succeeds_GrandpaContract() {
	ctx := context.Background()

	// set the trusting period to a value which will still be valid upon client creation, but invalid before the first update
	// the contract uses 1600s as the unbonding period with the trusting period evaluating to (unbonding period / 3)
	modifiedTrustingPeriod := (1600 * time.Second) / 3

	chainA, chainB := s.GetGrandpaTestChains()

	polkadotChain := chainA.(*polkadot.PolkadotChain)
	cosmosChain := chainB.(*cosmos.CosmosChain)

	// we explicitly skip path creation as the contract needs to be uploaded before we can create clients.
	r := s.ConfigureRelayer(ctx, polkadotChain, cosmosChain, nil, func(options *interchaintest.InterchainBuildOptions) {
		options.SkipPathCreation = true
	})

	s.InitGRPCClients(cosmosChain)

	cosmosWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	file, err := os.Open("contracts/ics10_grandpa_cw_expiry.wasm.gz")
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
	_, cosmosUser := s.fundUsers(ctx, fundAmount, polkadotChain, cosmosChain)

	pathName := s.GetPathName(0)
	err = r.GeneratePath(ctx, eRep, cosmosChain.Config().ChainID, polkadotChain.Config().ChainID, pathName)
	s.Require().NoError(err)

	// create client pair with subject (bad trusting period)
	subjectClientID := clienttypes.FormatClientIdentifier(ibcexported.Wasm, 0)
	// TODO: The hyperspace relayer makes no use of create client opts
	// https://github.com/strangelove-ventures/interchaintest/blob/main/relayer/hyperspace/hyperspace_commander.go#L83
	s.SetupClients(ctx, r, ibc.CreateClientOptions{
		TrustingPeriod: modifiedTrustingPeriod.String(), // NOTE: this is hardcoded within the cw contract: ics10_grapnda_cw_expiry.wasm
	})

	// wait for block
	err = testutil.WaitForBlocks(ctx, 1, cosmosChain, polkadotChain)
	s.Require().NoError(err)

	// wait the bad trusting period
	time.Sleep(modifiedTrustingPeriod)

	// create client pair with substitute
	substituteClientID := clienttypes.FormatClientIdentifier(ibcexported.Wasm, 1)
	s.SetupClients(ctx, r, ibc.DefaultClientOpts())

	// wait for block
	err = testutil.WaitForBlocks(ctx, 1, cosmosChain, polkadotChain)
	s.Require().NoError(err)

	// ensure subject client is expired
	status, err := s.clientStatus(ctx, cosmosChain, subjectClientID)
	s.Require().NoError(err)
	s.Require().Equal(ibcexported.Expired.String(), status, "unexpected subject client status")

	// ensure substitute client is active
	status, err = s.clientStatus(ctx, cosmosChain, substituteClientID)
	s.Require().NoError(err)
	s.Require().Equal(ibcexported.Active.String(), status, "unexpected substitute client status")

	// create and execute a client recovery proposal
	authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, cosmosChain)
	s.Require().NoError(err)
	msgRecoverClient := clienttypes.NewMsgRecoverClient(authority.String(), subjectClientID, substituteClientID)
	s.Require().NotNil(msgRecoverClient)
	s.ExecuteAndPassGovV1Proposal(ctx, msgRecoverClient, cosmosChain, cosmosUser)

	// ensure subject client is active
	status, err = s.clientStatus(ctx, cosmosChain, subjectClientID)
	s.Require().NoError(err)
	s.Require().Equal(ibcexported.Active.String(), status)

	// ensure substitute client is active
	status, err = s.clientStatus(ctx, cosmosChain, substituteClientID)
	s.Require().NoError(err)
	s.Require().Equal(ibcexported.Active.String(), status)
}

// extractChecksumFromGzippedContent takes a gzipped wasm contract and returns the checksum.
func (s *GrandpaTestSuite) extractChecksumFromGzippedContent(zippedContent []byte) string {
	content, err := wasmtypes.Uncompress(zippedContent, wasmtypes.MaxWasmByteSize())
	s.Require().NoError(err)

	checksum32 := sha256.Sum256(content)
	return hex.EncodeToString(checksum32[:])
}

// PushNewWasmClientProposal submits a new wasm client governance proposal to the chain.
func (s *GrandpaTestSuite) PushNewWasmClientProposal(ctx context.Context, chain *cosmos.CosmosChain, wallet ibc.Wallet, proposalContentReader io.Reader) string {
	zippedContent, err := io.ReadAll(proposalContentReader)
	s.Require().NoError(err)

	computedChecksum := s.extractChecksumFromGzippedContent(zippedContent)

	s.Require().NoError(err)
	message := wasmtypes.MsgStoreCode{
		Signer:       authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		WasmByteCode: zippedContent,
	}

	s.ExecuteAndPassGovV1Proposal(ctx, &message, chain, wallet)

	checksumBz, err := s.QueryWasmCode(ctx, chain, computedChecksum)
	s.Require().NoError(err)

	checksum32 := sha256.Sum256(checksumBz)
	actualChecksum := hex.EncodeToString(checksum32[:])
	s.Require().Equal(computedChecksum, actualChecksum, "checksum returned from query did not match the computed checksum")

	return actualChecksum
}

func (s *GrandpaTestSuite) clientStatus(ctx context.Context, chain ibc.Chain, clientID string) (string, error) {
	queryClient := s.GetChainGRCPClients(chain).ClientQueryClient
	res, err := queryClient.ClientStatus(ctx, &clienttypes.QueryClientStatusRequest{
		ClientId: clientID,
	})
	if err != nil {
		return "", err
	}

	return res.Status, nil
}

func (s *GrandpaTestSuite) fundUsers(ctx context.Context, fundAmount int64, polkadotChain ibc.Chain, cosmosChain ibc.Chain) (ibc.Wallet, ibc.Wallet) {
	users := interchaintest.GetAndFundTestUsers(s.T(), ctx, "user", fundAmount, polkadotChain, cosmosChain)
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

// validateTestConfig ensures that the given test config is valid for this test suite.
func validateTestConfig() {
	tc := testsuite.LoadConfig()
	if tc.ActiveRelayer != "hyperspace" {
		panic(fmt.Errorf("hyperspace relayer must be specified"))
	}
}

// getConfigOverrides returns configuration overrides that will be applied to the simapp.
func getConfigOverrides() map[string]any {
	consensusOverrides := make(testutil.Toml)
	blockTime := 5
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

// GetGrandpaTestChains returns the configured chains for the grandpa test suite.
func (s *GrandpaTestSuite) GetGrandpaTestChains() (ibc.Chain, ibc.Chain) {
	return s.GetChains(func(options *testsuite.ChainOptions) {
		// configure chain A (polkadot)
		options.ChainASpec.ChainName = composable
		options.ChainASpec.Type = "polkadot"
		options.ChainASpec.ChainID = "rococo-local"
		options.ChainASpec.Name = "composable"
		options.ChainASpec.Images = []ibc.DockerImage{
			// TODO: https://github.com/cosmos/ibc-go/issues/4965
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
		options.ChainASpec.Bech32Prefix = composable
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
		options.ChainBSpec.ChainName = simd // Set chain name so that a suffix with a "dash" is not appended (required for hyperspace)
		options.ChainBSpec.Type = "cosmos"
		options.ChainBSpec.Name = "simd"
		options.ChainBSpec.ChainID = simd
		options.ChainBSpec.Bin = simd
		options.ChainBSpec.Bech32Prefix = "cosmos"

		// TODO: hyperspace relayer assumes a denom of "stake", hard code this here right now.
		// https://github.com/cosmos/ibc-go/issues/4964
		options.ChainBSpec.Denom = "stake"
		options.ChainBSpec.GasPrices = "0.00stake"
		options.ChainBSpec.GasAdjustment = 1
		options.ChainBSpec.TrustingPeriod = "504h"
		options.ChainBSpec.CoinType = "118"

		options.ChainBSpec.ChainConfig.NoHostMount = false
		options.ChainBSpec.ConfigFileOverrides = getConfigOverrides()
		options.ChainBSpec.EncodingConfig = testsuite.SDKEncodingConfig()
	})
}
