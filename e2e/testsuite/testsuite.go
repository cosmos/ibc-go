package testsuite

import (
	"context"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	paramsproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/ibc-go/e2e/testconfig"
	feetypes "github.com/cosmos/ibc-go/v5/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
)

const (
	// ChainARelayerName is the name given to the relayer wallet on ChainA
	ChainARelayerName = "rlyA"
	// ChainBRelayerName is the name given to the relayer wallet on ChainB
	ChainBRelayerName = "rlyB"

	// emptyLogs is the string value returned from `BroadcastMessages`. There are some situations in which
	// the result is empty, when this happens we include the raw logs instead to get as much information
	// amount the failure as possible.
	emptyLogs = "[]"
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
	pathNameIndex uint64
}

// GRPCClients holds a reference to any GRPC clients that are needed by the tests.
// These should typically be used for query clients only. If we need to make changes, we should
// use E2ETestSuite.BroadcastMessages to broadcast transactions instead.
type GRPCClients struct {
	ClientQueryClient  clienttypes.QueryClient
	ChannelQueryClient channeltypes.QueryClient
	FeeQueryClient     feetypes.QueryClient
	ICAQueryClient     intertxtypes.QueryClient

	// SDK query clients
	GovQueryClient    govtypes.QueryClient
	ParamsQueryClient paramsproposaltypes.QueryClient
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
func (s *E2ETestSuite) GetRelayerUsers(ctx context.Context, chainOpts ...testconfig.ChainOptionConfiguration) (*ibc.Wallet, *ibc.Wallet) {
	chainA, chainB := s.GetChains(chainOpts...)
	chainAAccountBytes, err := chainA.GetAddress(ctx, ChainARelayerName)
	s.Require().NoError(err)

	chainBAccountBytes, err := chainB.GetAddress(ctx, ChainBRelayerName)
	s.Require().NoError(err)

	chainARelayerUser := ibc.Wallet{
		Address: string(chainAAccountBytes),
		KeyName: ChainARelayerName,
	}

	chainBRelayerUser := ibc.Wallet{
		Address: string(chainBAccountBytes),
		KeyName: ChainBRelayerName,
	}
	return &chainARelayerUser, &chainBRelayerUser
}

// SetupChainsRelayerAndChannel create two chains, a relayer, establishes a connection and creates a channel
// using the given channel options. The relayer returned by this function has not yet started. It should be started
// with E2ETestSuite.StartRelayer if needed.
// This should be called at the start of every test, unless fine grained control is required.
func (s *E2ETestSuite) SetupChainsRelayerAndChannel(ctx context.Context, channelOpts ...func(*ibc.CreateChannelOptions)) (ibc.Relayer, ibc.ChannelOutput) {
	chainA, chainB := s.GetChains()

	r := newCosmosRelayer(s.T(), testconfig.FromEnv(), s.logger, s.DockerClient, s.network)

	pathName := s.generatePathName()

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

	eRep := s.GetRelayerExecReporter()
	s.Require().NoError(ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:          s.T().Name(),
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

	s.initGRPCClients(chainA)
	s.initGRPCClients(chainB)

	chainAChannels, err := r.GetChannels(ctx, eRep, chainA.Config().ChainID)
	s.Require().NoError(err)
	return r, chainAChannels[len(chainAChannels)-1]
}

// generatePathName generates the path name using the test suites name
func (s *E2ETestSuite) generatePathName() string {
	pathName := fmt.Sprintf("%s-path-%d", s.T().Name(), s.pathNameIndex)
	s.pathNameIndex++
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
func (s *E2ETestSuite) BroadcastMessages(ctx context.Context, chain *cosmos.CosmosChain, user *ibc.Wallet, msgs ...sdk.Msg) (sdk.TxResponse, error) {
	broadcaster := cosmos.NewBroadcaster(s.T(), chain)
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
	user *ibc.Wallet, portID, channelID, relayerAddr, counterpartyPayeeAddr string) (sdk.TxResponse, error) {
	msg := feetypes.NewMsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayeeAddr)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

// PayPacketFeeAsync broadcasts a MsgPayPacketFeeAsync message.
func (s *E2ETestSuite) PayPacketFeeAsync(
	ctx context.Context,
	chain *cosmos.CosmosChain,
	user *ibc.Wallet,
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
		return ibc.Wallet{}, ibc.Wallet{}, fmt.Errorf("unable to find chain A relayer wallet")
	}

	chainBRelayerWallet, ok := relayer.GetWallet(chainB.Config().ChainID)
	if !ok {
		return ibc.Wallet{}, ibc.Wallet{}, fmt.Errorf("unable to find chain B relayer wallet")
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

// StopRelayer stops the given relayer.
func (s *E2ETestSuite) StopRelayer(ctx context.Context, relayer ibc.Relayer) {
	err := relayer.StopRelayer(ctx, s.GetRelayerExecReporter())
	s.Require().NoError(err)
}

// CreateUserOnChainA creates a user with the given amount of funds on chain A.
func (s *E2ETestSuite) CreateUserOnChainA(ctx context.Context, amount int64) *ibc.Wallet {
	chainA, _ := s.GetChains()
	return ibctest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), amount, chainA)[0]
}

// CreateUserOnChainB creates a user with the given amount of funds on chain B.
func (s *E2ETestSuite) CreateUserOnChainB(ctx context.Context, amount int64) *ibc.Wallet {
	_, chainB := s.GetChains()
	return ibctest.GetAndFundTestUsers(s.T(), ctx, strings.ReplaceAll(s.T().Name(), " ", "-"), amount, chainB)[0]
}

// GetChainANativeBalance gets the balance of a given user on chain A.
func (s *E2ETestSuite) GetChainANativeBalance(ctx context.Context, user *ibc.Wallet) (int64, error) {
	chainA, _ := s.GetChains()
	return GetNativeChainBalance(ctx, chainA, user)
}

// GetChainBNativeBalance gets the balance of a given user on chain B.
func (s *E2ETestSuite) GetChainBNativeBalance(ctx context.Context, user *ibc.Wallet) (int64, error) {
	_, chainB := s.GetChains()
	return GetNativeChainBalance(ctx, chainB, user)
}

// GetChainGRCPClients gets the GRPC clients associated with the given chain.
func (s *E2ETestSuite) GetChainGRCPClients(chain ibc.Chain) GRPCClients {
	cs, ok := s.grpcClients[chain.Config().ChainID]
	s.Require().True(ok, "chain %s does not have GRPC clients", chain.Config().ChainID)
	return cs
}

// initGRPCClients establishes GRPC clients with the given chain.
// The created GRPCClients can be retrieved with GetChainGRCPClients.
func (s *E2ETestSuite) initGRPCClients(chain *cosmos.CosmosChain) {
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
		ClientQueryClient:  clienttypes.NewQueryClient(grpcConn),
		ChannelQueryClient: channeltypes.NewQueryClient(grpcConn),
		FeeQueryClient:     feetypes.NewQueryClient(grpcConn),
		ICAQueryClient:     intertxtypes.NewQueryClient(grpcConn),
		GovQueryClient:     govtypes.NewQueryClient(grpcConn),
		ParamsQueryClient:  paramsproposaltypes.NewQueryClient(grpcConn),
	}
}

// AssertValidTxResponse verifies that an sdk.TxResponse
// has non-empty values.
func (s *E2ETestSuite) AssertValidTxResponse(resp sdk.TxResponse) {
	respLogsMsg := resp.Logs.String()
	if respLogsMsg == emptyLogs {
		respLogsMsg = resp.RawLog
	}
	s.Require().NotEqual(int64(0), resp.GasUsed, respLogsMsg)
	s.Require().NotEqual(int64(0), resp.GasWanted, respLogsMsg)
	s.Require().NotEmpty(resp.Events, respLogsMsg)
	s.Require().NotEmpty(resp.Data, respLogsMsg)
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
	client, network := ibctest.DockerSetup(s.T())

	s.logger = zap.NewExample()
	s.DockerClient = client
	s.network = network

	logger := zaptest.NewLogger(s.T())

	numValidators, numFullNodes := getValidatorsAndFullNodes(chainOptions)

	chainA := cosmos.NewCosmosChain(s.T().Name(), *chainOptions.ChainAConfig, numValidators, numFullNodes, logger)
	chainB := cosmos.NewCosmosChain(s.T().Name(), *chainOptions.ChainBConfig, numValidators, numFullNodes, logger)
	return chainA, chainB
}

// getValidatorsAndFullNodes returns the number of validators and full nodes which should be used
// for the given chain config.
func getValidatorsAndFullNodes(chainOptions testconfig.ChainOptions) (int, int) {
	// TODO: the icad tests are failing with a larger number of validators.
	// this function can be removed once https://github.com/cosmos/ibc-go/issues/2104 is resolved.
	numValidators := 4
	numFullNodes := 1
	isIcadImage := strings.Contains(chainOptions.ChainAConfig.Images[0].Repository, "icad")
	if isIcadImage {
		numValidators = 1
		numFullNodes = 0
	}
	return numValidators, numFullNodes
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
func GetNativeChainBalance(ctx context.Context, chain ibc.Chain, user *ibc.Wallet) (int64, error) {
	bal, err := chain.GetBalance(ctx, user.Bech32Address(chain.Config().Bech32Prefix), chain.Config().Denom)
	if err != nil {
		return -1, err
	}
	return bal, nil
}

// ExecuteGovProposal submits the given governance proposal using the provided user and uses all validators to vote yes on the proposal.
// It ensure the proposal successfully passes.
func (s *E2ETestSuite) ExecuteGovProposal(ctx context.Context, chain *cosmos.CosmosChain, user *ibc.Wallet, content govtypes.Content) {
	sender, err := sdk.AccAddressFromBech32(user.Bech32Address(chain.Config().Bech32Prefix))
	s.Require().NoError(err)

	msgSubmitProposal, err := govtypes.NewMsgSubmitProposal(content, sdk.NewCoins(sdk.NewCoin(chain.Config().Denom, govtypes.DefaultMinDepositTokens)), sender)
	s.Require().NoError(err)

	txResp, err := s.BroadcastMessages(ctx, chain, user, msgSubmitProposal)
	s.Require().NoError(err)
	s.AssertValidTxResponse(txResp)

	// TODO: replace with parsed proposal ID from MsgSubmitProposalResponse
	// https://github.com/cosmos/ibc-go/issues/2122

	proposal, err := s.QueryProposal(ctx, chain, 1)
	s.Require().NoError(err)
	s.Require().Equal(govtypes.StatusVotingPeriod, proposal.Status)

	err = chain.VoteOnProposalAllValidators(ctx, "1", ibc.ProposalVoteYes)
	s.Require().NoError(err)

	// ensure voting period has not passed before validators finished voting
	proposal, err = s.QueryProposal(ctx, chain, 1)
	s.Require().NoError(err)
	s.Require().Equal(govtypes.StatusVotingPeriod, proposal.Status)

	time.Sleep(time.Second * 30) // pass proposal

	proposal, err = s.QueryProposal(ctx, chain, 1)
	s.Require().NoError(err)
	s.Require().Equal(govtypes.StatusPassed, proposal.Status)
}
