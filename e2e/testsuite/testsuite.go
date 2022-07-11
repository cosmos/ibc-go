package testsuite

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/broadcast"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/relayer"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/ibc-go/v4/e2e/testconfig"
	feetypes "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
)

const (
	ChainARelayerName = "rlyA"
	ChainBRelayerName = "rlyB"
)

// E2ETestSuite has methods and functionality which can be shared among all test suites.
type E2ETestSuite struct {
	suite.Suite
	clientSets     map[string]GRPCClientSet
	chainPairs     map[string]chainPair
	logger         *zap.Logger
	DockerClient   *dockerclient.Client
	network        string
	startRelayerFn func(relayer ibc.Relayer)
}

type GRPCClientSet struct {
	FeeQueryClient feetypes.QueryClient
}

// chainPair is a pairing of two chains which will be used in a test.
type chainPair struct {
	chainA, chainB *cosmos.CosmosChain
}

// ChainOptions stores chain configurations for the chains that will be
// created for the tests. They can be modified by passing ChainOptionConfiguration
// to E2ETestSuite.GetChains.
type ChainOptions struct {
	ChainAConfig *ibc.ChainConfig
	ChainBConfig *ibc.ChainConfig
}

// ChainOptionConfiguration enables arbitrary configuration of ChainOptions.
type ChainOptionConfiguration func(options *ChainOptions)

// GetChains returns a chain a and b that can be used in a test. The pair returned
// is unique to the current test being run. Note: this function does not create containers.
func (s *E2ETestSuite) GetChains(chainOpts ...ChainOptionConfiguration) (*cosmos.CosmosChain, *cosmos.CosmosChain) {
	if s.chainPairs == nil {
		s.chainPairs = map[string]chainPair{}
	}
	cp, ok := s.chainPairs[s.T().Name()]
	if ok {
		return cp.chainA, cp.chainB
	}

	chainOptions := defaultChainOptions()
	for _, opt := range chainOpts {
		opt(&chainOptions)
	}

	chainA, chainB := s.createCosmosChains(chainOptions)
	cp = chainPair{
		chainA: chainA,
		chainB: chainB,
	}
	s.chainPairs[s.T().Name()] = cp

	return cp.chainA, cp.chainB
}

// CreateChainsRelayerAndChannel create two chains, a relayer, establishes a connection and creates a channel
// using the given channel options. The relayer returned by this function has not yet started. It should be started
// with E2ETestSuite.StartRelayer if needed.
// This should be called at the start of every test, unless fine grained control is required.
func (s *E2ETestSuite) CreateChainsRelayerAndChannel(ctx context.Context, channelOpts ...func(*ibc.CreateChannelOptions)) (ibc.Relayer, ibc.ChannelOutput) {
	chainA, chainB := s.GetChains()
	home, err := ioutil.TempDir("", "")
	s.Require().NoError(err)

	r := newRelayer(s.T(), s.logger, s.DockerClient, s.network)

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

	s.initClientSet(chainA)
	s.initClientSet(chainB)

	chainAChannels, err := r.GetChannels(ctx, eRep, chainA.Config().ChainID)
	s.Require().NoError(err)
	return r, chainAChannels[len(chainAChannels)-1]
}

func (s *E2ETestSuite) GetChainGRCPClientSet(chain ibc.Chain) GRPCClientSet {
	cs, ok := s.clientSets[chain.Config().ChainID]
	s.Require().True(ok, "chain %s does not have a clientset", chain.Config().ChainID)
	return cs
}

func (s *E2ETestSuite) initClientSet(chain *cosmos.CosmosChain) {
	// Create a connection to the gRPC server.
	grpcConn, err := grpc.Dial(
		chain.GetHostGRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)

	if s.clientSets == nil {
		s.clientSets = map[string]GRPCClientSet{}

	}
	s.clientSets[chain.Config().ChainID] = GRPCClientSet{
		FeeQueryClient: feetypes.NewQueryClient(grpcConn),
	}
}

// GetRelayerWallets returns the relayer wallets associated with the test chains.
func (s *E2ETestSuite) GetRelayerWallets(relayer ibc.Relayer) (ibc.RelayerWallet, ibc.RelayerWallet, error) {
	chainA, chainB := s.GetChains()
	chainARelayWallet, ok := relayer.GetWallet(chainA.Config().ChainID)
	if !ok {
		return ibc.RelayerWallet{}, ibc.RelayerWallet{}, fmt.Errorf("unable to find chain-a relayer wallet")
	}
	chainBRelayWallet, ok := relayer.GetWallet(chainB.Config().ChainID)
	if !ok {
		return ibc.RelayerWallet{}, ibc.RelayerWallet{}, fmt.Errorf("unable to find chain-b relayer wallet")
	}
	return chainARelayWallet, chainBRelayWallet, nil
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
		return fmt.Errorf("could not recover relayer wallet on chain-a: %s", err)
	}
	if err := chainB.RecoverKey(ctx, ChainBRelayerName, chainBRelayerWallet.Mnemonic); err != nil {
		return fmt.Errorf("could not recover relayer wallet on chain-b: %s", err)
	}
	return nil
}

