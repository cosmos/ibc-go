package testsuite

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	dockerclient "github.com/docker/docker/client"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"e2e/testconfig"
)

const (
	// ChainARelayerName is the name given to the relayer wallet on ChainA
	ChainARelayerName = "rlyA"
	// ChainBRelayerName is the name given to the relayer wallet on ChainB
	ChainBRelayerName = "rlyB"
)

// E2ETestSuite has methods and functionality which can be shared among all test suites.
type E2ETestSuite struct {
	suite.Suite
	paths          map[string]path
	logger         *zap.Logger
	DockerClient   *dockerclient.Client
	network        string
	startRelayerFn func(relayer ibc.Relayer)
}

// path is a pairing of two chains which will be used in a test.
type path struct {
	chainA, chainB *cosmos.CosmosChain
}

// newPath returns a path built from the given chains.
func newPath(chainA, chainB *cosmos.CosmosChain) path {
	return path{
		chainA: chainA,
		chainB: chainB,
	}
}

// SetupChainsRelayerAndChannel create two chains, a relayer, establishes a connection and creates a channel
// using the given channel options. The relayer returned by this function has not yet started. It should be started
// with E2ETestSuite.StartRelayer if needed.
// This should be called at the start of every test, unless fine grained control is required.
func (s *E2ETestSuite) SetupChainsRelayerAndChannel(ctx context.Context, channelOpts ...func(*ibc.CreateChannelOptions)) ibc.Relayer {
	chainA, chainB := s.GetChains()
	home, err := ioutil.TempDir("", "")
	s.Require().NoError(err)

	r := newCosmosRelayer(s.T(), s.logger, s.DockerClient, s.network)

	pathName := fmt.Sprintf("%s-path", s.T().Name())
	pathName = strings.ReplaceAll(pathName, "/", "-")

	ic := ibctest.NewInterchain().
		AddChain(chainA).
		AddChain(chainB).
		AddRelayer(r, "r").
		AddLink(ibctest.InterchainLink{
			Chain1:  chainA,
			Chain2:  chainB,
			Relayer: r,
			Path:    pathName,
		})

	channelOptions := ibc.DefaultChannelOpts()
	for _, opt := range channelOpts {
		opt(&channelOptions)
	}

	eRep := s.getRelayerExecReporter()
	s.Require().NoError(ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:          s.T().Name(),
		HomeDir:           home,
		Client:            s.DockerClient,
		NetworkID:         s.network,
		CreateChannelOpts: channelOptions,
	}))

	s.startRelayerFn = func(relayer ibc.Relayer) {
		err := relayer.StartRelayer(ctx, eRep, pathName)
		s.Require().NoError(err, fmt.Sprintf("failed to start relayer: %s", err))
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

	return r
}

// GetChains returns two chains that can be used in a test. The pair returned
// is unique to the current test being run. Note: this function does not create containers.
func (s *E2ETestSuite) GetChains(chainOpts ...testconfig.ChainOptionConfiguration) (*cosmos.CosmosChain, *cosmos.CosmosChain) {
	if s.paths == nil {
		s.paths = map[string]path{}
	}

	path, ok := s.paths[s.T().Name()]
	if ok {
		return path.chainA, path.chainB
	}

	chainOptions := testconfig.DefaultChainOptions()
	for _, opt := range chainOpts {
		opt(&chainOptions)
	}

	chainA, chainB := s.createCosmosChains(chainOptions)
	path = newPath(chainA, chainB)
	s.paths[s.T().Name()] = path

	return path.chainA, path.chainB
}

// GetRelayerWallets returns the relayer wallets associated with the chains.
func (s *E2ETestSuite) GetRelayerWallets(relayer ibc.Relayer) (ibc.RelayerWallet, ibc.RelayerWallet, error) {
	chainA, chainB := s.GetChains()
	chainARelayerWallet, ok := relayer.GetWallet(chainA.Config().ChainID)
	if !ok {
		return ibc.RelayerWallet{}, ibc.RelayerWallet{}, fmt.Errorf("unable to find chain A relayer wallet")
	}

	chainBRelayerWallet, ok := relayer.GetWallet(chainB.Config().ChainID)
	if !ok {
		return ibc.RelayerWallet{}, ibc.RelayerWallet{}, fmt.Errorf("unable to find chain B relayer wallet")
	}
	return chainARelayerWallet, chainBRelayerWallet, nil
}

