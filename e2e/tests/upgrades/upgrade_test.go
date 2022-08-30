package upgrades

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testconfig"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
)

const (
	haltHeight         = uint64(50)
	blocksAfterUpgrade = uint64(10)
)

func TestUpgradeTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeTestSuite))
}

type UpgradeTestSuite struct {
	testsuite.E2ETestSuite
}

// UpgradeChain upgrades a chain to a specific version using the planName provided.
// The software upgrade proposal is broadcast by the provided wallet.
func (s *UpgradeTestSuite) UpgradeChain(ctx context.Context, chain *cosmos.CosmosChain, wallet *ibc.Wallet, planName, upgradeVersion string) {
	plan := upgradetypes.Plan{
		Name:   planName,
		Height: int64(haltHeight),
		Info:   fmt.Sprintf("upgrade version test from %s to %s", chain.Nodes()[0].Image.Version, upgradeVersion),
	}
	upgradeProposal := upgradetypes.NewSoftwareUpgradeProposal("some title", "some description", plan)
	s.ExecuteGovProposal(ctx, chain, wallet, upgradeProposal)

	height, err := chain.Height(ctx)
	s.Require().NoError(err, "error fetching height before upgrade")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err = test.WaitForBlocks(timeoutCtx, int(haltHeight-height)+1, chain)
	s.Require().Error(err, "chain did not halt at halt height")

	err = chain.StopAllNodes(ctx)
	s.Require().NoError(err, "error stopping node(s)")

	chain.UpgradeVersion(ctx, s.DockerClient, upgradeVersion)

	err = chain.StartAllNodes(ctx)
	s.Require().NoError(err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	err = test.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	s.Require().NoError(err, "chain did not produce blocks after upgrade")

	height, err = chain.Height(ctx)
	s.Require().NoError(err, "error fetching height after upgrade")

	s.Require().GreaterOrEqual(height, haltHeight+blocksAfterUpgrade, "height did not increment enough after upgrade")
}

func (s *UpgradeTestSuite) TestV4ToV5ChainUpgrade() {
	// TODO: temporarily hard code the version upgrades.
	oldVersion := "v4.0.0"
	targetVersion := "pr-2144" // v5 version with upgrade handler, replace with v5.0.0-rc3 when it is cut.
	s.Require().NoError(os.Setenv(testconfig.ChainATagEnv, oldVersion))
	s.Require().NoError(os.Setenv(testconfig.ChainBTagEnv, oldVersion))

	ctx := context.Background()

	s.SetupChainsRelayerAndChannel(ctx)
	chainA, _ := s.GetChains()

	chainAUser := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	s.UpgradeChain(ctx, chainA, chainAUser, "normal upgrade", targetVersion)
}
