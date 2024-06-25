package testsuite

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	dockerclient "github.com/docker/docker/client"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/e2e/internal/directories"
	"github.com/cosmos/ibc-go/e2e/relayer"
	"github.com/cosmos/ibc-go/e2e/testsuite/diagnostics"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	feetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
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
	proposalIDs map[string]uint64

	// chains is a list of chains that are created for the test suite.
	// each test suite has a single slice of chains that are used for all individual test
	// cases.
	chains         []ibc.Chain
	relayers       relayer.Map
	logger         *zap.Logger
	DockerClient   *dockerclient.Client
	network        string
	startRelayerFn func(relayer ibc.Relayer)

	// pathNameIndex is the latest index to be used for generating chains
	pathNameIndex int64

	// TODO: refactor this to use a relayer per test
	// relayer is a single relayer which only works when running tests one per host.
	// this needs to be refactored to use a different relayer per test.
	relayer ibc.Relayer

	// testSuiteName is the name of the test suite, used to store chains under the test suite name.
	testSuiteName string
	testPaths     map[string][]string
	channels      map[string]map[ibc.Chain][]ibc.ChannelOutput
}

// initState populates variables that are used across the test suite.
// note: this should be called only from SetupSuite.
func (s *E2ETestSuite) initState() {
	s.initDockerClient()
	s.proposalIDs = map[string]uint64{}
	s.testPaths = make(map[string][]string)
	s.channels = make(map[string]map[ibc.Chain][]ibc.ChannelOutput)

	// testSuiteName gets populated in the context of SetupSuite and stored as s.T().Name()
	// will return the name of the suite and test when called from SetupTest or within the body of tests.
	// the chains will be stored under the test suite name, so we need to store this for future use.
	s.testSuiteName = s.T().Name()
}

// initDockerClient creates a docker client and populates the network to be used for the test.
func (s *E2ETestSuite) initDockerClient() {
	client, network := interchaintest.DockerSetup(s.T())
	s.logger = zap.NewExample()
	s.DockerClient = client
	s.network = network
}

// SetupSuite will by default create chains with no additional options. If additional options are required,
// the test suite must define the SetupSuite function and provide the required options.
func (s *E2ETestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), nil)
}

// configureGenesisDebugExport sets, if needed, env variables to enable exporting of Genesis debug files.
func (s *E2ETestSuite) configureGenesisDebugExport() {
	tc := LoadConfig()
	t := s.T()
	cfg := tc.DebugConfig.GenesisDebug
	if !cfg.DumpGenesisDebugInfo {
		return
	}

	// Set the export path.
	exportPath := cfg.ExportFilePath

	// If no path is provided, use the default (e2e/diagnostics/genesis.json).
	if exportPath == "" {
		e2eDir, err := directories.E2E(t)
		s.Require().NoError(err, "can't get e2edir")
		exportPath = path.Join(e2eDir, directories.DefaultGenesisExportPath)
	}

	if !path.IsAbs(exportPath) {
		wd, err := os.Getwd()
		s.Require().NoError(err, "can't get working directory")
		exportPath = path.Join(wd, exportPath)
	}

	// This env variables are set by the interchain test code:
	// https://github.com/strangelove-ventures/interchaintest/blob/7aa0fd6487f76238ab44231fdaebc34627bc5990/chain/cosmos/cosmos_chain.go#L1007-L1008
	t.Setenv("EXPORT_GENESIS_FILE_PATH", exportPath)

	chainName := tc.GetGenesisChainName()
	chainIdx, err := tc.GetChainIndex(chainName)
	s.Require().NoError(err)

	// Interchaintest adds a suffix (https://github.com/strangelove-ventures/interchaintest/blob/a3f4c7bcccf1925ffa6dc793a298f15497919a38/chainspec.go#L125)
	// to the chain name, so we need to do the same.
	genesisChainName := fmt.Sprintf("%s-%d", chainName, chainIdx+1)
	t.Setenv("EXPORT_GENESIS_CHAIN", genesisChainName)
}

