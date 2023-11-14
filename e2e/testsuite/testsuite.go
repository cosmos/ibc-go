package testsuite

import (
	"context"
	"errors"
	"fmt"
	"strings"

	dockerclient "github.com/docker/docker/client"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/cosmos/ibc-go/e2e/relayer"
	"github.com/cosmos/ibc-go/e2e/testsuite/diagnostics"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
)

const (
	// ChainARelayerName is the name given to the relayer wallet on ChainA
	ChainARelayerName = "rlyA"
	// ChainBRelayerName is the name given to the relayer wallet on ChainB
	ChainBRelayerName = "rlyB"
	// DefaultGasValue is the default gas value used to configure tx.Factory
	DefaultGasValue = 500_000_0000
)

// E2ETestSuite has methods and functionality which can be shared among all test suites.
type E2ETestSuite struct {
	testifysuite.Suite

	// proposalIDs keeps track of the active proposal ID for each chain.
	proposalIDs    map[string]uint64
	grpcClients    map[string]GRPCClients
	paths          map[string]pathPair
	relayers       relayer.Map
	logger         *zap.Logger
	DockerClient   *dockerclient.Client
	network        string
	startRelayerFn func(relayer ibc.Relayer)

	// pathNameIndex is the latest index to be used for generating paths
	pathNameIndex int64
}

// pathPair is a pairing of two chains which will be used in a test.
type pathPair struct {
	chainA, chainB ibc.Chain
}

// newPath returns a path built from the given chains.
func newPath(chainA, chainB ibc.Chain) pathPair {
	return pathPair{
		chainA: chainA,
		chainB: chainB,
	}
}

// GetRelayerUsers returns two ibc.Wallet instances which can be used for the relayer users
// on the two chains.
func (s *E2ETestSuite) GetRelayerUsers(ctx context.Context, chainOpts ...ChainOptionConfiguration) (ibc.Wallet, ibc.Wallet) {
	chainA, chainB := s.GetChains(chainOpts...)
	chainAAccountBytes, err := chainA.GetAddress(ctx, ChainARelayerName)
	s.Require().NoError(err)

	chainBAccountBytes, err := chainB.GetAddress(ctx, ChainBRelayerName)
	s.Require().NoError(err)

	chainARelayerUser := cosmos.NewWallet(ChainARelayerName, chainAAccountBytes, "", chainA.Config())
	chainBRelayerUser := cosmos.NewWallet(ChainBRelayerName, chainBAccountBytes, "", chainB.Config())

	if s.relayers == nil {
		s.relayers = make(relayer.Map)
	}
	s.relayers.AddRelayer(s.T().Name(), chainARelayerUser)
	s.relayers.AddRelayer(s.T().Name(), chainBRelayerUser)

	return chainARelayerUser, chainBRelayerUser
}

// SetupChainsRelayerAndChannel create two chains, a relayer, establishes a connection and creates a channel
// using the given channel options. The relayer returned by this function has not yet started. It should be started
// with E2ETestSuite.StartRelayer if needed.
// This should be called at the start of every test, unless fine grained control is required.
func (s *E2ETestSuite) SetupChainsRelayerAndChannel(ctx context.Context, channelOpts func(*ibc.CreateChannelOptions), chainSpecOpts ...ChainOptionConfiguration) (ibc.Relayer, ibc.ChannelOutput) {
	chainA, chainB := s.GetChains(chainSpecOpts...)
	r := s.ConfigureRelayer(ctx, chainA, chainB, channelOpts)
	s.InitGRPCClients(chainA)
	s.InitGRPCClients(chainB)
	chainAChannels, err := r.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
	s.Require().NoError(err)
	return r, chainAChannels[len(chainAChannels)-1]
}

func (s *E2ETestSuite) ConfigureRelayer(ctx context.Context, chainA, chainB ibc.Chain, channelOpts func(*ibc.CreateChannelOptions), buildOptions ...func(options *interchaintest.InterchainBuildOptions)) ibc.Relayer {
	r := relayer.New(s.T(), *LoadConfig().GetActiveRelayerConfig(), s.logger, s.DockerClient, s.network)

	pathName := s.generatePathName()

	channelOptions := ibc.DefaultChannelOpts()
	if channelOpts != nil {
		channelOpts(&channelOptions)
	}

	ic := interchaintest.NewInterchain().
		AddChain(chainA).
		AddChain(chainB).
		AddRelayer(r, "r").
		AddLink(interchaintest.InterchainLink{
			Chain1:            chainA,
			Chain2:            chainB,
			Relayer:           r,
			Path:              pathName,
			CreateChannelOpts: channelOptions,
		})

	buildOpts := interchaintest.InterchainBuildOptions{
		TestName:  s.T().Name(),
		Client:    s.DockerClient,
		NetworkID: s.network,
	}

	for _, opt := range buildOptions {
		opt(&buildOpts)
	}

	eRep := s.GetRelayerExecReporter()
	s.Require().NoError(ic.Build(ctx, eRep, buildOpts))

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
		s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")
	}

	return r
}