// RecoverRelayerWallets adds the corresponding relayer address to the keychain of the chain.
// This is useful if commands executed on the chains expect the relayer information to present in the keychain.
func (s *E2ETestSuite) RecoverRelayerWallets(ctx context.Context, relayer ibc.Relayer) error {
	chainARelayerWallet, chainBRelayerWallet, err := s.GetRelayerWallets(relayer)
	if err != nil {
		return err
	}

	chainA, chainB := s.GetChains()

	if err := chainA.RecoverKey(ctx, ChainARelayerName, chainARelayerWallet.Mnemonic); err != nil {
		return fmt.Errorf("could not recover relayer wallet on chain A: %s", err)
	}
	if err := chainB.RecoverKey(ctx, ChainBRelayerName, chainBRelayerWallet.Mnemonic); err != nil {
		return fmt.Errorf("could not recover relayer wallet on chain B: %s", err)
	}
	return nil
}

// StartRelayer starts the given relayer.
func (s *E2ETestSuite) StartRelayer(relayer ibc.Relayer) {
	if s.startRelayerFn == nil {
		panic("cannot start relayer before it is created!")
	}

	s.startRelayerFn(relayer)
}

// CreateUserOnChainA creates a user with the given amount of funds on chain A.
func (s *E2ETestSuite) CreateUserOnChainA(ctx context.Context, amount int64) *ibctest.User {
	chainA, _ := s.GetChains()
	return ibctest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), amount, chainA)[0]
}

// CreateUserOnChainB creates a user with the given amount of funds on chain B.
func (s *E2ETestSuite) CreateUserOnChainB(ctx context.Context, amount int64) *ibctest.User {
	_, chainB := s.GetChains()
	return ibctest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), amount, chainB)[0]
}

// GetChainANativeBalance gets the balance of a given user on chain A.
func (s *E2ETestSuite) GetChainANativeBalance(ctx context.Context, user *ibctest.User) (int64, error) {
	chainA, _ := s.GetChains()
	return GetNativeChainBalance(ctx, chainA, user)
}

// GetChainBNativeBalance gets the balance of a given user on chain B.
func (s *E2ETestSuite) GetChainBNativeBalance(ctx context.Context, user *ibctest.User) (int64, error) {
	_, chainB := s.GetChains()
	return GetNativeChainBalance(ctx, chainB, user)
}

// createCosmosChains creates two separate chains in docker containers.
// test and can be retrieved with GetChains.
func (s *E2ETestSuite) createCosmosChains(chainOptions testconfig.ChainOptions) (*cosmos.CosmosChain, *cosmos.CosmosChain) {
	ctx := context.Background()
	client, network := ibctest.DockerSetup(s.T())

	s.logger = zap.NewExample()
	s.DockerClient = client
	s.network = network

	logger := zaptest.NewLogger(s.T())

	// TODO(chatton): allow for controller over number of validators and full nodes.
	chainA := cosmos.NewCosmosChain(s.T().Name(), *chainOptions.ChainAConfig, 1, 0, logger)
	chainB := cosmos.NewCosmosChain(s.T().Name(), *chainOptions.ChainBConfig, 1, 0, logger)

	s.T().Cleanup(func() {
		if !s.T().Failed() {
			for _, c := range []*cosmos.CosmosChain{chainA, chainB} {
				if err := c.Cleanup(ctx); err != nil {
					s.T().Logf("Chain cleanup for %s failed: %v", c.Config().ChainID, err)
				}
			}
		}
	})

	return chainA, chainB
}

// getRelayerExecReporter returns a testreporter.RelayerExecReporter instances
// using the current test's testing.T.
func (s *E2ETestSuite) getRelayerExecReporter() *testreporter.RelayerExecReporter {
	rep := testreporter.NewNopReporter()
	return rep.RelayerExecReporter(s.T())
}

// GetNativeChainBalance returns the balance of a specific user on a chain using the native denom.
func GetNativeChainBalance(ctx context.Context, chain ibc.Chain, user *ibctest.User) (int64, error) {
	bal, err := chain.GetBalance(ctx, user.Bech32Address(chain.Config().Bech32Prefix), chain.Config().Denom)
	if err != nil {
		return -1, err
	}
	return bal, nil
}
