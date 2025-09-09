package testsuite

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	test "github.com/cosmos/interchaintest/v10/testutil"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testsuite/sanitize"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

// BroadcastMessages broadcasts the provided messages to the given chain and signs them on behalf of the provided user.
// Once the broadcast response is returned, we wait for a few blocks to be created on the chain the message was broadcast to.
func (s *E2ETestSuite) BroadcastMessages(ctx context.Context, chain ibc.Chain, user ibc.Wallet, msgs ...sdk.Msg) sdk.TxResponse {
	cosmosChain, ok := chain.(*cosmos.CosmosChain)
	if !ok {
		panic("BroadcastMessages expects a cosmos.CosmosChain")
	}

	broadcaster := cosmos.NewBroadcaster(s.T(), cosmosChain)

	// strip out any fields that may not be supported for the given chain version.
	msgs = sanitize.Messages(cosmosChain.Nodes()[0].Image.Version, msgs...)

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
	if s.relayerWallets.ContainsRelayer(s.T().Name(), user) {
		// Retry five times, the value of 5 chosen is arbitrary.
		resp, err = s.retryNtimes(broadcastFunc, 5)
	} else {
		resp, err = broadcastFunc()
	}
	s.Require().NoError(err)

	s.Require().NoError(test.WaitForBlocks(ctx, 2, chain))
	s.T().Logf("blocks created on chain %s", chain.Config().ChainID)
	return resp
}

// retryNtimes retries the provided function up to the provided number of attempts.
func (s *E2ETestSuite) retryNtimes(f func() (sdk.TxResponse, error), attempts int) (sdk.TxResponse, error) {
	// Ignore account sequence mismatch errors.
	retryMessages := []string{"account sequence mismatch"}
	var resp sdk.TxResponse
	var err error
	// If the response's raw log doesn't contain any of the allowed prefixes we return, else, we retry.
	for range attempts {
		resp, err = f()
		if err != nil {
			return sdk.TxResponse{}, err
		}
		// If the response's raw log doesn't contain any of the allowed prefixes we return, else, we retry.
		if !slices.ContainsFunc(retryMessages, func(s string) bool { return strings.Contains(resp.RawLog, s) }) {
			return resp, err
		}
		s.T().Logf("retrying tx due to non deterministic failure: %+v", resp)
	}
	return resp, err
}

// AssertTxFailure verifies that an sdk.TxResponse has failed.
func (s *E2ETestSuite) AssertTxFailure(resp sdk.TxResponse, expectedError *errorsmod.Error, alternativeError ...*errorsmod.Error) {
	errorMsg := fmt.Sprintf("%+v", resp)
	// In older versions, the codespace and abci codes were different. So in compatibility tests
	// we can not make assertions on them.
	if GetChainATag() == GetChainBTag() {
		s.Require().Equal(expectedError.ABCICode(), resp.Code, errorMsg)
		s.Require().Equal(expectedError.Codespace(), resp.Codespace, errorMsg)
	}
	// Verify that the error message contains the expected error message or one of the alternative error messages.
	if strings.Contains(resp.RawLog, expectedError.Error()) {
		return
	}

	for _, altErr := range alternativeError {
		if strings.Contains(resp.RawLog, altErr.Error()) {
			return
		}
	}
	s.Require().FailNow(fmt.Sprintf("expected error: %s, got: %s", expectedError.Error(), resp.RawLog))
}

// AssertTxSuccess verifies that an sdk.TxResponse has succeeded.
func (s *E2ETestSuite) AssertTxSuccess(resp sdk.TxResponse) {
	errorMsg := addDebuggingInformation(fmt.Sprintf("%+v", resp))
	s.Require().Equal(resp.Code, uint32(0), errorMsg)
	s.Require().NotEmpty(resp.TxHash, errorMsg)
	s.Require().NotEqual(int64(0), resp.GasUsed, errorMsg)
	s.Require().NotEqual(int64(0), resp.GasWanted, errorMsg)
	s.Require().NotEmpty(resp.Events, errorMsg)
	s.Require().NotEmpty(resp.Data, errorMsg)
}

// addDebuggingInformation adds additional debugging information to the error message
// based on common types of errors that can occur.
func addDebuggingInformation(errorMsg string) string {
	if strings.Contains(errorMsg, "errUnknownField") {
		errorMsg += `

This error is likely due to a new an unrecognized proto field being provided to a chain using an older version of the sdk.
If this is a compatibility test, ensure that the fields are being sanitized in the sanitize.Messages function.

`
	}
	return errorMsg
}

