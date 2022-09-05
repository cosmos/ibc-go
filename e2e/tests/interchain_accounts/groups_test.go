package interchain_accounts

import (
	"context"
	"testing"
	"time"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	grouptypes "github.com/cosmos/cosmos-sdk/x/group"

	controllertypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
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

// TestInterchainAccountsGroupsIntegration_Success runs a full integration test between the x/group module and ICS27 interchain accounts.
// 1. Create a group
// 2. Create a group policy
// 3. Query group policy address
// 4. Create group proposal: MsgRegisterAccount
// 5. Vote on proposal
// 6. Exec propsoal
// 7. Query interchain account address
// 8. Fund the interchain account on chainB
// 9. Create group proposal: MsgSubmitTx
func (s *InterchainAccountsGroupsTestSuite) TestInterchainAccountsGroupsIntegration_Success() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	_ = relayer
	_ = chainB

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	t.Run("create and configure group and group policy", func(t *testing.T) {
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

		groupPolicyAddr := s.QueryGroupPolicyAddress(ctx, chainA)
		msgRegisterAccount := controllertypes.NewMsgRegisterAccount(ibctesting.FirstConnectionID, groupPolicyAddr, icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID))

		msgSubmitProposal, err := grouptypes.NewMsgSubmitProposal(groupPolicyAddr, []string{chainAAddress}, []sdk.Msg{msgRegisterAccount}, "", grouptypes.Exec_EXEC_TRY)
		s.Require().NoError(err)

		s.BroadcastMessages(ctx, chainA, chainAWallet, msgSubmitProposal)

		interchainAccAddr, err := s.QueryInterchainAccount(ctx, chainA, groupPolicyAddr, ibctesting.FirstConnectionID)
		s.Require().NoError(err)

		t.Logf("successfully registered interchain account via controller group: %s", interchainAccAddr)
	})

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	// controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	// chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	// var hostAccount string

	// t.Run("start relayer", func(t *testing.T) {
	// 	s.StartRelayer(relayer)
	// })

}
