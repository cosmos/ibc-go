//go:build !test_e2e

package wasm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v9/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v9/ibc"
	"github.com/strangelove-ventures/interchaintest/v9/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	govtypes "cosmossdk.io/x/gov/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

const (
	haltHeight         = int64(325)
	blocksAfterUpgrade = uint64(10)
)

func TestIBCWasmUpgradeTestSuite(t *testing.T) {
	testCfg := testsuite.LoadConfig()
	if strings.TrimSpace(testCfg.UpgradePlanName) == "" {
		t.Fatalf("%s must be set when running an upgrade test", testsuite.ChainUpgradePlanEnv)
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
	chain := s.GetAllChains()[0]
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
		s.UpgradeChain(ctx, chain.(*cosmos.CosmosChain), userWallet, testCfg.GetUpgradeConfig().PlanName, testCfg.ChainConfigs[0].Tag, testCfg.GetUpgradeConfig().Tag)
	})

	t.Run("query wasm checksums", func(t *testing.T) {
		checksumsResp, err := query.GRPCQuery[wasmtypes.QueryChecksumsResponse](ctx, chain, &wasmtypes.QueryChecksumsRequest{})
		s.Require().NoError(err)
		s.Require().Contains(checksumsResp.Checksums, checksum)
	})
}

// UpgradeChain upgrades a chain to a specific version using the planName provided.
// The software upgrade proposal is broadcast by the provided wallet.
func (s *IBCWasmUpgradeTestSuite) UpgradeChain(ctx context.Context, chain *cosmos.CosmosChain, wallet ibc.Wallet, planName, currentVersion, upgradeVersion string) {
	plan := upgradetypes.Plan{
		Name:   planName,
		Height: haltHeight,
		Info:   fmt.Sprintf("upgrade version test from %s to %s", currentVersion, upgradeVersion),
	}

	upgradeProposal := upgradetypes.SoftwareUpgradeProposal{
		Title:       fmt.Sprintf("upgrade from %s to %s", currentVersion, upgradeVersion),
		Description: "upgrade chain E2E test",
		Plan:        plan,
	}

	s.ExecuteAndPassGovV1Proposal(ctx, &upgradeProposal, chain, wallet)

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

	codeResp, err := query.GRPCQuery[wasmtypes.QueryCodeResponse](ctx, chain, &wasmtypes.QueryCodeRequest{Checksum: computedChecksum})
	s.Require().NoError(err)

	checksumBz := codeResp.Data
	checksum32 := sha256.Sum256(checksumBz)
	actualChecksum := hex.EncodeToString(checksum32[:])
	s.Require().Equal(computedChecksum, actualChecksum, "checksum returned from query did not match the computed checksum")

	return actualChecksum
}

// extractChecksumFromGzippedContent takes a gzipped wasm contract and returns the checksum.
func (s *IBCWasmUpgradeTestSuite) extractChecksumFromGzippedContent(zippedContent []byte) string {
	content, err := wasmtypes.Uncompress(zippedContent, wasmtypes.MaxWasmSize)
	s.Require().NoError(err)

	checksum32 := sha256.Sum256(content)
	return hex.EncodeToString(checksum32[:])
}