// ExecuteAndPassGovV1Proposal submits a v1 governance proposal using the provided user and message and uses all validators
// to vote yes on the proposal. It ensures the proposal successfully passes.
func (s *E2ETestSuite) ExecuteAndPassGovV1Proposal(ctx context.Context, msg sdk.Msg, chain ibc.Chain, user ibc.Wallet) {
	err := s.ExecuteGovV1Proposal(ctx, msg, chain, user)
	s.Require().NoError(err)
}

// ExecuteGovV1Proposal submits a v1 governance proposal using the provided user and message and uses all validators
// to vote yes on the proposal.
func (s *E2ETestSuite) ExecuteGovV1Proposal(ctx context.Context, msg sdk.Msg, chain ibc.Chain, user ibc.Wallet) error {
	cosmosChain, ok := chain.(*cosmos.CosmosChain)
	if !ok {
		panic("ExecuteAndPassGovV1Proposal must be passed a cosmos.CosmosChain")
	}

	sender, err := sdk.AccAddressFromBech32(user.FormattedAddress())
	s.Require().NoError(err)

	proposalID := s.proposalIDs[cosmosChain.Config().ChainID]
	defer func() {
		s.proposalIDs[cosmosChain.Config().ChainID] = proposalID + 1
	}()

	msgs := []sdk.Msg{msg}

	msgSubmitProposal, err := govtypesv1.NewMsgSubmitProposal(
		msgs,
		sdk.NewCoins(sdk.NewCoin(cosmosChain.Config().Denom, sdkmath.NewInt(testvalues.DefaultGovV1ProposalTokenAmount))),
		sender.String(),
		"",
		fmt.Sprintf("e2e gov proposal: %d", proposalID),
		fmt.Sprintf("executing gov proposal %d", proposalID),
		false,
	)
	s.Require().NoError(err)

	s.T().Logf("submitting proposal with ID: %d", proposalID)
	resp := s.BroadcastMessages(ctx, cosmosChain, user, msgSubmitProposal)
	s.AssertTxSuccess(resp)

	s.Require().NoError(cosmosChain.VoteOnProposalAllValidators(ctx, proposalID, cosmos.ProposalVoteYes))

	s.T().Logf("validators voted %s on proposal with ID: %d", cosmos.ProposalVoteYes, proposalID)
	return s.waitForGovV1ProposalToPass(ctx, cosmosChain, proposalID)
}

// waitForGovV1ProposalToPass polls for the entire voting period to see if the proposal has passed.
// if the proposal has not passed within the duration of the voting period, an error is returned.
func (s *E2ETestSuite) waitForGovV1ProposalToPass(ctx context.Context, chain ibc.Chain, proposalID uint64) error {
	var govProposal *govtypesv1.Proposal
	// poll for the query for the entire voting period to see if the proposal has passed.
	err := test.WaitForCondition(testvalues.VotingPeriod, 10*time.Second, func() (bool, error) {
		s.T().Logf("waiting for proposal with ID: %d to pass", proposalID)
		proposalResp, err := query.GRPCQuery[govtypesv1.QueryProposalResponse](ctx, chain, &govtypesv1.QueryProposalRequest{
			ProposalId: proposalID,
		})
		if err != nil {
			return false, err
		}

		govProposal = proposalResp.Proposal
		return govProposal.Status == govtypesv1.StatusPassed, nil
	})

	// in the case of a failed proposal, we wrap the polling error with additional information about why the proposal failed.
	if err != nil && govProposal.FailedReason != "" {
		err = errorsmod.Wrap(err, govProposal.FailedReason)
	}
	return err
}

// ExecuteAndPassGovV1Beta1Proposal submits the given v1beta1 governance proposal using the provided user and uses all validators to vote yes on the proposal.
// It ensures the proposal successfully passes.
func (s *E2ETestSuite) ExecuteAndPassGovV1Beta1Proposal(ctx context.Context, chain ibc.Chain, user ibc.Wallet, content govtypesv1beta1.Content) {
	cosmosChain, ok := chain.(*cosmos.CosmosChain)
	if !ok {
		panic("ExecuteAndPassGovV1Beta1Proposal must be passed a cosmos.CosmosChain")
	}

	txResp := s.ExecuteGovV1Beta1Proposal(ctx, cosmosChain, user, content)
	s.AssertTxSuccess(txResp)

	var submitProposalResponse govtypesv1beta1.MsgSubmitProposalResponse
	s.Require().NoError(UnmarshalMsgResponses(txResp, &submitProposalResponse))

	proposalID := submitProposalResponse.ProposalId
	defer func() {
		s.proposalIDs[chain.Config().ChainID] = proposalID + 1
	}()

	proposalResp, err := query.GRPCQuery[govtypesv1beta1.QueryProposalResponse](ctx, cosmosChain, &govtypesv1beta1.QueryProposalRequest{
		ProposalId: proposalID,
	})
	s.Require().NoError(err)

	proposal := proposalResp.Proposal
	s.Require().Equal(govtypesv1beta1.StatusVotingPeriod, proposal.Status)

	err = cosmosChain.VoteOnProposalAllValidators(ctx, proposalID, cosmos.ProposalVoteYes)
	s.Require().NoError(err)

	// ensure voting period has not passed before validators finished voting
	proposalResp, err = query.GRPCQuery[govtypesv1beta1.QueryProposalResponse](ctx, cosmosChain, &govtypesv1beta1.QueryProposalRequest{
		ProposalId: proposalID,
	})
	s.Require().NoError(err)

	proposal = proposalResp.Proposal
	s.Require().Equal(govtypesv1beta1.StatusVotingPeriod, proposal.Status)

	err = s.waitForGovV1Beta1ProposalToPass(ctx, cosmosChain, proposalID)
	s.Require().NoError(err)
}

