package testsuite

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	dockerclient "github.com/docker/docker/client"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/relayer"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/cosmos/ibc-go/v4/e2e/testconfig"
)

const (
	SourceRelayerName      = "srcRly"
	DestinationRelayerName = "dstRly"
)

// E2ETestSuite has methods and functionality which can be shared among all test suites.
type E2ETestSuite struct {
	suite.Suite
	chainPairs     map[string]chainPair
	logger         *zap.Logger
	Client         *dockerclient.Client
	network        string
	startRelayerFn func(relayer ibc.Relayer)
}

// chainPair is a pairing of two chains which will be used in a test.
type chainPair struct {
	srcChain, dstChain *cosmos.CosmosChain
}

// ChainOptions stores chain configurations for the chains that will be
// created for the tests. They can be modified by passing ChainOptionConfiguration
// to E2ETestSuite.GetChains.
type ChainOptions struct {
	SrcChainConfig *ibc.ChainConfig
	DstChainConfig *ibc.ChainConfig
}

// ChainOptionConfiguration enables arbitrary configuration of ChainOptions.
type ChainOptionConfiguration func(options *ChainOptions)

// GetChains returns a src and dst chain that can be used in a test. The pair returned
// is unique to the current test being run. Note: this function does not create containers.
func (s *E2ETestSuite) GetChains(chainOpts ...ChainOptionConfiguration) (*cosmos.CosmosChain, *cosmos.CosmosChain) {
	if s.chainPairs == nil {
		s.chainPairs = map[string]chainPair{}
	}
	cp, ok := s.chainPairs[s.T().Name()]
	if ok {
		return cp.srcChain, cp.dstChain
	}

	chainOptions := defaultChainOptions()
	for _, opt := range chainOpts {
		opt(&chainOptions)
	}

	srcChain, dstChain := s.createCosmosChains(chainOptions)
	cp = chainPair{
		srcChain: srcChain,
		dstChain: dstChain,
	}
	s.chainPairs[s.T().Name()] = cp

	return cp.srcChain, cp.dstChain
}

// CreateChainsRelayerAndChannel create two chains, a relayer, establishes a connection and creates a channel
// using the given channel options. The relayer returned by this function has not yet started. It should be started
// with E2ETestSuite.StartRelayer if needed.
// This should be called at the start of every test, unless fine grained control is required.
func (s *E2ETestSuite) CreateChainsRelayerAndChannel(ctx context.Context, channelOpts ...func(*ibc.CreateChannelOptions)) ibc.Relayer {
	srcChain, dstChain := s.GetChains()
	home, err := ioutil.TempDir("", "")
	s.Require().NoError(err)

	r := newRelayer(s.T(), s.logger, s.Client, s.network, home)

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

	channelOptions := ibc.DefaultChannelOpts()
	for _, opt := range channelOpts {
		opt(&channelOptions)
	}

	eRep := s.getRelayerExecReporter()
	s.Require().NoError(ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:          s.T().Name(),
		HomeDir:           home,
		Client:            s.Client,
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
// This is useful if commands executed on the chains expect the relayer information to present in the keychain.
func (s *E2ETestSuite) RecoverRelayerWallets(ctx context.Context, relayer ibc.Relayer) error {
	srcRelayerWallet, dstRelayerWallet, err := s.GetRelayerWallets(relayer)
	if err != nil {
		return err
	}

	srcChain, dstChain := s.GetChains()

	if err := srcChain.RecoverKey(ctx, SourceRelayerName, srcRelayerWallet.Mnemonic); err != nil {
		return fmt.Errorf("could not recover relayer wallet on source chain: %s", err)
	}
	if err := dstChain.RecoverKey(ctx, DestinationRelayerName, dstRelayerWallet.Mnemonic); err != nil {
		return fmt.Errorf("could not recover relayer wallet on destination chain: %s", err)
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

// CreateUserOnSourceChain creates a user with the given amount of funds on the source chain.
func (s *E2ETestSuite) CreateUserOnSourceChain(ctx context.Context, amount int64) *ibctest.User {
	srcChain, _ := s.GetChains()
	return ibctest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), amount, srcChain)[0]
}

// CreateUserOnDestinationChain creates a user with the given amount of funds on the destination chain.
func (s *E2ETestSuite) CreateUserOnDestinationChain(ctx context.Context, amount int64) *ibctest.User {
	_, dstChain := s.GetChains()
	return ibctest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), amount, dstChain)[0]
}

// GetSourceChainNativeBalance gets the balance of a given user on the source chain.
func (s *E2ETestSuite) GetSourceChainNativeBalance(ctx context.Context, user *ibctest.User) (int64, error) {
	srcChain, _ := s.GetChains()
	return GetNativeChainBalance(ctx, srcChain, user)
}

// GetDestinationChainNativeBalance gets the balance of a given user on the destination chain.
func (s *E2ETestSuite) GetDestinationChainNativeBalance(ctx context.Context, user *ibctest.User) (int64, error) {
	_, dstChain := s.GetChains()
	return GetNativeChainBalance(ctx, dstChain, user)
}

// createCosmosChains creates two separate chains in docker containers.
// test and can be retrieved with GetChains.
func (s *E2ETestSuite) createCosmosChains(chainOptions ChainOptions) (*cosmos.CosmosChain, *cosmos.CosmosChain) {
	ctx := context.Background()
	client, network := ibctest.DockerSetup(s.T())

	s.logger = zap.NewExample()
	s.Client = client
	s.network = network

	logger := zaptest.NewLogger(s.T())

	// TODO(chatton): allow for controller over number of validators and full nodes.
	srcChain := cosmos.NewCosmosChain(s.T().Name(), *chainOptions.SrcChainConfig, 1, 0, logger)
	dstChain := cosmos.NewCosmosChain(s.T().Name(), *chainOptions.DstChainConfig, 1, 0, logger)

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

// defaultChainOptions returns the default configuration for the source and destination chains.
// These options can be configured by passing configuration functions to E2ETestSuite.GetChains.
func defaultChainOptions() ChainOptions {
	tc := testconfig.FromEnv()
	srcChainCfg := newDefaultSimappConfig(tc, "simapp-a", "chain-a", "atoma")
	dstChainCfg := newDefaultSimappConfig(tc, "simapp-b", "chain-b", "atomb")
	return ChainOptions{
		SrcChainConfig: &srcChainCfg,
		DstChainConfig: &dstChainCfg,
	}
}

// newRelayer returns an instance of the go relayer.
func newRelayer(t *testing.T, logger *zap.Logger, client *dockerclient.Client, network string, home string) ibc.Relayer {
	return ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger, relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "main")).Build(
		t, client, network, home,
	)
}

// newDefaultSimappConfig creates an ibc configuration for simd.
func newDefaultSimappConfig(tc testconfig.TestConfig, name, chainId, denom string) ibc.ChainConfig {
	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    name,
		ChainID: chainId,
		Images: []ibc.DockerImage{
			{
				Repository: tc.SimdImage,
				Version:    tc.SimdTag,
			},
		},
		Bin:            "simd",
		Bech32Prefix:   "cosmos",
		Denom:          denom,
		GasPrices:      fmt.Sprintf("0.01%s", denom),
		GasAdjustment:  1.3,
		TrustingPeriod: "508h",
		NoHostMount:    false,
	}
}