// SetupChains creates the chains for the test suite, and also a relayer that is wired up to establish
// connections and channels between the chains.
func (s *E2ETestSuite) SetupChains(ctx context.Context, channelOptionsModifier ChainOptionModifier, chainSpecOpts ...ChainOptionConfiguration) {
	s.T().Logf("Setting up chains: %s", s.T().Name())
	s.initState()
	s.configureGenesisDebugExport()

	chainOptions := DefaultChainOptions()
	for _, opt := range chainSpecOpts {
		opt(&chainOptions)
	}

	s.chains = s.createChains(chainOptions)

	// TODO: we need to create a relayer for each test that will run
	// having a single relayer for all tests will cause issues when running tests in parallel.
	s.relayer = relayer.New(s.T(), *LoadConfig().GetActiveRelayerConfig(), s.logger, s.DockerClient, s.network)
	ic := s.newInterchain(ctx, s.relayer, s.chains, channelOptionsModifier)

	buildOpts := interchaintest.InterchainBuildOptions{
		TestName:  s.T().Name(),
		Client:    s.DockerClient,
		NetworkID: s.network,
		// we skip path creation because we are just creating the chains and not connections/channels
		SkipPathCreation: true,
	}

	s.Require().NoError(ic.Build(ctx, s.GetRelayerExecReporter(), buildOpts))
}

// SetupTest will by default use the default channel options to create a path between the chains.
// if non default channel options are required, the test suite must override the `SetupTest` function.
func (s *E2ETestSuite) SetupTest() {
	s.SetupPaths(ibc.DefaultClientOpts(), defaultChannelOpts(s.GetAllChains()))
}

// SetupPaths creates paths between the chains using the provided client and channel options.
// The paths are created such that ChainA is connected to ChainB, ChainB is connected to ChainC etc.
func (s *E2ETestSuite) SetupPaths(clientOpts ibc.CreateClientOptions, channelOpts ibc.CreateChannelOptions) {
	s.T().Logf("Setting up path for: %s", s.T().Name())

	ctx := context.TODO()
	allChains := s.GetAllChains()
	for i := 0; i < len(allChains)-1; i++ {
		chainA, chainB := allChains[i], allChains[i+1]
		_, _ = s.CreatePath(ctx, chainA, chainB, clientOpts, channelOpts)
	}
}

func (s *E2ETestSuite) CreatePath(
	ctx context.Context,
	chainA ibc.Chain,
	chainB ibc.Chain,
	clientOpts ibc.CreateClientOptions,
	channelOpts ibc.CreateChannelOptions,
) (chainAChannel ibc.ChannelOutput, chainBChannel ibc.ChannelOutput) {
	r := s.relayer

	pathName := s.generatePathName()
	s.T().Logf("establishing path between %s and %s on path %s", chainA.Config().ChainID, chainB.Config().ChainID, pathName)

	err := r.GeneratePath(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID, chainB.Config().ChainID, pathName)
	s.Require().NoError(err)

	// Create new clients
	err = r.CreateClients(ctx, s.GetRelayerExecReporter(), pathName, clientOpts)
	s.Require().NoError(err)
	err = test.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)

	err = r.CreateConnections(ctx, s.GetRelayerExecReporter(), pathName)
	s.Require().NoError(err)
	err = test.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)

	err = r.CreateChannel(ctx, s.GetRelayerExecReporter(), pathName, channelOpts)
	s.Require().NoError(err)
	err = test.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)

	channelsA, err := r.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
	s.Require().NoError(err)

	channelsB, err := r.GetChannels(ctx, s.GetRelayerExecReporter(), chainB.Config().ChainID)
	s.Require().NoError(err)

	if s.channels[s.T().Name()] == nil {
		s.channels[s.T().Name()] = make(map[ibc.Chain][]ibc.ChannelOutput)
	}

	// keep track of channels associated with a given chain for access within the tests.
	s.channels[s.T().Name()][chainA] = channelsA
	s.channels[s.T().Name()][chainB] = channelsB

	s.testPaths[s.T().Name()] = append(s.testPaths[s.T().Name()], pathName)

	return channelsA[len(channelsA)-1], channelsB[len(channelsB)-1]
}

// GetChainAChannel returns the ibc.ChannelOutput for the current test.
// this defaults to the first entry in the list, and will be what is needed in the case of
// a single channel test.
func (s *E2ETestSuite) GetChainAChannel() ibc.ChannelOutput {
	chainA := s.GetAllChains()[0]
	return s.GetChannels(chainA)[0]
}

// GetChannels returns all channels for the current test.
func (s *E2ETestSuite) GetChannels(chain ibc.Chain) []ibc.ChannelOutput {
	channels, ok := s.channels[s.T().Name()][chain]
	s.Require().True(ok, "channel not found for test %s", s.T().Name())
	return channels
}

