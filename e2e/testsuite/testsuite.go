package testsuite

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/ibc-go/v3/e2e/e2efee"
	"github.com/cosmos/ibc-go/v3/e2e/setup"
	"github.com/cosmos/ibc-go/v3/e2e/testconfig"
	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
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
	chainPairs       map[string]chainPair
	logger           *zap.Logger
	pool             *dockertest.Pool
	network          string
	startRelayerFunc func(relayer ibc.Relayer)
	Rep              *testreporter.Reporter
	Req              *require.Assertions
}

type chainPair struct {
	srcChain, dstChain *cosmos.CosmosChain
}

// GetChains returns a src and dst chain that can be used in a test. The pair returned
// is unique to the current test being run.
func (s *E2ETestSuite) GetChains(chainOpts ...func(*ChainOptions)) (*cosmos.CosmosChain, *cosmos.CosmosChain) {
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

	srcChain, dstChain := s.CreateCosmosChains(chainOptions)
	cp = chainPair{
		srcChain: srcChain,
		dstChain: dstChain,
	}
	s.chainPairs[s.T().Name()] = cp

	return cp.srcChain, cp.dstChain
}

func defaultChainOptions() ChainOptions {
	tc := testconfig.FromEnv()
	srcChainCfg := setup.NewSimappConfig(tc, "simapp-a", "chain-a", "atoma")
	dstChainCfg := setup.NewSimappConfig(tc, "simapp-b", "chain-b", "atomb")
	return ChainOptions{
		SrcChainConfig: &srcChainCfg,
		DstChainConfig: &dstChainCfg,
	}
}

func (s *E2ETestSuite) SetupTest() {
	s.Rep = testreporter.NewNopReporter()
	s.Req = require.New(s.Rep.TestifyT(s.T()))
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
		fmt.Sprintf(`echo "%s" | %s keys add %s --recover --keyring-backend %s --home %s --output json`, mnemonic, chain.Config().Bin, name, keyring.BackendTest, tn.NodeHome()),
	}

	_, _, err := tn.Exec(ctx, cmd, nil)
	return err
}

// CreateCosmosChains creates two separate chains in docker containers.
// test and can be retrieved with GetChains.
func (s *E2ETestSuite) CreateCosmosChains(chainOptions ChainOptions) (*cosmos.CosmosChain, *cosmos.CosmosChain) {
	ctx := context.Background()
	pool, network := ibctest.DockerSetup(s.T())

	s.logger = zap.NewExample()
	s.pool = pool
	s.network = network

	logger := zaptest.NewLogger(s.T())
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

type ChainOptions struct {
	SrcChainConfig *ibc.ChainConfig
	DstChainConfig *ibc.ChainConfig
}

func (s *E2ETestSuite) CreateChainsRelayerAndChannel(ctx context.Context, channelOpts ...func(*ibc.CreateChannelOptions)) (ibc.Relayer, ibc.ChannelOutput) {
	srcChain, dstChain := s.GetChains()
	req := s.Req
	eRep := s.Rep.RelayerExecReporter(s.T())

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

	channelOptions := &ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          "unordered",
		Version:        "ics20-1",
	}

	for _, opt := range channelOpts {
		opt(channelOptions)
	}

	// The client isn't created immediately -- wait for two blocks to ensure the clients are ready.
	req.NoError(test.WaitForBlocks(ctx, 2, srcChain, dstChain))
	req.NoError(r.CreateConnections(ctx, eRep, pathName))
	req.NoError(r.CreateChannel(ctx, eRep, pathName, *channelOptions))

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

// StartRelayer starts the given relayer.
func (s *E2ETestSuite) StartRelayer(relayer ibc.Relayer) {
	if s.startRelayerFunc == nil {
		panic("cannot start relayer before it is created!")
	}
	s.startRelayerFunc(relayer)
}

// CreateUserOnSourceChain creates a user with the given amount of funds on the source chain.
func (s *E2ETestSuite) CreateUserOnSourceChain(ctx context.Context, amount int64) *ibctest.User {
	srcChain, _ := s.GetChains()
	return ibctest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), amount, srcChain)[0]
}

//// CreateUserOnSourceChainWithMnemonic creates a user with the given amount of funds on the source chain from the given mnemonic.
//func (s *E2ETestSuite) CreateUserOnSourceChainWithMnemonic(ctx context.Context, amount int64, mnemonic string) *ibctest.User {
//	srcChain, _ := s.GetChains()
//	return ibctest.GetAndFundTestUserWithMnemonic(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), mnemonic, amount, srcChain)
//}

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

// GetNativeChainBalance returns the balance of a specific user on a chain using the native denom.
func GetNativeChainBalance(ctx context.Context, chain ibc.Chain, user *ibctest.User) (int64, error) {
	bal, err := chain.GetBalance(ctx, user.Bech32Address(chain.Config().Bech32Prefix), chain.Config().Denom)
	if err != nil {
		return -1, err
	}
	return bal, nil
}

func GetCounterPartyChainBalance(ctx context.Context, nativeChain, counterPartyChain ibc.Chain, user *ibctest.User, counterPartyPortId, counterPartyChannelId string) (int64, error) {
	nativeChainDenom := nativeChain.Config().Denom

	nativeDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(counterPartyPortId, counterPartyChannelId, nativeChainDenom))
	counterPartyIBCDenom := nativeDenomTrace.IBCDenom()

	bal, err := counterPartyChain.GetBalance(ctx, user.Bech32Address(counterPartyChain.Config().Bech32Prefix), counterPartyIBCDenom)
	if err != nil {
		return -1, err
	}
	return bal, nil
}

func (s *E2ETestSuite) AssertRelayerWalletsCanBeRecovered(ctx context.Context, relayer ibc.Relayer) func(t *testing.T) {
	return func(t *testing.T) {
		s.Req.NoError(s.RecoverRelayerWallets(ctx, relayer))
	}
}

func (s *E2ETestSuite) AssertChainNativeBalance(ctx context.Context, chain ibc.Chain, user *ibctest.User, expected int64) func(t *testing.T) {
	return func(t *testing.T) {
		actualBalance, err := GetNativeChainBalance(ctx, chain, user)
		s.Req.NoError(err)
		s.Req.Equal(expected, actualBalance)
	}
}

func (s *E2ETestSuite) AssertEmptyPackets(ctx context.Context, chain *cosmos.CosmosChain, portId, channelId string) func(t *testing.T) {
	return func(t *testing.T) {
		packets, err := e2efee.QueryPackets(ctx, chain, portId, channelId)
		s.Req.NoError(err)
		s.Req.Len(packets.IncentivizedPackets, 0)
	}
}
