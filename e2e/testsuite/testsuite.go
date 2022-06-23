package testsuite

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/ibc-go/v3/e2e/dockerutil"
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
	"time"
)

// E2ETestSuite has methods and functionality which can be shared among all test suites.
type E2ETestSuite struct {
	suite.Suite
	chainPairs       map[string]chainPair
	logger           *zap.Logger
	pool             *dockertest.Pool
	network          string
	startRelayerFunc func(relayer ibc.Relayer)
}

type chainPair struct {
	srcChain, dstChain *cosmos.CosmosChain
}

// GetChains returns a src and dst chain that can be used in a test. The pair returned
// is unique to the current test being run.
func (s *E2ETestSuite) GetChains() (*cosmos.CosmosChain, *cosmos.CosmosChain) {
	chainPair, ok := s.chainPairs[s.T().Name()]
	if !ok {
		panic(fmt.Sprintf("no chain pair found for test %s", s.T().Name()))
	}
	return chainPair.srcChain, chainPair.dstChain
}

// GetRelayerWallets returns the relayer wallets associated with the source and destination chains.
func (s *E2ETestSuite) GetRelayerWallets(relayer ibc.Relayer) (ibc.RelayerWallet, ibc.RelayerWallet, error) {
	srcChain, dstChain := s.GetChains()
	srcRelayWallet, ok := relayer.GetWallet(srcChain.Config().ChainID)
	if !ok {
		return ibc.RelayerWallet{}, ibc.RelayerWallet{}, fmt.Errorf("unable to find source chain relayer wallet")
	}
	dstRelayWallet, ok := relayer.GetWallet(dstChain.Config().ChainID)
	if !ok {
		return ibc.RelayerWallet{}, ibc.RelayerWallet{}, fmt.Errorf("unable to find destination chain relayer wallet")
	}
	return srcRelayWallet, dstRelayWallet, nil
}

// RecoverRelayerWallets adds the corresponding relayer address to the keychain of the chain.
func (s *E2ETestSuite) RecoverRelayerWallets(ctx context.Context, relayer ibc.Relayer) error {
	srcRelayerWallet, dstRelayerWallet, err := s.GetRelayerWallets(relayer)
	if err != nil {
		return err
	}

	srcChain, dstChain := s.GetChains()

	const srcRelayerName = "srcRly"
	const dstRelayerName = "dstRly"

	if err := recoverKeyring(ctx, srcChain, srcRelayerName, srcRelayerWallet.Mnemonic); err != nil {
		return fmt.Errorf("could not recover relayer wallet on source chain: %s", err)
	}

	if err := recoverKeyring(ctx, dstChain, dstRelayerName, dstRelayerWallet.Mnemonic); err != nil {
		return fmt.Errorf("could not recover relayer wallet on destination chain: %s", err)
	}

	return nil
}

func recoverKeyring(ctx context.Context, chain *cosmos.CosmosChain, name, mnemonic string) error {
	tn := chain.ChainNodes[0]

	cmd := []string{
		"bash",
		"-c",
		fmt.Sprintf(`echo "%s" | %s keys add %s --recover --keyring-backend %s --home %s`, mnemonic, chain.Config().Bin, name, keyring.BackendTest, tn.NodeHome()),
	}

	exitCode, stdout, stderr, err := tn.NodeJob(ctx, cmd)
	if err != nil {
		return dockerutil.HandleNodeJobError(exitCode, stdout, stderr, err)
	}
	return nil
}

// createCosmosChains creates two separate chains in docker containers.
// test and can be retrieved with GetChains.
func (s *E2ETestSuite) createCosmosChains() (*cosmos.CosmosChain, *cosmos.CosmosChain) {
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

func (s *E2ETestSuite) SetupSuite() {
	s.chainPairs = map[string]chainPair{}
}

// SetupTest creates two cosmos.CosmosChain instances and maps them to the test name so they
// are accessible within the test.
// NOTE: if this method is implemented in other test suites, they will need to call E2ETestSuite.SetupTest manually.
func (s *E2ETestSuite) SetupTest() {
	srcChain, dstChain := s.createCosmosChains()
	s.chainPairs[s.T().Name()] = chainPair{
		srcChain: srcChain,
		dstChain: dstChain,
	}
}

func (s *E2ETestSuite) CreateRelayerAndChannel(ctx context.Context, req *require.Assertions, eRep *testreporter.RelayerExecReporter, channelOpts ...func(*ibc.CreateChannelOptions)) (ibc.Relayer, ibc.ChannelOutput) {
	srcChain, dstChain := s.GetChains()

	home, err := ioutil.TempDir("", "")
	req.NoError(err)

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

	defaultChannelsOpts := &ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          "unordered",
		Version:        "ics20-1",
	}

	for _, opt := range channelOpts {
		opt(defaultChannelsOpts)
	}

	// The client isn't created immediately -- wait for two blocks to ensure the clients are ready.
	req.NoError(test.WaitForBlocks(ctx, 2, srcChain, dstChain))
	req.NoError(r.CreateConnections(ctx, eRep, pathName))
	req.NoError(r.CreateChannel(ctx, eRep, pathName, *defaultChannelsOpts))

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

	s.startRelayerFunc = func(relayer ibc.Relayer) {
		err := relayer.StartRelayer(ctx, eRep, pathName)
		req.NoError(err, fmt.Sprintf("failed to start relayer: %s", err))
		s.T().Cleanup(func() {
			if !s.T().Failed() {
				if err := relayer.StopRelayer(ctx, eRep); err != nil {
					s.T().Logf("error stopping relayer: %v", err)
				}
			}
		})
		// wait for relayer to start.
		time.Sleep(time.Second * 10)
	}

	return r, srcChainChannels[len(srcChainChannels)-1]
}

func (s *E2ETestSuite) StartRelayer(relayer ibc.Relayer) {
	if s.startRelayerFunc == nil {
		panic("cannot start relayer before it is creatd!")
	}
	s.startRelayerFunc(relayer)
}

func (s *E2ETestSuite) GetSourceChainBalance(ctx context.Context, user *ibctest.User) (int64, error) {
	srcChain, _ := s.GetChains()
	return getChainBalance(ctx, srcChain, user)
}

func (s *E2ETestSuite) GetDestinationChainBalance(ctx context.Context, user *ibctest.User) (int64, error) {
	_, dstChain := s.GetChains()
	return getChainBalance(ctx, dstChain, user)
}

// getChainBalance returns the balance of a specific user on a chain using the native denom.
func getChainBalance(ctx context.Context, chain ibc.Chain, user *ibctest.User) (int64, error) {
	bal, err := chain.GetBalance(ctx, user.Bech32Address(chain.Config().Bech32Prefix), chain.Config().Denom)
	if err != nil {
		return -1, err
	}
	return bal, nil
}
