package testsuite

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	test "github.com/strangelove-ventures/interchaintest/v7/testutil"

	"github.com/cosmos/ibc-go/e2e/semverutil"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	feetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

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

	// Retry the operation a few times if the user signing the transaction is a relayer. (See issue #3264)
	var resp sdk.TxResponse
	var err error
	broadcastFunc := func() (sdk.TxResponse, error) {
		return cosmos.BroadcastTx(ctx, broadcaster, user, msgs...)
	}
	if s.relayers.ContainsRelayer(s.T().Name(), user) {
		// Retry five times, the value of 5 chosen is arbitrary.
		resp, err = s.retryNtimes(broadcastFunc, 5)
	} else {
		resp, err = broadcastFunc()
	}
	if err != nil {
		return sdk.TxResponse{}, err
	}

	chainA, chainB := s.GetChains()
	err = test.WaitForBlocks(ctx, 2, chainA, chainB)
	return resp, err
}

// retryNtimes retries the provided function up to the provided number of attempts.
func (s *E2ETestSuite) retryNtimes(f func() (sdk.TxResponse, error), attempts int) (sdk.TxResponse, error) {
	// Ignore account sequence mismatch errors.
	retryMessages := []string{"account sequence mismatch"}
	var resp sdk.TxResponse
	var err error
	// If the response's raw log doesn't contain any of the allowed prefixes we return, else, we retry.
	for i := 0; i < attempts; i++ {
		resp, err = f()
		if err != nil {
			return sdk.TxResponse{}, err
		}
		if !containsMessage(resp.RawLog, retryMessages) {
			return resp, err
		}
		s.T().Logf("retrying tx due to non deterministic failure: %+v", resp)
	}
	return resp, err
}

// containsMessages returns true if the string s contains any of the messages in the slice.
func containsMessage(s string, messages []string) bool {
	for _, message := range messages {
		if strings.Contains(s, message) {
			return true
		}
	}
	return false
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

// Transfer broadcasts a MsgTransfer message.
func (s *E2ETestSuite) Transfer(ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet,
	portID, channelID string, token sdk.Coin, sender, receiver string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, memo string,
) (sdk.TxResponse, error) {
	msg := transfertypes.NewMsgTransfer(portID, channelID, token, sender, receiver, timeoutHeight, timeoutTimestamp, memo)
	return s.BroadcastMessages(ctx, chain, user, msg)
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