// SetupSingleChain creates and returns a single CosmosChain for usage in e2e tests.
// This is useful for testing single chain functionality when performing coordinated upgrades as well as testing localhost ibc client functionality.
// TODO: Actually setup a single chain. Seeing panic: runtime error: index out of range [0] with length 0 when using a single chain.
// issue: https://github.com/strangelove-ventures/interchaintest/issues/401
func (s *E2ETestSuite) SetupSingleChain(ctx context.Context) ibc.Chain {
	chainA, chainB := s.GetChains()

	ic := interchaintest.NewInterchain().AddChain(chainA).AddChain(chainB)

	eRep := s.GetRelayerExecReporter()
	s.Require().NoError(ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         s.T().Name(),
		Client:           s.DockerClient,
		NetworkID:        s.network,
		SkipPathCreation: true,
	}))

	s.InitGRPCClients(chainA)
	s.InitGRPCClients(chainB)

	return chainA
}

// generatePathName generates the path name using the test suites name
func (s *E2ETestSuite) generatePathName() string {
	path := s.GetPathName(s.pathNameIndex)
	s.pathNameIndex++
	return path
}

// GetPathName returns the name of a path at a specific index. This can be used in tests
// when the path name is required.
func (s *E2ETestSuite) GetPathName(idx int64) string {
	pathName := fmt.Sprintf("%s-path-%d", s.T().Name(), idx)
	return strings.ReplaceAll(pathName, "/", "-")
}

// generatePath generates the path name using the test suites name
func (s *E2ETestSuite) generatePath(ctx context.Context, ibcrelayer ibc.Relayer) string {
	chainA, chainB := s.GetChains()
	chainAID := chainA.Config().ChainID
	chainBID := chainB.Config().ChainID

	pathName := s.generatePathName()

	err := ibcrelayer.GeneratePath(ctx, s.GetRelayerExecReporter(), chainAID, chainBID, pathName)
	s.Require().NoError(err)

	return pathName
}

// SetupClients creates clients on chainA and chainB using the provided create client options
func (s *E2ETestSuite) SetupClients(ctx context.Context, ibcrelayer ibc.Relayer, opts ibc.CreateClientOptions) {
	pathName := s.generatePath(ctx, ibcrelayer)
	err := ibcrelayer.CreateClients(ctx, s.GetRelayerExecReporter(), pathName, opts)
	s.Require().NoError(err)
}

// UpdateClients updates clients on chainA and chainB
func (s *E2ETestSuite) UpdateClients(ctx context.Context, ibcrelayer ibc.Relayer, pathName string) {
	err := ibcrelayer.UpdateClients(ctx, s.GetRelayerExecReporter(), pathName)
	s.Require().NoError(err)
}

// GetChains returns two chains that can be used in a test. The pair returned
// is unique to the current test being run. Note: this function does not create containers.
func (s *E2ETestSuite) GetChains(chainOpts ...ChainOptionConfiguration) (ibc.Chain, ibc.Chain) {
	if s.paths == nil {
		s.paths = map[string]pathPair{}
	}

	path, ok := s.paths[s.T().Name()]
	if ok {
		return path.chainA, path.chainB
	}

	chainOptions := DefaultChainOptions()
	for _, opt := range chainOpts {
		opt(&chainOptions)
	}

	chainA, chainB := s.createChains(chainOptions)
	path = newPath(chainA, chainB)
	s.paths[s.T().Name()] = path

	if s.proposalIDs == nil {
		s.proposalIDs = map[string]uint64{}
	}

	s.proposalIDs[chainA.Config().ChainID] = 1
	s.proposalIDs[chainB.Config().ChainID] = 1

	return path.chainA, path.chainB
}

// GetRelayerWallets returns the ibcrelayer wallets associated with the chains.
func (s *E2ETestSuite) GetRelayerWallets(ibcrelayer ibc.Relayer) (ibc.Wallet, ibc.Wallet, error) {
	chainA, chainB := s.GetChains()
	chainARelayerWallet, ok := ibcrelayer.GetWallet(chainA.Config().ChainID)
	if !ok {
		return nil, nil, fmt.Errorf("unable to find chain A relayer wallet")
	}

	chainBRelayerWallet, ok := ibcrelayer.GetWallet(chainB.Config().ChainID)
	if !ok {
		return nil, nil, fmt.Errorf("unable to find chain B relayer wallet")
	}
	return chainARelayerWallet, chainBRelayerWallet, nil
}

