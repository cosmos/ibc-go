package upgrades

import (
	"context"
	"os"
	"testing"
	"time"

	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testconfig"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
)

const (
	haltHeight         = uint64(100)
	blocksAfterUpgrade = uint64(10)
)

func TestUpgradeTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeTestSuite))
}

type UpgradeTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *UpgradeTestSuite) TestChainUpgrade() {
	// TODO: temporarily hard code the version upgrades.
	oldVersion := "v5.0.0-rc0"
	s.Require().NoError(os.Setenv(testconfig.ChainATagEnv, oldVersion))
	s.Require().NoError(os.Setenv(testconfig.ChainBTagEnv, oldVersion))
	upgradeVersion := "v5.0.0-rc1"

	t := s.T()
	ctx := context.Background()

	s.SetupChainsRelayerAndChannel(ctx)
	chainA, _ := s.GetChains()

	chainAUser := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	plan := upgradetypes.Plan{
		Name:   "some-plan",
		Height: int64(haltHeight),
		Info:   "some info",
	}
	upgradeProposal := upgradetypes.NewSoftwareUpgradeProposal("some title", "some description", plan)
	s.ExecuteGovProposal(ctx, chainA, chainAUser, upgradeProposal)

	height, err := chainA.Height(ctx)
	require.NoError(t, err, "error fetching height before upgrade")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	err = test.WaitForBlocks(timeoutCtx, int(haltHeight-height)+1, chainA)
	require.Error(t, err, "chain did not halt at halt height")

	err = chainA.StopAllNodes(ctx)
	s.Require().NoError(err, "error stopping node(s)")

	chainA.UpgradeVersion(ctx, s.DockerClient, upgradeVersion)

	err = chainA.StartAllNodes(ctx)
	s.Require().NoError(err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Minute*2)
	defer timeoutCtxCancel()

	time.Sleep(100 * time.Hour)

	err = test.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chainA)
	s.Require().NoError(err, "chain did not produce blocks after upgrade")

	height, err = chainA.Height(ctx)
	s.Require().NoError(err, "error fetching height after upgrade")

	s.Require().GreaterOrEqual(height, haltHeight+blocksAfterUpgrade, "height did not increment enough after upgrade")
}