// BroadcastMessages broadcasts the provided messages to the given chain and signs them on behalf of the provided user.
func (s *E2ETestSuite) BroadcastMessages(ctx context.Context, chain *cosmos.CosmosChain, user broadcast.User, msgs ...sdk.Msg) (sdk.TxResponse, error) {
	broadcaster := cosmos.NewBroadcaster(s.T(), chain)
	resp, err := ibctest.BroadcastTx(ctx, broadcaster, user, msgs...)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	chainA, chainB := s.GetChains()
	err = test.WaitForBlocks(ctx, 2, chainA, chainB)
	return resp, err
}

// StartRelayer starts the given relayer.
func (s *E2ETestSuite) StartRelayer(relayer ibc.Relayer) {
	if s.startRelayerFn == nil {
		panic("cannot start relayer before it is created!")
	}
	s.startRelayerFn(relayer)
}

// CreateUserOnChainA creates a user with the given amount of funds on chain-a.
func (s *E2ETestSuite) CreateUserOnChainA(ctx context.Context, amount int64) *ibctest.User {
	chainA, _ := s.GetChains()
	return ibctest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), amount, chainA)[0]
}

// CreateUserOnChainB creates a user with the given amount of funds on chain-b.
func (s *E2ETestSuite) CreateUserOnChainB(ctx context.Context, amount int64) *ibctest.User {
	_, chainB := s.GetChains()
	return ibctest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), amount, chainB)[0]
}

// GetChainANativeBalance gets the balance of a given user on chain-a.
func (s *E2ETestSuite) GetChainANativeBalance(ctx context.Context, user *ibctest.User) (int64, error) {
	chainA, _ := s.GetChains()
	return GetNativeChainBalance(ctx, chainA, user)
}

// GetChainBNativeBalance gets the balance of a given user on chain-b.
func (s *E2ETestSuite) GetChainBNativeBalance(ctx context.Context, user *ibctest.User) (int64, error) {
	_, chainB := s.GetChains()
	return GetNativeChainBalance(ctx, chainB, user)
}

// createCosmosChains creates two separate chains in docker containers.
// test and can be retrieved with GetChains.
func (s *E2ETestSuite) createCosmosChains(chainOptions ChainOptions) (*cosmos.CosmosChain, *cosmos.CosmosChain) {
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

func (s *E2ETestSuite) GetChannel(ctx context.Context, chain ibc.Chain, r ibc.Relayer) ibc.ChannelOutput {
	eRep := s.getRelayerExecReporter()
	channels, err := r.GetChannels(ctx, eRep, chain.Config().ChainID)
	s.Require().NoError(err)
	return channels[len(channels)-1]
}

// GetRelayerUsers returns two ibctest.User instances which can be used for the relayer users
// on the two chains.
func (s *E2ETestSuite) GetRelayerUsers(ctx context.Context, chainOpts ...ChainOptionConfiguration) (*ibctest.User, *ibctest.User) {
	chainA, chainB := s.GetChains(chainOpts...)
	chainAAccountBytes, err := chainA.GetAddress(ctx, ChainARelayerName)
	s.Require().NoError(err)

	chainBAccountBytes, err := chainB.GetAddress(ctx, ChainBRelayerName)
	s.Require().NoError(err)

	chainARelayerUser := ibctest.User{
		Address: chainAAccountBytes,
		KeyName: ChainARelayerName,
	}

	chainBRelayerUser := ibctest.User{
		Address: chainBAccountBytes,
		KeyName: ChainBRelayerName,
	}
	return &chainARelayerUser, &chainBRelayerUser
}

func (s *E2ETestSuite) AssertValidTxResponse(resp sdk.TxResponse) {
	respLogsMsg := resp.Logs.String()
	s.Require().NotEqual(int64(0), resp.GasUsed, respLogsMsg)
	s.Require().NotEqual(int64(0), resp.GasWanted, respLogsMsg)
	s.Require().NotEmpty(resp.Events, respLogsMsg)
	s.Require().NotEmpty(resp.Data, respLogsMsg)
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

// defaultChainOptions returns the default configuration for the chains.
// These options can be configured by passing configuration functions to E2ETestSuite.GetChains.
func defaultChainOptions() ChainOptions {
	tc := testconfig.FromEnv()
	chainACfg := newDefaultSimappConfig(tc, "simapp-a", "chain-a", "atoma")
	chainBCfg := newDefaultSimappConfig(tc, "simapp-b", "chain-b", "atomb")
	return ChainOptions{
		ChainAConfig: &chainACfg,
		ChainBConfig: &chainBCfg,
	}
}

// newRelayer returns an instance of the go relayer.
func newRelayer(t *testing.T, logger *zap.Logger, client *dockerclient.Client, network string) ibc.Relayer {
	return ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger, relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "main")).Build(
		t, client, network,
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
