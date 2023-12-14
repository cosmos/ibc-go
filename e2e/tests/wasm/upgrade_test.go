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

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

const (
	haltHeight         = uint64(325)
	blocksAfterUpgrade = uint64(10)
)

func TestIBCWasmUpgradeTestSuite(t *testing.T) {
	testCfg := testsuite.LoadConfig()
	if testCfg.UpgradeConfig.Tag == "" || testCfg.UpgradeConfig.PlanName == "" {
		t.Fatalf("%s and %s must be set when running an upgrade test", testsuite.ChainUpgradeTagEnv, testsuite.ChainUpgradePlanEnv)
	}

	// wasm tests require a longer voting period to account for the time it takes to upload a contract.
	testvalues.VotingPeriod = time.Minute * 5

	testifysuite.Run(t, new(IBCWasmUpgradeTestSuite))
}

type IBCWasmUpgradeTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *IBCWasmUpgradeTestSuite) TestIBCWasmChainUpgrade() {
	t := s.T()

	ctx := context.Background()
	chain := s.SetupSingleChain(ctx)
	checksum := ""

	userWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	s.Require().NoError(testutil.WaitForBlocks(ctx, 1, chain), "failed to wait for blocks")

	t.Run("create and exec store code proposal", func(t *testing.T) {
		file, err := os.Open("contracts/ics10_grandpa_cw.wasm.gz")
		s.Require().NoError(err)

		checksum = s.ExecStoreCodeProposal(ctx, chain.(*cosmos.CosmosChain), userWallet, file)
		s.Require().NotEmpty(checksum, "checksum must not be empty")
	})

	t.Run("upgrade chain", func(t *testing.T) {
		testCfg := testsuite.LoadConfig()
		s.UpgradeChain(ctx, chain.(*cosmos.CosmosChain), userWallet, testCfg.UpgradeConfig.PlanName, testCfg.ChainConfigs[0].Tag, testCfg.UpgradeConfig.Tag)
	})

	t.Run("query wasm checksums", func(t *testing.T) {
		checksums, err := s.QueryWasmChecksums(ctx, chain)
		s.Require().NoError(err)
		s.Require().Contains(checksums, checksum)
	})
}

// UpgradeChain upgrades a chain to a specific version using the planName provided.
// The software upgrade proposal is broadcast by the provided wallet.
func (s *IBCWasmUpgradeTestSuite) UpgradeChain(ctx context.Context, chain *cosmos.CosmosChain, wallet ibc.Wallet, planName, currentVersion, upgradeVersion string) {
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

	err = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height)+1, chain)
	s.Require().Error(err, "chain did not halt at halt height")

	var allNodes []testutil.ChainHeighter
	for _, node := range chain.Nodes() {
		allNodes = append(allNodes, node)
	}

	err = testutil.WaitForInSync(ctx, chain, allNodes...)
	s.Require().NoError(err, "error waiting for node(s) to sync")

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

	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	s.Require().NoError(err, "chain did not produce blocks after upgrade")

	height, err = chain.Height(ctx)
	s.Require().NoError(err, "error fetching height after upgrade")

	s.Require().Greater(height, haltHeight, "height did not increment after upgrade")
}

func (s *IBCWasmUpgradeTestSuite) ExecStoreCodeProposal(ctx context.Context, chain *cosmos.CosmosChain, wallet ibc.Wallet, proposalContentReader io.Reader) string {
	zippedContent, err := io.ReadAll(proposalContentReader)
	s.Require().NoError(err)

	computedChecksum := s.extractChecksumFromGzippedContent(zippedContent)

	msgStoreCode := wasmtypes.MsgStoreCode{
		Signer:       authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		WasmByteCode: zippedContent,
	}

	s.ExecuteAndPassGovV1Proposal(ctx, &msgStoreCode, chain, wallet)

	checksumBz, err := s.QueryWasmCode(ctx, chain, computedChecksum)
	s.Require().NoError(err)

	checksum32 := sha256.Sum256(checksumBz)
	actualChecksum := hex.EncodeToString(checksum32[:])
	s.Require().Equal(computedChecksum, actualChecksum, "checksum returned from query did not match the computed checksum")

	return actualChecksum
}

// extractChecksumFromGzippedContent takes a gzipped wasm contract and returns the checksum.
func (s *IBCWasmUpgradeTestSuite) extractChecksumFromGzippedContent(zippedContent []byte) string {
	content, err := wasmtypes.Uncompress(zippedContent, wasmtypes.MaxWasmByteSize())
	s.Require().NoError(err)

	checksum32 := sha256.Sum256(content)
	return hex.EncodeToString(checksum32[:])
}
