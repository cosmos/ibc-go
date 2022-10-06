package interchain_accounts

import (
	"context"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	grouptypes "github.com/cosmos/cosmos-sdk/x/group"
	ibctest "github.com/strangelove-ventures/ibctest/v6"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/strangelove-ventures/ibctest/v6/test"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
	simappparams "github.com/cosmos/ibc-go/v6/testing/simapp/params"
)

func TestInterchainAccountsGroupsTestSuite(t *testing.T) {
	suite.Run(t, new(InterchainAccountsGroupsTestSuite))
}

type InterchainAccountsGroupsTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *InterchainAccountsGroupsTestSuite) QueryGroupPolicyAddress(ctx context.Context, chain ibc.Chain) string {
	queryClient := s.GetChainGRCPClients(chain).GroupsQueryClient
	res, err := queryClient.GroupPoliciesByGroup(ctx, &grouptypes.QueryGroupPoliciesByGroupRequest{
		GroupId: 1, // always use the initial group id
	})
	s.Require().NoError(err)

	return res.GroupPolicies[0].Address
}

func (s *InterchainAccountsGroupsTestSuite) TestInterchainAccountsGroupsIntegration_Success() {
	t := s.T()
	ctx := context.TODO()

	var (
		groupPolicyAddr   string
		interchainAccAddr string
		err               error
	)

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.Bech32Address(chainB.Config().Bech32Prefix)

	t.Run("create group with new threshold decision policy", func(t *testing.T) {
		members := []grouptypes.MemberRequest{
			{
				Address: chainAAddress,
				Weight:  "1",
			},
		}

		decisionPolicy := grouptypes.NewThresholdDecisionPolicy("1", time.Duration(time.Minute), time.Duration(0))
		msgCreateGroupWithPolicy, err := grouptypes.NewMsgCreateGroupWithPolicy(chainAAddress, members, "ics27-controller-group", "ics27-controller-policy", true, decisionPolicy)
		s.Require().NoError(err)

		txResp, err := s.BroadcastMessages(ctx, chainA, chainAWallet, msgCreateGroupWithPolicy)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)

	})

	t.Run("submit proposal for MsgRegisterInterchainAccount", func(t *testing.T) {
		groupPolicyAddr = s.QueryGroupPolicyAddress(ctx, chainA)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, groupPolicyAddr, icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID))

		msgSubmitProposal, err := grouptypes.NewMsgSubmitProposal(groupPolicyAddr, []string{chainAAddress}, []sdk.Msg{msgRegisterAccount}, "", grouptypes.Exec_EXEC_UNSPECIFIED)
		s.Require().NoError(err)

		txResp, err := s.BroadcastMessages(ctx, chainA, chainAWallet, msgSubmitProposal)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("vote and exec proposal", func(t *testing.T) {
		msgVote := &grouptypes.MsgVote{
			ProposalId: 1,
			Voter:      chainAAddress,
			Option:     grouptypes.VOTE_OPTION_YES,
			Exec:       grouptypes.Exec_EXEC_TRY,
		}

		txResp, err := s.BroadcastMessages(ctx, chainA, chainAWallet, msgVote)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify interchain account registration success", func(t *testing.T) {
		interchainAccAddr, err = s.QueryInterchainAccount(ctx, chainA, groupPolicyAddr, ibctesting.FirstConnectionID)
		s.Require().NoError(err)

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)
	})

	t.Run("fund interchain account wallet", func(t *testing.T) {
		err := chainB.SendFunds(ctx, ibctest.FaucetAccountKeyName, ibc.WalletAmount{
			Address: interchainAccAddr,
			Amount:  testvalues.StartingTokenAmount,
			Denom:   chainB.Config().Denom,
		})
		s.Require().NoError(err)
	})

	t.Run("submit proposal for MsgSendTx", func(t *testing.T) {
		msgBankSend := &banktypes.MsgSend{
			FromAddress: interchainAccAddr,
			ToAddress:   chainBAddress,
			Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
		}

		cfg := simappparams.MakeTestEncodingConfig()
		banktypes.RegisterInterfaces(cfg.InterfaceRegistry)
		cdc := codec.NewProtoCodec(cfg.InterfaceRegistry)

		bz, err := icatypes.SerializeCosmosTx(cdc, []sdk.Msg{msgBankSend})
		s.Require().NoError(err)

		packetData := icatypes.InterchainAccountPacketData{
			Type: icatypes.EXECUTE_TX,
			Data: bz,
			Memo: "e2e",
		}

		// timeoutTimestamp := time.Now().Add(time.Hour * 24).UnixNano() // TODO: find a better solution
		msgSubmitTx := controllertypes.NewMsgSendTx(groupPolicyAddr, ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)
		msgSubmitProposal, err := grouptypes.NewMsgSubmitProposal(groupPolicyAddr, []string{chainAAddress}, []sdk.Msg{msgSubmitTx}, "", grouptypes.Exec_EXEC_UNSPECIFIED)
		s.Require().NoError(err)

		txResp, err := s.BroadcastMessages(ctx, chainA, chainAWallet, msgSubmitProposal)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("vote and exec proposal", func(t *testing.T) {
		msgVote := &grouptypes.MsgVote{
			ProposalId: 2,
			Voter:      chainAAddress,
			Option:     grouptypes.VOTE_OPTION_YES,
			Exec:       grouptypes.Exec_EXEC_TRY,
		}

		txResp, err := s.BroadcastMessages(ctx, chainA, chainAWallet, msgVote)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("verify tokens transferred", func(t *testing.T) {
		s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")

		balance, err := chainB.GetBalance(ctx, chainBAddress, chainB.Config().Denom)
		s.Require().NoError(err)

		_, err = chainB.GetBalance(ctx, interchainAccAddr, chainB.Config().Denom)
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
		s.Require().Equal(expected, balance)
	})
}