// GetRelayer returns the relayer to be used in the specific test.
// TODO: for now a single instance is still used, preventing parallel test runs.
func (s *E2ETestSuite) GetRelayer() ibc.Relayer {
	return s.relayer
}

// GetRelayerUsers returns two ibc.Wallet instances which can be used for the relayer users
// on the two chains.
func (s *E2ETestSuite) GetRelayerUsers(ctx context.Context, chainOpts ...ChainOptionConfiguration) (ibc.Wallet, ibc.Wallet) {
	chains := s.GetAllChains(chainOpts...)
	chainA, chainB := chains[0], chains[1]
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

// ChainOptionModifier is a function which accepts 2 chains as inputs, and returns a channel creation modifier function
// in order to conditionally modify the channel options based on the chains being used.
type ChainOptionModifier func(chainA, chainB ibc.Chain) func(options *ibc.CreateChannelOptions)

// newInterchain constructs a new interchain instance that creates channels between the chains.
func (s *E2ETestSuite) newInterchain(ctx context.Context, r ibc.Relayer, chains []ibc.Chain, modificationProvider ChainOptionModifier) *interchaintest.Interchain {
	ic := interchaintest.NewInterchain()
	for _, chain := range chains {
		ic.AddChain(chain)
	}
	ic.AddRelayer(r, "r")

	// iterate through all chains, and create links such that there is a channel between
	// - chainA and chainB
	// - chainB and chainC
	// - chainC and chainD etc
	for i := 0; i < len(chains)-1; i++ {
		pathName := s.generatePathName()
		channelOpts := defaultChannelOpts(chains)
		chain1, chain2 := chains[i], chains[i+1]

		if modificationProvider != nil {
			// make a modification to the channel options based on the chains which are being used.
			modificationFn := modificationProvider(chain1, chain2)
			modificationFn(&channelOpts)
		}

		ic.AddLink(interchaintest.InterchainLink{
			Chain1:            chains[i],
			Chain2:            chains[i+1],
			Relayer:           r,
			Path:              pathName,
			CreateChannelOpts: channelOpts,
		})
	}

	s.startRelayerFn = func(relayer ibc.Relayer) {
		// depending on the test, the path names will be different.
		// whenever a relayer is started, it should use the paths associated with the test the relayer is running in.
		pathNames, ok := s.testPaths[s.T().Name()]
		s.Require().True(ok, "path names not found for test %s", s.T().Name())

		err := relayer.StartRelayer(ctx, s.GetRelayerExecReporter(), pathNames...)
		s.Require().NoError(err, fmt.Sprintf("failed to start relayer: %s", err))

		var chainHeighters []test.ChainHeighter
		for _, c := range chains {
			chainHeighters = append(chainHeighters, c)
		}

		// wait for every chain to produce some blocks before using the relayer.
		s.Require().NoError(test.WaitForBlocks(ctx, 10, chainHeighters...), "failed to wait for blocks")
	}

	return ic
}

// generatePathName generates the path name using the test suites name
func (s *E2ETestSuite) generatePathName() string {
	pathName := GetPathName(s.pathNameIndex)
	s.pathNameIndex++
	return pathName
}

func (s *E2ETestSuite) GetPaths() []string {
	paths, ok := s.testPaths[s.T().Name()]
	s.Require().True(ok, "paths not found for test %s", s.T().Name())
	return paths
}

// GetPathName returns the name of a path at a specific index. This can be used in tests
// when the path name is required.
func GetPathName(idx int64) string {
	pathName := fmt.Sprintf("path-%d", idx)
	return strings.ReplaceAll(pathName, "/", "-")
}

// generatePath generates the path name using the test suites name. The indices provided specify which chains should be
// used. E.g. to generate a path between chain A and B, you would use 0 and 1, to specify between A and C, you would
// use 0 and 2 etc.
func (s *E2ETestSuite) generatePath(ctx context.Context, ibcrelayer ibc.Relayer, chainAIdx, chainBIdx int) string {
	chains := s.GetAllChains()
	chainA, chainB := chains[chainAIdx], chains[chainBIdx]
	chainAID := chainA.Config().ChainID
	chainBID := chainB.Config().ChainID

	pathName := s.generatePathName()

	err := ibcrelayer.GeneratePath(ctx, s.GetRelayerExecReporter(), chainAID, chainBID, pathName)
	s.Require().NoError(err)

	return pathName
}

// SetupClients creates clients on chainA and chainB using the provided create client options
func (s *E2ETestSuite) SetupClients(ctx context.Context, ibcrelayer ibc.Relayer, opts ibc.CreateClientOptions) {
	pathName := s.generatePath(ctx, ibcrelayer, 0, 1)
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
	chains := s.GetAllChains(chainOpts...)
	return chains[0], chains[1]
}

// GetAllChains returns all chains that can be used in a test. The chains returned
// are unique to the current test being run. Note: this function does not create containers.
func (s *E2ETestSuite) GetAllChains(chainOpts ...ChainOptionConfiguration) []ibc.Chain {
	// chains are stored on a per test suite level
	chains := s.chains
	s.Require().NotEmpty(chains, "chains not found for test %s", s.testSuiteName)
	return chains
}

// GetRelayerWallets returns the ibcrelayer wallets associated with the chains.
func (s *E2ETestSuite) GetRelayerWallets(ibcrelayer ibc.Relayer) (ibc.Wallet, ibc.Wallet, error) {
	chains := s.GetAllChains()
	chainA, chainB := chains[0], chains[1]
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

	chains := s.GetAllChains()
	chainA, chainB := chains[0], chains[1]

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
	return s.createWalletOnChainIndex(ctx, amount, 0)
}

// CreateUserOnChainB creates a user with the given amount of funds on chain B.
func (s *E2ETestSuite) CreateUserOnChainB(ctx context.Context, amount int64) ibc.Wallet {
	return s.createWalletOnChainIndex(ctx, amount, 1)
}

// CreateUserOnChainC creates a user with the given amount of funds on chain C.
func (s *E2ETestSuite) CreateUserOnChainC(ctx context.Context, amount int64) ibc.Wallet {
	return s.createWalletOnChainIndex(ctx, amount, 2)
}

// createWalletOnChainIndex creates a wallet with the given amount of funds on the chain of the given index.
func (s *E2ETestSuite) createWalletOnChainIndex(ctx context.Context, amount, chainIndex int64) ibc.Wallet {
	chain := s.GetAllChains()[chainIndex]
	return interchaintest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), sdkmath.NewInt(amount), chain)[0]
}

