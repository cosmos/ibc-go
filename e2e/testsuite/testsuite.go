package testsuite

import (
	"context"
	"fmt"
	"github.com/cosmos/ibc-go/v3/e2e/setup"
	"github.com/ory/dockertest/v3"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

// E2ETestSuite has methods and functionality which can be shared among all test suites.
type E2ETestSuite struct {
	suite.Suite
	logger  *zap.Logger
	pool    *dockertest.Pool
	network string
}

func (s *E2ETestSuite) CreateCosmosChains() (*cosmos.CosmosChain, *cosmos.CosmosChain) {
	ctx := context.Background()
	pool, network := ibctest.DockerSetup(s.T())

	s.logger = zap.NewExample()
	s.pool = pool
	s.network = network

	chainAConfig := setup.NewSimappConfig("simapp-a", "chain-a", "atoma")
	chainBConfig := setup.NewSimappConfig("simapp-b", "chain-b", "atomb")
	logger := zaptest.NewLogger(s.T())
	srcChain := cosmos.NewCosmosChain(s.T().Name(), chainAConfig, 1, 0, logger)
	dstChain := cosmos.NewCosmosChain(s.T().Name(), chainBConfig, 1, 0, logger)

	s.T().Cleanup(func() {
		if !s.T().Failed() {
			for _, c := range []*cosmos.CosmosChain{srcChain, dstChain} {
				if err := c.Cleanup(ctx); err != nil {
					s.T().Logf("Chain cleanup for %s failed: %v", c.Config().ChainID, err)
				}
			}
		}
	})
	return srcChain, dstChain
}

func (s *E2ETestSuite) CreateRelayerAndChannel(ctx context.Context, srcChain, dstChain ibc.Chain, req *require.Assertions, eRep *testreporter.RelayerExecReporter) (ibc.Relayer, ibc.ChannelOutput, func(t *testing.T)) {

	home, err := ioutil.TempDir("", "")
	req.NoError(err)

	//home := s.T().TempDir() // Must be before chain cleanup to avoid test error during cleanup.
	r := setup.NewRelayer(s.T(), s.logger, s.pool, s.network, home)

	pathName := fmt.Sprintf("%s-path", s.T().Name())
	pathName = strings.ReplaceAll(pathName, "/", "-")

	ic := ibctest.NewInterchain().
		AddChain(srcChain).
		AddChain(dstChain).
		AddRelayer(r, "r").
		AddLink(ibctest.InterchainLink{
			Chain1:  srcChain,
			Chain2:  dstChain,
			Relayer: r,
			Path:    pathName,
		})

	req.NoError(ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:         s.T().Name(),
		HomeDir:          home,
		Pool:             s.pool,
		NetworkID:        s.network,
		SkipPathCreation: true,
	}))

	req.NoError(r.GeneratePath(ctx, eRep, srcChain.Config().ChainID, dstChain.Config().ChainID, pathName))
	req.NoError(r.CreateClients(ctx, eRep, pathName))

	// The client isn't created immediately -- wait for two blocks to ensure the clients are ready.
	req.NoError(test.WaitForBlocks(ctx, 2, srcChain, dstChain))
	req.NoError(r.CreateConnections(ctx, eRep, pathName))
	req.NoError(r.CreateChannel(ctx, eRep, pathName, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          "unordered",
		Version:        "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}",
	}))

	// Now validate that the channels correctly report as created.
	// GetChannels takes around two seconds with rly,
	// so getting the channels concurrently is a measurable speedup.
	eg, egCtx := errgroup.WithContext(ctx)
	var srcChainChannels []ibc.ChannelOutput
	eg.Go(func() error {
		var err error
		srcChainChannels, err = r.GetChannels(egCtx, eRep, srcChain.Config().ChainID)
		return err
	})
	eg.Go(func() error {
		var err error
		_, err = r.GetChannels(egCtx, eRep, dstChain.Config().ChainID)
		return err
	})
	req.NoError(eg.Wait(), "failure retrieving channels")

	return r, srcChainChannels[len(srcChainChannels)-1], func(t *testing.T) {
		err := r.StartRelayer(ctx, eRep, pathName)
		req.NoError(err, fmt.Sprintf("failed to start relayer: %s", err))
		t.Cleanup(func() {
			if !t.Failed() {
				if err := r.StopRelayer(ctx, eRep); err != nil {
					t.Logf("error stopping relayer: %v", err)
				}
			}
		})
		// wait for relayer to start.
		time.Sleep(time.Second * 10)
	}
}