// waitForGovV1Beta1ProposalToPass polls for the entire voting period to see if the proposal has passed.
// if the proposal has not passed within the duration of the voting period, an error is returned.
func (*E2ETestSuite) waitForGovV1Beta1ProposalToPass(ctx context.Context, chain ibc.Chain, proposalID uint64) error {
	// poll for the query for the entire voting period to see if the proposal has passed.
	return test.WaitForCondition(testvalues.VotingPeriod, 10*time.Second, func() (bool, error) {
		proposalResp, err := query.GRPCQuery[govtypesv1beta1.QueryProposalResponse](ctx, chain, &govtypesv1beta1.QueryProposalRequest{
			ProposalId: proposalID,
		})
		if err != nil {
			return false, err
		}

		proposal := proposalResp.Proposal
		return proposal.Status == govtypesv1beta1.StatusPassed, nil
	})
}

// ExecuteGovV1Beta1Proposal submits a v1beta1 governance proposal using the provided content.
func (s *E2ETestSuite) ExecuteGovV1Beta1Proposal(ctx context.Context, chain ibc.Chain, user ibc.Wallet, content govtypesv1beta1.Content) sdk.TxResponse {
	sender, err := sdk.AccAddressFromBech32(user.FormattedAddress())
	s.Require().NoError(err)

	msgSubmitProposal, err := govtypesv1beta1.NewMsgSubmitProposal(content, sdk.NewCoins(sdk.NewCoin(chain.Config().Denom, govtypesv1beta1.DefaultMinDepositTokens)), sender)
	s.Require().NoError(err)

	return s.BroadcastMessages(ctx, chain, user, msgSubmitProposal)
}

// Transfer broadcasts a MsgTransfer message.
func (s *E2ETestSuite) Transfer(ctx context.Context, chain ibc.Chain, user ibc.Wallet,
	portID, channelID string, token sdk.Coin, sender, receiver string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64,
	memo string,
) sdk.TxResponse {
	channel, err := query.Channel(ctx, chain, portID, channelID)
	s.Require().NoError(err)
	s.Require().NotNil(channel)

	msg := GetMsgTransfer(portID, channelID, channel.Version, token, sender, receiver, timeoutHeight, timeoutTimestamp, memo)

	return s.BroadcastMessages(ctx, chain, user, msg)
}

// QueryTxsByEvents runs the QueryTxsByEvents command on the given chain.
// https://github.com/cosmos/cosmos-sdk/blob/65ab2530cc654fd9e252b124ed24cbaa18023b2b/x/auth/client/cli/query.go#L33
func (*E2ETestSuite) QueryTxsByEvents(
	ctx context.Context, chain ibc.Chain,
	page, limit int, queryReq, orderBy string,
) (*txtypes.GetTxsEventResponse, error) {
	cosmosChain, ok := chain.(*cosmos.CosmosChain)
	if !ok {
		return nil, errors.New("QueryTxsByEvents must be passed a cosmos.CosmosChain")
	}

	req := &txtypes.GetTxsEventRequest{
		Page:  uint64(page),
		Limit: uint64(limit),
		Query: queryReq,
	}

	if !testvalues.TransactionEventQueryFeatureReleases.IsSupported(chain.Config().Images[0].Version) {
		req.Events = []string{queryReq}
		req.Query = ""
	}

	res, err := query.GRPCQuery[txtypes.GetTxsEventResponse](ctx, cosmosChain, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ExtractValueFromEvents extracts the value of an attribute from a list of events.
// If the attribute is not found, the function returns an empty string and false.
// If the attribute is found, the function returns the value and true.
func (*E2ETestSuite) ExtractValueFromEvents(events []abci.Event, eventType, attrKey string) (string, bool) {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}

		for _, attr := range event.Attributes {
			if attr.Key != attrKey {
				continue
			}

			return attr.Value, true
		}
	}

	return "", false
}
