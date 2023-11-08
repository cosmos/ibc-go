//go:build !test_e2e

package wasm

import (
	"context"
	// "crypto/sha256"
	"encoding/hex"
	// "fmt"
	// "io"
	"os"
	// "testing"
	// "time"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/chain/polkadot"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	// testifysuite "github.com/stretchr/testify/suite"

	// "cosmossdk.io/math"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	// transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

func (s *GrandpaTestSuite) TestMsgMigrateContract_FailedMigration_GrandpaContract() {
	ctx := context.Background()

	chainA, chainB := s.GetChains(func(options *testsuite.ChainOptions) {
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

	polkadotChain := chainA.(*polkadot.PolkadotChain)
	cosmosChain := chainB.(*cosmos.CosmosChain)

	// we explicitly skip path creation as the contract needs to be uploaded before we can create clients.
	r := s.ConfigureRelayer(ctx, polkadotChain, cosmosChain, nil, func(options *interchaintest.InterchainBuildOptions) {
		options.SkipPathCreation = true
	})

	s.InitGRPCClients(cosmosChain)

	var err error
	cosmosWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

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
	// fundAmount := int64(12_333_000_000_000)
	// polkadotUser, cosmosUser := s.fundUsers(ctx, fundAmount, polkadotChain, cosmosChain)

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

	// Setup complete, now we can test migration
	migrate_file, err := os.Open("../data/migrate_error.wasm.gz")
	s.Require().NoError(err)

	// First Store the code
	newCodeHashHex := s.PushNewWasmClientProposal(ctx, cosmosChain, cosmosWallet, migrate_file)
	s.Require().NotEmpty(newCodeHashHex, "codehash was empty but should not have been")

	newCodeHashBz, err := hex.DecodeString(newCodeHashHex)

	// Attempt to migrate the contract
	message := wasmtypes.MsgMigrateContract{
		Signer:       authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		ClientId: "08-wasm-0",
		CodeHash: newCodeHashBz,
		Msg: []byte("{}"),
	}

	err = s.ExecuteGovV1Proposal(ctx, &message, cosmosChain, cosmosWallet)
	s.Require().Error(err)
}
