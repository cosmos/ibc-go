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
	proposalIDs    map[string]uint64
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

func (s *E2ETestSuite) SetupTest() {
	s.configureGenesisDebugExport()
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
	} else {
		chainAVersion := chainA.Config().Images[0].Version
		chainBVersion := chainB.Config().Images[0].Version

		// select the transfer version based on the chains' version
		transferVersion := transfertypes.V1
		if testvalues.ICS20v2FeatureReleases.IsSupported(chainAVersion) && testvalues.ICS20v2FeatureReleases.IsSupported(chainBVersion) {
			transferVersion = transfertypes.V2
		}
		channelOptions.Version = transferVersion
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

	return chainA
}

// generatePathName generates the path name using the test suites name
func (s *E2ETestSuite) generatePathName() string {
	pathName := s.GetPathName(s.pathNameIndex)
	s.pathNameIndex++
	return pathName
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

	suitePath, ok := s.paths[s.T().Name()]
	if ok {
		return suitePath.chainA, suitePath.chainB
	}

	chainOptions := DefaultChainOptions()
	for _, opt := range chainOpts {
		opt(&chainOptions)
	}

	chainA, chainB := s.createChains(chainOptions)
	suitePath = newPath(chainA, chainB)
	s.paths[s.T().Name()] = suitePath

	if s.proposalIDs == nil {
		s.proposalIDs = map[string]uint64{}
	}

	s.proposalIDs[chainA.Config().ChainID] = 1
	s.proposalIDs[chainB.Config().ChainID] = 1

	return suitePath.chainA, suitePath.chainB
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
	return interchaintest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), sdkmath.NewInt(amount), chainA)[0]
}

// CreateUserOnChainB creates a user with the given amount of funds on chain B.
func (s *E2ETestSuite) CreateUserOnChainB(ctx context.Context, amount int64) ibc.Wallet {
	_, chainB := s.GetChains()
	return interchaintest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), sdkmath.NewInt(amount), chainB)[0]
}

// GetChainANativeBalance gets the balance of a given user on chain A.
func (s *E2ETestSuite) GetChainANativeBalance(ctx context.Context, user ibc.Wallet) (int64, error) {
	chainA, _ := s.GetChains()

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
	_, chainB := s.GetChains()

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
		dumpLogs := LoadConfig().DebugConfig.DumpLogs
		chains := []string{chainOptions.ChainASpec.ChainConfig.Name, chainOptions.ChainBSpec.ChainConfig.Name}
		diagnostics.Collect(t, s.DockerClient, dumpLogs, chains...)
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
func (s *E2ETestSuite) TransferChannelOptions() func(options *ibc.CreateChannelOptions) {
	chainA, chainB := s.GetChains()
	chainAVersion := chainA.Config().Images[0].Version
	chainBVersion := chainB.Config().Images[0].Version

	// select the transfer version based on the chain versions
	transferVersion := transfertypes.V1
	if testvalues.ICS20v2FeatureReleases.IsSupported(chainAVersion) && testvalues.ICS20v2FeatureReleases.IsSupported(chainBVersion) {
		transferVersion = transfertypes.V2
	}

	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = transferVersion
		opts.SourcePortName = transfertypes.PortID
		opts.DestPortName = transfertypes.PortID
	}
}

// FeeMiddlewareChannelOptions configures both of the chains to have fee middleware enabled.
func (s *E2ETestSuite) FeeMiddlewareChannelOptions() func(options *ibc.CreateChannelOptions) {
	chainA, chainB := s.GetChains()
	chainAVersion := chainA.Config().Images[0].Version
	chainBVersion := chainB.Config().Images[0].Version

	// select the transfer version based on the chain versions
	transferVersion := transfertypes.V1
	if testvalues.ICS20v2FeatureReleases.IsSupported(chainAVersion) && testvalues.ICS20v2FeatureReleases.IsSupported(chainBVersion) {
		transferVersion = transfertypes.V2
	}

	versionMetadata := feetypes.Metadata{
		FeeVersion: feetypes.Version,
		AppVersion: transferVersion,
	}
	versionBytes, err := feetypes.ModuleCdc.MarshalJSON(&versionMetadata)
	s.Require().NoError(err)

	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = string(versionBytes)
		opts.DestPortName = transfertypes.PortID
		opts.SourcePortName = transfertypes.PortID
	}
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
func GetMsgTransfer(portID, channelID, version string, tokens sdk.Coins, sender, receiver string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, memo string, forwardingPath *transfertypes.ForwardingInfo) *transfertypes.MsgTransfer {
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
		msg = transfertypes.NewMsgTransfer(portID, channelID, tokens, sender, receiver, timeoutHeight, timeoutTimestamp, memo, forwardingPath)
	default:
		panic(fmt.Errorf("unsupported transfer version: %s", version))
	}

	return msg
}
