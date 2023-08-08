package testsuite

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	grouptypes "github.com/cosmos/cosmos-sdk/x/group"
	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"
	dockerclient "github.com/docker/docker/client"
	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	test "github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/ibc-go/e2e/relayer"
	"github.com/cosmos/ibc-go/e2e/semverutil"
	"github.com/cosmos/ibc-go/e2e/testconfig"
	"github.com/cosmos/ibc-go/e2e/testsuite/diagnostics"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	feetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

const (
	// ChainARelayerName is the name given to the relayer wallet on ChainA
	ChainARelayerName = "rlyA"
	// ChainBRelayerName is the name given to the relayer wallet on ChainB
	ChainBRelayerName = "rlyB"
	// DefaultGasValue is the default gas value used to configure tx.Factory
	DefaultGasValue = 500000
)

// E2ETestSuite has methods and functionality which can be shared among all test suites.
type E2ETestSuite struct {
	suite.Suite

	grpcClients    map[string]GRPCClients
	paths          map[string]path
	logger         *zap.Logger
	DockerClient   *dockerclient.Client
	network        string
	startRelayerFn func(relayer ibc.Relayer)

	// pathNameIndex is the latest index to be used for generating paths
	pathNameIndex int64
}

// GRPCClients holds a reference to any GRPC clients that are needed by the tests.
// These should typically be used for query clients only. If we need to make changes, we should
// use E2ETestSuite.BroadcastMessages to broadcast transactions instead.
type GRPCClients struct {
	ClientQueryClient     clienttypes.QueryClient
	ConnectionQueryClient connectiontypes.QueryClient
	ChannelQueryClient    channeltypes.QueryClient
	FeeQueryClient        feetypes.QueryClient
	ICAQueryClient        controllertypes.QueryClient
	InterTxQueryClient    intertxtypes.QueryClient

	// SDK query clients
	GovQueryClient    govtypesv1beta1.QueryClient
	GovQueryClientV1  govtypesv1.QueryClient
	GroupsQueryClient grouptypes.QueryClient
	ParamsQueryClient paramsproposaltypes.QueryClient
	AuthQueryClient   authtypes.QueryClient
	AuthZQueryClient  authz.QueryClient

	ConsensusServiceClient tmservice.ServiceClient
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

// GetRelayerUsers returns two ibc.Wallet instances which can be used for the relayer users
// on the two chains.
func (s *E2ETestSuite) GetRelayerUsers(ctx context.Context, chainOpts ...testconfig.ChainOptionConfiguration) (ibc.Wallet, ibc.Wallet) {
	chainA, chainB := s.GetChains(chainOpts...)
	chainAAccountBytes, err := chainA.GetAddress(ctx, ChainARelayerName)
	s.Require().NoError(err)

	chainBAccountBytes, err := chainB.GetAddress(ctx, ChainBRelayerName)
	s.Require().NoError(err)

	chainARelayerUser := cosmos.NewWallet(ChainARelayerName, chainAAccountBytes, "", chainA.Config())
	chainBRelayerUser := cosmos.NewWallet(ChainBRelayerName, chainBAccountBytes, "", chainB.Config())

	return chainARelayerUser, chainBRelayerUser
}

// SetupChainsRelayerAndChannel create two chains, a relayer, establishes a connection and creates a channel
// using the given channel options. The relayer returned by this function has not yet started. It should be started
// with E2ETestSuite.StartRelayer if needed.
// This should be called at the start of every test, unless fine grained control is required.
func (s *E2ETestSuite) SetupChainsRelayerAndChannel(ctx context.Context, channelOpts ...func(*ibc.CreateChannelOptions)) (ibc.Relayer, ibc.ChannelOutput) {
	chainA, chainB := s.GetChains()

	r := relayer.New(s.T(), testconfig.LoadConfig().RelayerConfig, s.logger, s.DockerClient, s.network)

	pathName := s.generatePathName()

	channelOptions := ibc.DefaultChannelOpts()
	for _, opt := range channelOpts {
		opt(&channelOptions)
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

	eRep := s.GetRelayerExecReporter()
	s.Require().NoError(ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:  s.T().Name(),
		Client:    s.DockerClient,
		NetworkID: s.network,
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

	s.InitGRPCClients(chainA)
	s.InitGRPCClients(chainB)

	chainAChannels, err := r.GetChannels(ctx, eRep, chainA.Config().ChainID)
	s.Require().NoError(err)
	return r, chainAChannels[len(chainAChannels)-1]
}

// SetupSingleChain creates and returns a single CosmosChain for usage in e2e tests.
// This is useful for testing single chain functionality when performing coordinated upgrades as well as testing localhost ibc client functionality.
// TODO: Actually setup a single chain. Seeing panic: runtime error: index out of range [0] with length 0 when using a single chain.
// issue: https://github.com/strangelove-ventures/interchaintest/issues/401
func (s *E2ETestSuite) SetupSingleChain(ctx context.Context) *cosmos.CosmosChain {
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
func (s *E2ETestSuite) generatePath(ctx context.Context, relayer ibc.Relayer) string {
	chainA, chainB := s.GetChains()
	chainAID := chainA.Config().ChainID
	chainBID := chainB.Config().ChainID

	pathName := s.generatePathName()

	err := relayer.GeneratePath(ctx, s.GetRelayerExecReporter(), chainAID, chainBID, pathName)
	s.Require().NoError(err)

	return pathName
}

// SetupClients creates clients on chainA and chainB using the provided create client options
func (s *E2ETestSuite) SetupClients(ctx context.Context, relayer ibc.Relayer, opts ibc.CreateClientOptions) {
	pathName := s.generatePath(ctx, relayer)
	err := relayer.CreateClients(ctx, s.GetRelayerExecReporter(), pathName, opts)
	s.Require().NoError(err)
}

// UpdateClients updates clients on chainA and chainB
func (s *E2ETestSuite) UpdateClients(ctx context.Context, relayer ibc.Relayer, pathName string) {
	err := relayer.UpdateClients(ctx, s.GetRelayerExecReporter(), pathName)
	s.Require().NoError(err)
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

// BroadcastMessages broadcasts the provided messages to the given chain and signs them on behalf of the provided user.
// Once the broadcast response is returned, we wait for a few blocks to be created on both chain A and chain B.
func (s *E2ETestSuite) BroadcastMessages(ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, msgs ...sdk.Msg) (sdk.TxResponse, error) {
	broadcaster := cosmos.NewBroadcaster(s.T(), chain)

	broadcaster.ConfigureClientContextOptions(func(clientContext client.Context) client.Context {
		// use a codec with all the types our tests care about registered.
		// BroadcastTx will deserialize the response and will not be able to otherwise.
		cdc := Codec()
		return clientContext.WithCodec(cdc).WithTxConfig(authtx.NewTxConfig(cdc, []signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_DIRECT}))
	})

	broadcaster.ConfigureFactoryOptions(func(factory tx.Factory) tx.Factory {
		return factory.WithGas(DefaultGasValue)
	})

	resp, err := cosmos.BroadcastTx(ctx, broadcaster, user, msgs...)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	chainA, chainB := s.GetChains()
	err = test.WaitForBlocks(ctx, 2, chainA, chainB)
	return resp, err
}

// RegisterCounterPartyPayee broadcasts a MsgRegisterCounterpartyPayee message.
func (s *E2ETestSuite) RegisterCounterPartyPayee(ctx context.Context, chain *cosmos.CosmosChain,
	user ibc.Wallet, portID, channelID, relayerAddr, counterpartyPayeeAddr string,
) (sdk.TxResponse, error) {
	msg := feetypes.NewMsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayeeAddr)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

// PayPacketFeeAsync broadcasts a MsgPayPacketFeeAsync message.
func (s *E2ETestSuite) PayPacketFeeAsync(
	ctx context.Context,
	chain *cosmos.CosmosChain,
	user ibc.Wallet,
	packetID channeltypes.PacketId,
	packetFee feetypes.PacketFee,
) (sdk.TxResponse, error) {
	msg := feetypes.NewMsgPayPacketFeeAsync(packetID, packetFee)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

// GetRelayerWallets returns the relayer wallets associated with the chains.
func (s *E2ETestSuite) GetRelayerWallets(relayer ibc.Relayer) (ibc.Wallet, ibc.Wallet, error) {
	chainA, chainB := s.GetChains()
	chainARelayerWallet, ok := relayer.GetWallet(chainA.Config().ChainID)
	if !ok {
		return nil, nil, fmt.Errorf("unable to find chain A relayer wallet")
	}

	chainBRelayerWallet, ok := relayer.GetWallet(chainB.Config().ChainID)
	if !ok {
		return nil, nil, fmt.Errorf("unable to find chain B relayer wallet")
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

	if err := chainA.RecoverKey(ctx, ChainARelayerName, chainARelayerWallet.Mnemonic()); err != nil {
		return fmt.Errorf("could not recover relayer wallet on chain A: %s", err)
	}
	if err := chainB.RecoverKey(ctx, ChainBRelayerName, chainBRelayerWallet.Mnemonic()); err != nil {
		return fmt.Errorf("could not recover relayer wallet on chain B: %s", err)
	}
	return nil
}

// Transfer broadcasts a MsgTransfer message.
func (s *E2ETestSuite) Transfer(ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet,
	portID, channelID string, token sdk.Coin, sender, receiver string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, memo string,
) (sdk.TxResponse, error) {
	msg := transfertypes.NewMsgTransfer(portID, channelID, token, sender, receiver, timeoutHeight, timeoutTimestamp, memo)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

// StartRelayer starts the given relayer.
func (s *E2ETestSuite) StartRelayer(relayer ibc.Relayer) {
	if s.startRelayerFn == nil {
		panic("cannot start relayer before it is created!")
	}

	s.startRelayerFn(relayer)
}

// StopRelayer stops the given relayer.
func (s *E2ETestSuite) StopRelayer(ctx context.Context, relayer ibc.Relayer) {
	err := relayer.StopRelayer(ctx, s.GetRelayerExecReporter())
	s.Require().NoError(err)
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
	return GetNativeChainBalance(ctx, chainA, user)
}

// GetChainBNativeBalance gets the balance of a given user on chain B.
func (s *E2ETestSuite) GetChainBNativeBalance(ctx context.Context, user ibc.Wallet) (int64, error) {
	_, chainB := s.GetChains()
	return GetNativeChainBalance(ctx, chainB, user)
}

// GetChainGRCPClients gets the GRPC clients associated with the given chain.
func (s *E2ETestSuite) GetChainGRCPClients(chain ibc.Chain) GRPCClients {
	cs, ok := s.grpcClients[chain.Config().ChainID]
	s.Require().True(ok, "chain %s does not have GRPC clients", chain.Config().ChainID)
	return cs
}

// InitGRPCClients establishes GRPC clients with the given chain.
// The created GRPCClients can be retrieved with GetChainGRCPClients.
func (s *E2ETestSuite) InitGRPCClients(chain *cosmos.CosmosChain) {
	// Create a connection to the gRPC server.
	grpcConn, err := grpc.Dial(
		chain.GetHostGRPCAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		if err := grpcConn.Close(); err != nil {
			s.T().Logf("failed closing GRPC connection to chain %s: %s", chain.Config().ChainID, err)
		}
	})

	if s.grpcClients == nil {
		s.grpcClients = make(map[string]GRPCClients)
	}

	s.grpcClients[chain.Config().ChainID] = GRPCClients{
		ClientQueryClient:      clienttypes.NewQueryClient(grpcConn),
		ChannelQueryClient:     channeltypes.NewQueryClient(grpcConn),
		FeeQueryClient:         feetypes.NewQueryClient(grpcConn),
		ICAQueryClient:         controllertypes.NewQueryClient(grpcConn),
		InterTxQueryClient:     intertxtypes.NewQueryClient(grpcConn),
		GovQueryClient:         govtypesv1beta1.NewQueryClient(grpcConn),
		GovQueryClientV1:       govtypesv1.NewQueryClient(grpcConn),
		GroupsQueryClient:      grouptypes.NewQueryClient(grpcConn),
		ParamsQueryClient:      paramsproposaltypes.NewQueryClient(grpcConn),
		AuthQueryClient:        authtypes.NewQueryClient(grpcConn),
		AuthZQueryClient:       authz.NewQueryClient(grpcConn),
		ConsensusServiceClient: tmservice.NewServiceClient(grpcConn),
	}
}

// AssertValidTxResponse verifies that an sdk.TxResponse
// has non-empty values.
func (s *E2ETestSuite) AssertValidTxResponse(resp sdk.TxResponse) {
	errorMsg := fmt.Sprintf("%+v", resp)
	s.Require().NotEmpty(resp.TxHash, errorMsg)
	s.Require().NotEqual(int64(0), resp.GasUsed, errorMsg)
	s.Require().NotEqual(int64(0), resp.GasWanted, errorMsg)
	s.Require().NotEmpty(resp.Events, errorMsg)
	s.Require().NotEmpty(resp.Data, errorMsg)
}

// AssertPacketRelayed asserts that the packet commitment does not exist on the sending chain.
// The packet commitment will be deleted upon a packet acknowledgement or timeout.
func (s *E2ETestSuite) AssertPacketRelayed(ctx context.Context, chain *cosmos.CosmosChain, portID, channelID string, sequence uint64) {
	commitment, _ := s.QueryPacketCommitment(ctx, chain, portID, channelID, sequence)
	s.Require().Empty(commitment)
}

// createCosmosChains creates two separate chains in docker containers.
// test and can be retrieved with GetChains.
func (s *E2ETestSuite) createCosmosChains(chainOptions testconfig.ChainOptions) (*cosmos.CosmosChain, *cosmos.CosmosChain) {
	client, network := interchaintest.DockerSetup(s.T())
	t := s.T()

	s.logger = zap.NewExample()
	s.DockerClient = client
	s.network = network

	logger := zaptest.NewLogger(t)

	numValidators, numFullNodes := getValidatorsAndFullNodes(0)
	chainA := cosmos.NewCosmosChain(t.Name(), *chainOptions.ChainAConfig, numValidators, numFullNodes, logger)
	numValidators, numFullNodes = getValidatorsAndFullNodes(1)
	chainB := cosmos.NewCosmosChain(t.Name(), *chainOptions.ChainBConfig, numValidators, numFullNodes, logger)

	// this is intentionally called after the interchaintest.DockerSetup function. The above function registers a
	// cleanup task which deletes all containers. By registering a cleanup function afterwards, it is executed first
	// this allows us to process the logs before the containers are removed.
	t.Cleanup(func() {
		diagnostics.Collect(t, s.DockerClient, chainOptions)
	})

	return chainA, chainB
}

// GetRelayerExecReporter returns a testreporter.RelayerExecReporter instances
// using the current test's testing.T.
func (s *E2ETestSuite) GetRelayerExecReporter() *testreporter.RelayerExecReporter {
	rep := testreporter.NewNopReporter()
	return rep.RelayerExecReporter(s.T())
}

// GetTimeoutHeight returns a timeout height of 1000 blocks above the current block height.
// This function should be used when the timeout is never expected to be reached
func (s *E2ETestSuite) GetTimeoutHeight(ctx context.Context, chain *cosmos.CosmosChain) clienttypes.Height {
	height, err := chain.Height(ctx)
	s.Require().NoError(err)
	return clienttypes.NewHeight(clienttypes.ParseChainID(chain.Config().ChainID), uint64(height)+1000)
}

// GetNativeChainBalance returns the balance of a specific user on a chain using the native denom.
func GetNativeChainBalance(ctx context.Context, chain ibc.Chain, user ibc.Wallet) (int64, error) {
	bal, err := chain.GetBalance(ctx, user.FormattedAddress(), chain.Config().Denom)
	if err != nil {
		return -1, err
	}
	return bal, nil
}

// ExecuteGovProposal submits the given governance proposal using the provided user and uses all validators to vote yes on the proposal.
// It ensures the proposal successfully passes.
func (s *E2ETestSuite) ExecuteGovProposal(ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, content govtypesv1beta1.Content) {
	sender, err := sdk.AccAddressFromBech32(user.FormattedAddress())
	s.Require().NoError(err)

	msgSubmitProposal, err := govtypesv1beta1.NewMsgSubmitProposal(content, sdk.NewCoins(sdk.NewCoin(chain.Config().Denom, govtypesv1beta1.DefaultMinDepositTokens)), sender)
	s.Require().NoError(err)

	txResp, err := s.BroadcastMessages(ctx, chain, user, msgSubmitProposal)
	s.Require().NoError(err)
	s.AssertValidTxResponse(txResp)

	// TODO: replace with parsed proposal ID from MsgSubmitProposalResponse
	// https://github.com/cosmos/ibc-go/issues/2122

	proposal, err := s.QueryProposal(ctx, chain, 1)
	s.Require().NoError(err)
	s.Require().Equal(govtypesv1beta1.StatusVotingPeriod, proposal.Status)

	err = chain.VoteOnProposalAllValidators(ctx, "1", cosmos.ProposalVoteYes)
	s.Require().NoError(err)

	// ensure voting period has not passed before validators finished voting
	proposal, err = s.QueryProposal(ctx, chain, 1)
	s.Require().NoError(err)
	s.Require().Equal(govtypesv1beta1.StatusVotingPeriod, proposal.Status)

	time.Sleep(testvalues.VotingPeriod) // pass proposal

	proposal, err = s.QueryProposal(ctx, chain, 1)
	s.Require().NoError(err)
	s.Require().Equal(govtypesv1beta1.StatusPassed, proposal.Status)
}

// govv1ProposalTitleAndSummary represents the releases that support the new title and summary fields.
var govv1ProposalTitleAndSummary = semverutil.FeatureReleases{
	MajorVersion: "v7",
}

// ExecuteGovProposalV1 submits a governance proposal using the provided user and message and uses all validators
// to vote yes on the proposal. It ensures the proposal successfully passes.
func (s *E2ETestSuite) ExecuteGovProposalV1(ctx context.Context, msg sdk.Msg, chain *cosmos.CosmosChain, user ibc.Wallet, proposalID uint64) {
	sender, err := sdk.AccAddressFromBech32(user.FormattedAddress())
	s.Require().NoError(err)

	msgs := []sdk.Msg{msg}
	msgSubmitProposal, err := govtypesv1.NewMsgSubmitProposal(msgs, sdk.NewCoins(sdk.NewCoin(chain.Config().Denom, govtypesv1.DefaultMinDepositTokens)), sender.String(), "", fmt.Sprintf("e2e gov proposal: %d", proposalID), fmt.Sprintf("executing gov proposal %d", proposalID))
	s.Require().NoError(err)

	if !govv1ProposalTitleAndSummary.IsSupported(chain.Nodes()[0].Image.Version) {
		msgSubmitProposal.Title = ""
		msgSubmitProposal.Summary = ""
	}

	resp, err := s.BroadcastMessages(ctx, chain, user, msgSubmitProposal)
	s.AssertValidTxResponse(resp)
	s.Require().NoError(err)

	s.Require().NoError(chain.VoteOnProposalAllValidators(ctx, strconv.Itoa(int(proposalID)), cosmos.ProposalVoteYes))

	time.Sleep(testvalues.VotingPeriod)

	proposal, err := s.QueryProposalV1(ctx, chain, proposalID)
	s.Require().NoError(err)
	s.Require().Equal(govtypesv1.StatusPassed, proposal.Status)
}

// QueryModuleAccountAddress returns the sdk.AccAddress of a given module name.
func (s *E2ETestSuite) QueryModuleAccountAddress(ctx context.Context, moduleName string, chain *cosmos.CosmosChain) (sdk.AccAddress, error) {
	authClient := s.GetChainGRCPClients(chain).AuthQueryClient

	resp, err := authClient.ModuleAccountByName(ctx, &authtypes.QueryModuleAccountByNameRequest{
		Name: moduleName,
	})
	if err != nil {
		return nil, err
	}

	cfg := EncodingConfig()

	var account authtypes.AccountI
	if err := cfg.InterfaceRegistry.UnpackAny(resp.Account, &account); err != nil {
		return nil, err
	}
	moduleAccount, ok := account.(authtypes.ModuleAccountI)
	if !ok {
		return nil, fmt.Errorf("failed to cast account: %T as ModuleAccount", moduleAccount)
	}

	return moduleAccount.GetAddress(), nil
}

// QueryGranterGrants returns all GrantAuthorizations for the given granterAddress.
func (s *E2ETestSuite) QueryGranterGrants(ctx context.Context, chain *cosmos.CosmosChain, granterAddress string) ([]*authz.GrantAuthorization, error) {
	authzClient := s.GetChainGRCPClients(chain).AuthZQueryClient
	queryRequest := &authz.QueryGranterGrantsRequest{
		Granter: granterAddress,
	}

	grants, err := authzClient.GranterGrants(ctx, queryRequest)
	if err != nil {
		return nil, err
	}

	return grants.Grants, nil
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
	if testconfig.IsCI() {
		return 4, 1
	}
	tc := testconfig.LoadConfig()
	return tc.GetChainNumValidators(chainIdx), tc.GetChainNumFullNodes(chainIdx)
}