// RecoverRelayerWallets adds the corresponding ibcrelayer address to the keychain of the chain.
// This is useful if commands executed on the chains expect the relayer information to present in the keychain.
func (s *E2ETestSuite) RecoverRelayerWallets(ctx context.Context, ibcrelayer ibc.Relayer) error {
	chainARelayerWallet, chainBRelayerWallet, err := s.GetRelayerWallets(ibcrelayer)
	if err != nil {
		return err
	}

	chainA, chainB := s.GetChains()

	if err := chainA.RecoverKey(ctx, ChainARelayerName, chainARelayerWallet.Mnemonic()); err != nil {
		return fmt.Errorf("could not recover relayer wallet on chain A: %s", err)
	}
	if err := chainB.RecoverKey(ctx, ChainBRelayerName, chainBRelayerWallet.Mnemonic()); err != nil {
		return fmt.Errorf("could not recover relayer wallet on chain B: %s", err)
	}
	return nil
}

// StartRelayer starts the given ibcrelayer.
func (s *E2ETestSuite) StartRelayer(ibcrelayer ibc.Relayer) {
	if s.startRelayerFn == nil {
		panic(errors.New("cannot start relayer before it is created"))
	}

	s.startRelayerFn(ibcrelayer)
}

// StopRelayer stops the given ibcrelayer.
func (s *E2ETestSuite) StopRelayer(ctx context.Context, ibcrelayer ibc.Relayer) {
	err := ibcrelayer.StopRelayer(ctx, s.GetRelayerExecReporter())
	s.Require().NoError(err)
}

// RestartRelayer restarts the given relayer.
func (s *E2ETestSuite) RestartRelayer(ctx context.Context, ibcrelayer ibc.Relayer) {
	s.StopRelayer(ctx, ibcrelayer)
	s.StartRelayer(ibcrelayer)
}

// CreateUserOnChainA creates a user with the given amount of funds on chain A.
func (s *E2ETestSuite) CreateUserOnChainA(ctx context.Context, amount int64) ibc.Wallet {
	chainA, _ := s.GetChains()
	return interchaintest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), amount, chainA)[0]
}

// CreateUserOnChainB creates a user with the given amount of funds on chain B.
func (s *E2ETestSuite) CreateUserOnChainB(ctx context.Context, amount int64) ibc.Wallet {
	_, chainB := s.GetChains()
	return interchaintest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), amount, chainB)[0]
}

// GetChainANativeBalance gets the balance of a given user on chain A.
func (s *E2ETestSuite) GetChainANativeBalance(ctx context.Context, user ibc.Wallet) (int64, error) {
	chainA, _ := s.GetChains()

	balance, err := s.QueryBalance(ctx, chainA, user.FormattedAddress(), chainA.Config().Denom)
	if err != nil {
		return 0, err
	}
	return balance.Int64(), nil
}

// GetChainBNativeBalance gets the balance of a given user on chain B.
func (s *E2ETestSuite) GetChainBNativeBalance(ctx context.Context, user ibc.Wallet) (int64, error) {
	_, chainB := s.GetChains()
	balance, err := s.QueryBalance(ctx, chainB, user.FormattedAddress(), chainB.Config().Denom)
	if err != nil {
		return -1, err
	}
	return balance.Int64(), nil
}

// GetChainGRCPClients gets the GRPC clients associated with the given chain.
func (s *E2ETestSuite) GetChainGRCPClients(chain ibc.Chain) GRPCClients {
	cs, ok := s.grpcClients[chain.Config().ChainID]
	s.Require().True(ok, "chain %s does not have GRPC clients", chain.Config().ChainID)
	return cs
}

// AssertPacketRelayed asserts that the packet commitment does not exist on the sending chain.
// The packet commitment will be deleted upon a packet acknowledgement or timeout.
func (s *E2ETestSuite) AssertPacketRelayed(ctx context.Context, chain ibc.Chain, portID, channelID string, sequence uint64) {
	commitment, _ := s.QueryPacketCommitment(ctx, chain, portID, channelID, sequence)
	s.Require().Empty(commitment)
}