// GetChainANativeBalance gets the balance of a given user on chain A.
func (s *E2ETestSuite) GetChainANativeBalance(ctx context.Context, user ibc.Wallet) (int64, error) {
	chainA := s.GetAllChains()[0]

	balanceResp, err := query.GRPCQuery[banktypes.QueryBalanceResponse](ctx, chainA, &banktypes.QueryBalanceRequest{
		Address: user.FormattedAddress(),
		Denom:   chainA.Config().Denom,
	})
	if err != nil {
		return 0, err
	}

	return balanceResp.Balance.Amount.Int64(), nil
}

// GetChainBNativeBalance gets the balance of a given user on chain B.
func (s *E2ETestSuite) GetChainBNativeBalance(ctx context.Context, user ibc.Wallet) (int64, error) {
	chainB := s.GetAllChains()[1]

	balanceResp, err := query.GRPCQuery[banktypes.QueryBalanceResponse](ctx, chainB, &banktypes.QueryBalanceRequest{
		Address: user.FormattedAddress(),
		Denom:   chainB.Config().Denom,
	})
	if err != nil {
		return 0, err
	}

	return balanceResp.Balance.Amount.Int64(), nil
}

// AssertPacketRelayed asserts that the packet commitment does not exist on the sending chain.
// The packet commitment will be deleted upon a packet acknowledgement or timeout.
func (s *E2ETestSuite) AssertPacketRelayed(ctx context.Context, chain ibc.Chain, portID, channelID string, sequence uint64) {
	_, err := query.GRPCQuery[channeltypes.QueryPacketCommitmentResponse](ctx, chain, &channeltypes.QueryPacketCommitmentRequest{
		PortId:    portID,
		ChannelId: channelID,
		Sequence:  sequence,
	})
	s.Require().ErrorContains(err, "packet commitment hash not found")
}