// AssertHumanReadableDenom asserts that a human readable denom is present for a given chain.
func (s *E2ETestSuite) AssertHumanReadableDenom(ctx context.Context, chain ibc.Chain, counterpartyNativeDenom string, counterpartyChannel ibc.ChannelOutput) {
	chainIBCDenom := GetIBCToken(counterpartyNativeDenom, counterpartyChannel.Counterparty.PortID, counterpartyChannel.Counterparty.ChannelID)

	denomMetadata, err := s.QueryDenomMetadata(ctx, chain, chainIBCDenom.IBCDenom())
	s.Require().NoError(err)

	s.Require().Equal(chainIBCDenom.IBCDenom(), denomMetadata.Base, "denom metadata base does not match expected %s: got %s", chainIBCDenom.IBCDenom(), denomMetadata.Base)
	expectedName := fmt.Sprintf("%s/%s/%s IBC token", counterpartyChannel.Counterparty.PortID, counterpartyChannel.Counterparty.ChannelID, counterpartyNativeDenom)
	s.Require().Equal(expectedName, denomMetadata.Name, "denom metadata name does not match expected %s: got %s", expectedName, denomMetadata.Name)
	expectedDisplay := fmt.Sprintf("%s/%s/%s", counterpartyChannel.Counterparty.PortID, counterpartyChannel.Counterparty.ChannelID, counterpartyNativeDenom)
	s.Require().Equal(expectedDisplay, denomMetadata.Display, "denom metadata display does not match expected %s: got %s", expectedDisplay, denomMetadata.Display)
	s.Require().Equal(strings.ToUpper(counterpartyNativeDenom), denomMetadata.Symbol, "denom metadata symbol does not match expected %s: got %s", strings.ToUpper(counterpartyNativeDenom), denomMetadata.Symbol)
}

// createChains creates two separate chains in docker containers.
// test and can be retrieved with GetChains.
func (s *E2ETestSuite) createChains(chainOptions ChainOptions) (ibc.Chain, ibc.Chain) {
	client, network := interchaintest.DockerSetup(s.T())
	t := s.T()

	s.logger = zap.NewExample()
	s.DockerClient = client
	s.network = network

	cf := interchaintest.NewBuiltinChainFactory(s.logger, []*interchaintest.ChainSpec{chainOptions.ChainASpec, chainOptions.ChainBSpec})

	// this is intentionally called after the interchaintest.DockerSetup function. The above function registers a
	// cleanup task which deletes all containers. By registering a cleanup function afterwards, it is executed first
	// this allows us to process the logs before the containers are removed.
	t.Cleanup(func() {
		debugModeEnabled := LoadConfig().DebugConfig.DumpLogs
		chains := []string{chainOptions.ChainASpec.ChainConfig.Name, chainOptions.ChainBSpec.ChainConfig.Name}
		diagnostics.Collect(t, s.DockerClient, debugModeEnabled, chains...)
	})

	chains, err := cf.Chains(t.Name())
	s.Require().NoError(err)

	return chains[0], chains[1]
}

// GetRelayerExecReporter returns a testreporter.RelayerExecReporter instances
// using the current test's testing.T.
func (s *E2ETestSuite) GetRelayerExecReporter() *testreporter.RelayerExecReporter {
	rep := testreporter.NewNopReporter()
	return rep.RelayerExecReporter(s.T())
}

// TransferChannelOptions configures both of the chains to have non-incentivized transfer channels.
func (*E2ETestSuite) TransferChannelOptions() func(options *ibc.CreateChannelOptions) {
	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = transfertypes.Version
		opts.SourcePortName = transfertypes.PortID
		opts.DestPortName = transfertypes.PortID
	}
}

// GetTimeoutHeight returns a timeout height of 1000 blocks above the current block height.
// This function should be used when the timeout is never expected to be reached
func (s *E2ETestSuite) GetTimeoutHeight(ctx context.Context, chain ibc.Chain) clienttypes.Height {
	height, err := chain.Height(ctx)
	s.Require().NoError(err)
	return clienttypes.NewHeight(clienttypes.ParseChainID(chain.Config().ChainID), height+1000)
}

// GetIBCToken returns the denomination of the full token denom sent to the receiving channel
func GetIBCToken(fullTokenDenom string, portID, channelID string) transfertypes.DenomTrace {
	return transfertypes.ParseDenomTrace(fmt.Sprintf("%s/%s/%s", portID, channelID, fullTokenDenom))
}

// getValidatorsAndFullNodes returns the number of validators and full nodes respectively that should be used for
// the test. If the test is running in CI, more nodes are used, when running locally a single node is used by default to
// use less resources and allow the tests to run faster.
// both the number of validators and full nodes can be overwritten in a config file.
func getValidatorsAndFullNodes(chainIdx int) (int, int) {
	if IsCI() {
		return 4, 1
	}
	tc := LoadConfig()
	return tc.GetChainNumValidators(chainIdx), tc.GetChainNumFullNodes(chainIdx)
}