// AssertHumanReadableDenom asserts that a human readable denom is present for a given chain.
func (s *E2ETestSuite) AssertHumanReadableDenom(ctx context.Context, chain ibc.Chain, counterpartyNativeDenom string, counterpartyChannel ibc.ChannelOutput) {
	chainIBCDenom := GetIBCToken(counterpartyNativeDenom, counterpartyChannel.Counterparty.PortID, counterpartyChannel.Counterparty.ChannelID)

	denomMetadataResp, err := query.GRPCQuery[banktypes.QueryDenomMetadataResponse](ctx, chain, &banktypes.QueryDenomMetadataRequest{
		Denom: chainIBCDenom.IBCDenom(),
	})
	s.Require().NoError(err)

	denomMetadata := denomMetadataResp.Metadata

	s.Require().Equal(chainIBCDenom.IBCDenom(), denomMetadata.Base, "denom metadata base does not match expected %s: got %s", chainIBCDenom.IBCDenom(), denomMetadata.Base)
	expectedName := fmt.Sprintf("%s/%s/%s IBC token", counterpartyChannel.Counterparty.PortID, counterpartyChannel.Counterparty.ChannelID, counterpartyNativeDenom)
	s.Require().Equal(expectedName, denomMetadata.Name, "denom metadata name does not match expected %s: got %s", expectedName, denomMetadata.Name)
	expectedDisplay := fmt.Sprintf("%s/%s/%s", counterpartyChannel.Counterparty.PortID, counterpartyChannel.Counterparty.ChannelID, counterpartyNativeDenom)
	s.Require().Equal(expectedDisplay, denomMetadata.Display, "denom metadata display does not match expected %s: got %s", expectedDisplay, denomMetadata.Display)
	s.Require().Equal(strings.ToUpper(counterpartyNativeDenom), denomMetadata.Symbol, "denom metadata symbol does not match expected %s: got %s", strings.ToUpper(counterpartyNativeDenom), denomMetadata.Symbol)
}

// createChains creates two separate chains in docker containers.
// test and can be retrieved with GetChains.
func (s *E2ETestSuite) createChains(chainOptions ChainOptions) []ibc.Chain {
	t := s.T()
	cf := interchaintest.NewBuiltinChainFactory(s.logger, chainOptions.ChainSpecs)

	// this is intentionally called after the interchaintest.DockerSetup function. The above function registers a
	// cleanup task which deletes all containers. By registering a cleanup function afterwards, it is executed first
	// this allows us to process the logs before the containers are removed.
	t.Cleanup(func() {
		dumpLogs := LoadConfig().DebugConfig.DumpLogs
		var chainNames []string
		for _, chain := range chainOptions.ChainSpecs {
			chainNames = append(chainNames, chain.Name)
		}
		diagnostics.Collect(t, s.DockerClient, dumpLogs, s.testSuiteName, chainNames...)
	})

	chains, err := cf.Chains(t.Name())
	s.Require().NoError(err)

	// initialise proposal ids for all chains.
	for _, chain := range chains {
		s.proposalIDs[chain.Config().ChainID] = 1
	}

	return chains
}

// GetRelayerExecReporter returns a testreporter.RelayerExecReporter instances
// using the current test's testing.T.
func (s *E2ETestSuite) GetRelayerExecReporter() *testreporter.RelayerExecReporter {
	rep := testreporter.NewNopReporter()
	return rep.RelayerExecReporter(s.T())
}

// TransferChannelOptions configures both of the chains to have non-incentivized transfer channels.
func (s *E2ETestSuite) TransferChannelOptions() ibc.CreateChannelOptions {
	opts := ibc.DefaultChannelOpts()
	opts.Version = determineDefaultTransferVersion(s.GetAllChains())
	return opts
}

// FeeTransferChannelOptions configures both of the chains to have fee middleware enabled.
func (s *E2ETestSuite) FeeTransferChannelOptions() ibc.CreateChannelOptions {
	versionMetadata := feetypes.Metadata{
		FeeVersion: feetypes.Version,
		AppVersion: determineDefaultTransferVersion(s.GetAllChains()),
	}
	versionBytes, err := feetypes.ModuleCdc.MarshalJSON(&versionMetadata)
	s.Require().NoError(err)

	opts := ibc.DefaultChannelOpts()
	opts.Version = string(versionBytes)
	return opts
}

// GetTimeoutHeight returns a timeout height of 1000 blocks above the current block height.
// This function should be used when the timeout is never expected to be reached
func (s *E2ETestSuite) GetTimeoutHeight(ctx context.Context, chain ibc.Chain) clienttypes.Height {
	height, err := chain.Height(ctx)
	s.Require().NoError(err)
	return clienttypes.NewHeight(clienttypes.ParseChainID(chain.Config().ChainID), uint64(height)+1000)
}

// CreateUpgradeFields creates upgrade fields for channel with fee middleware
func (s *E2ETestSuite) CreateUpgradeFields(channel channeltypes.Channel) channeltypes.UpgradeFields {
	versionMetadata := feetypes.Metadata{
		FeeVersion: feetypes.Version,
		AppVersion: channel.Version,
	}
	versionBytes, err := feetypes.ModuleCdc.MarshalJSON(&versionMetadata)
	s.Require().NoError(err)

	return channeltypes.NewUpgradeFields(channel.Ordering, channel.ConnectionHops, string(versionBytes))
}

// SetUpgradeTimeoutParam creates and submits a governance proposal to execute the message to update 04-channel params with a timeout of 1s
func (s *E2ETestSuite) SetUpgradeTimeoutParam(ctx context.Context, chain ibc.Chain, wallet ibc.Wallet) {
	const timeoutDelta = 1000000000 // use 1 second as relative timeout to force upgrade timeout on the counterparty
	govModuleAddress, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chain)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	upgradeTimeout := channeltypes.NewTimeout(channeltypes.DefaultTimeout.Height, timeoutDelta)
	msg := channeltypes.NewMsgUpdateChannelParams(govModuleAddress.String(), channeltypes.NewParams(upgradeTimeout))
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, wallet)
}

// InitiateChannelUpgrade creates and submits a governance proposal to execute the message to initiate a channel upgrade
func (s *E2ETestSuite) InitiateChannelUpgrade(ctx context.Context, chain ibc.Chain, wallet ibc.Wallet, portID, channelID string, upgradeFields channeltypes.UpgradeFields) {
	govModuleAddress, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chain)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	msg := channeltypes.NewMsgChannelUpgradeInit(portID, channelID, upgradeFields, govModuleAddress.String())
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, wallet)
}

// GetIBCToken returns the denomination of the full token denom sent to the receiving channel
func GetIBCToken(fullTokenDenom string, portID, channelID string) transfertypes.Denom {
	return transfertypes.ExtractDenomFromPath(fmt.Sprintf("%s/%s/%s", portID, channelID, fullTokenDenom))
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

// GetMsgTransfer returns a MsgTransfer that is constructed based on the channel version
func GetMsgTransfer(portID, channelID, version string, tokens sdk.Coins, sender, receiver string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, memo string, forwarding transfertypes.Forwarding) *transfertypes.MsgTransfer {
	if len(tokens) == 0 {
		panic(errors.New("tokens cannot be empty"))
	}

	var msg *transfertypes.MsgTransfer
	switch version {
	case transfertypes.V1:
		msg = &transfertypes.MsgTransfer{
			SourcePort:       portID,
			SourceChannel:    channelID,
			Token:            tokens[0],
			Sender:           sender,
			Receiver:         receiver,
			TimeoutHeight:    timeoutHeight,
			TimeoutTimestamp: timeoutTimestamp,
			Memo:             memo,
			Tokens:           sdk.NewCoins(),
		}
	case transfertypes.V2:
		msg = transfertypes.NewMsgTransfer(portID, channelID, tokens, sender, receiver, timeoutHeight, timeoutTimestamp, memo, forwarding)
	default:
		panic(fmt.Errorf("unsupported transfer version: %s", version))
	}

	return msg
}

// SuiteName returns the name of the test suite.
func (s *E2ETestSuite) SuiteName() string {
	return s.testSuiteName
}

// ThreeChainSetup provides the default behaviour to wire up 3 chains in the tests.
func ThreeChainSetup() ChainOptionConfiguration {
	// copy all values of existing chains and tweak to make unique to new chain.
	return func(options *ChainOptions) {
		chainCSpec := *options.ChainSpecs[0] // nolint

		chainCSpec.ChainID = "chainC-1"
		chainCSpec.Name = "simapp-c"

		options.ChainSpecs = append(options.ChainSpecs, &chainCSpec)
	}
}

// DefaultChainOptions returns the default chain options for the test suite based on the provided chains.
func defaultChannelOpts(chains []ibc.Chain) ibc.CreateChannelOptions {
	channelOptions := ibc.DefaultChannelOpts()
	channelOptions.Version = determineDefaultTransferVersion(chains)
	return channelOptions
}

// determineDefaultTransferVersion determines the version of transfer that should be used with an arbitrary number of chains.
// the default is V2, but if any chain does not support V2, then V1 is used.
func determineDefaultTransferVersion(chains []ibc.Chain) string {
	for _, chain := range chains {
		chainVersion := chain.Config().Images[0].Version
		if !testvalues.ICS20v2FeatureReleases.IsSupported(chainVersion) {
			return transfertypes.V1
		}
	}
	return transfertypes.V2
}
