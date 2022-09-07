package interchain_accounts

import (
	"context"
	"testing"

	ibctest "github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/stretchr/testify/suite"
	"golang.org/x/mod/semver"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"

	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/cosmos/ibc-go/e2e/testconfig"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"

	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	feetypes "github.com/cosmos/ibc-go/v5/modules/apps/29-fee/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
)

func TestInterchainAccountsTestSuite(t *testing.T) {
	suite.Run(t, new(InterchainAccountsTestSuite))
}

type InterchainAccountsTestSuite struct {
	testsuite.E2ETestSuite
}

// RegisterInterchainAccount will attempt to register an interchain account on the counterparty chain.
func (s *InterchainAccountsTestSuite) RegisterInterchainAccount(ctx context.Context, chain *cosmos.CosmosChain, user *ibc.Wallet, msgRegisterAccount *intertxtypes.MsgRegisterAccount) error {
	txResp, err := s.BroadcastMessages(ctx, chain, user, msgRegisterAccount)
	s.Require().NoError(err)
	s.AssertValidTxResponse(txResp)
	return err
}

// RegisterCounterPartyPayee broadcasts a MsgRegisterCounterpartyPayee message.
func (s *InterchainAccountsTestSuite) RegisterCounterPartyPayee(ctx context.Context, chain *cosmos.CosmosChain,
	user *ibc.Wallet, portID, channelID, relayerAddr, counterpartyPayeeAddr string) (sdk.TxResponse, error) {
	msg := feetypes.NewMsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayeeAddr)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

// getICAVersion returns the version which should be used in the MsgRegisterAccount broadcast from the
// controller chain.
func getICAVersion(chainAVersion, chainBVersion string) string {
	chainBIsGreaterThanChainA := semver.Compare(chainAVersion, chainBVersion) == -1
	if chainBIsGreaterThanChainA {
		// allow version to be specified by the controller chain
		return ""
	}
	// explicitly set the version string because the host chain might not yet support incentivized channels.
	return icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
}

func (s *InterchainAccountsTestSuite) TestMsgSubmitTx_SuccessfulTransfer() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	var hostAccount string

	t.Run("register interchain account", func(t *testing.T) {
		version := getICAVersion(testconfig.GetChainATag(), testconfig.GetChainBTag())
		msgRegisterAccount := intertxtypes.NewMsgRegisterAccount(controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID, version)
		err := s.RegisterInterchainAccount(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.Require().NoError(err)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = s.QueryInterchainAccount(ctx, chainA, controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(hostAccount))

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)
	})

	t.Run("interchain account executes a bank transfer on behalf of the corresponding owner account", func(t *testing.T) {

		t.Run("fund interchain account wallet", func(t *testing.T) {
			// fund the host account account so it has some $$ to send
			err := chainB.SendFunds(ctx, ibctest.FaucetAccountKeyName, ibc.WalletAmount{
				Address: hostAccount,
				Amount:  testvalues.StartingTokenAmount,
				Denom:   chainB.Config().Denom,
			})
			s.Require().NoError(err)
		})

		t.Run("broadcast MsgSubmitTx", func(t *testing.T) {
			// assemble bank transfer message from host account to user account on host chain
			msgSend := &banktypes.MsgSend{
				FromAddress: hostAccount,
				ToAddress:   chainBAccount.Bech32Address(chainB.Config().Bech32Prefix),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
			}

			// assemble submitMessage tx for intertx
			msgSubmitTx, err := intertxtypes.NewMsgSubmitTx(
				msgSend,
				ibctesting.FirstConnectionID,
				controllerAccount.Bech32Address(chainA.Config().Bech32Prefix),
			)
			s.Require().NoError(err)

			// broadcast submitMessage tx from controller account on chain A
			// this message should trigger the sending of an ICA packet over channel-1 (channel created between controller and host)
			// this ICA packet contains the assembled bank transfer message from above, which will be executed by the host account on the host chain.
			resp, err := s.BroadcastMessages(
				ctx,
				chainA,
				controllerAccount,
				msgSubmitTx,
			)

			s.AssertValidTxResponse(resp)
			s.Require().NoError(err)

			s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))
		})

		t.Run("verify tokens transferred", func(t *testing.T) {
			balance, err := chainB.GetBalance(ctx, chainBAccount.Bech32Address(chainB.Config().Bech32Prefix), chainB.Config().Denom)
			s.Require().NoError(err)

			_, err = chainB.GetBalance(ctx, hostAccount, chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance)
		})
	})
}

func (s *InterchainAccountsTestSuite) TestMsgSubmitTx_FailedTransfer_InsufficientFunds() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	// setup 2 accounts: controller account on chain A, a second chain B account.
	// host account will be created when the ICA is registered
	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	var hostAccount string

	t.Run("register interchain account", func(t *testing.T) {
		version := getICAVersion(testconfig.GetChainATag(), testconfig.GetChainBTag())
		msgRegisterAccount := intertxtypes.NewMsgRegisterAccount(controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID, version)
		err := s.RegisterInterchainAccount(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.Require().NoError(err)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = s.QueryInterchainAccount(ctx, chainA, controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(hostAccount))

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)
	})

	t.Run("fail to execute bank transfer over ICA", func(t *testing.T) {
		t.Run("verify empty host wallet", func(t *testing.T) {
			hostAccountBalance, err := chainB.GetBalance(ctx, hostAccount, chainB.Config().Denom)
			s.Require().NoError(err)
			s.Require().Zero(hostAccountBalance)
		})

		t.Run("broadcast MsgSubmitTx", func(t *testing.T) {
			// assemble bank transfer message from host account to user account on host chain
			transferMsg := &banktypes.MsgSend{
				FromAddress: hostAccount,
				ToAddress:   chainBAccount.Bech32Address(chainB.Config().Bech32Prefix),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
			}

			// assemble submitMessage tx for intertx
			submitMsg, err := intertxtypes.NewMsgSubmitTx(
				transferMsg,
				ibctesting.FirstConnectionID,
				controllerAccount.Bech32Address(chainA.Config().Bech32Prefix),
			)
			s.Require().NoError(err)

			// broadcast submitMessage tx from controller account on chain A
			// this message should trigger the sending of an ICA packet over channel-1 (channel created between controller and host)
			// this ICA packet contains the assembled bank transfer message from above, which will be executed by the host account on the host chain.
			resp, err := s.BroadcastMessages(
				ctx,
				chainA,
				controllerAccount,
				submitMsg,
			)

			s.AssertValidTxResponse(resp)
			s.Require().NoError(err)
		})

		t.Run("verify balance is the same", func(t *testing.T) {
			balance, err := chainB.GetBalance(ctx, chainBAccount.Bech32Address(chainB.Config().Bech32Prefix), chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance)
		})
	})
}

func (s *InterchainAccountsTestSuite) TestRegistration_WithGovernance() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	_ = relayer
	chainA, chainB := s.GetChains()
	_ = chainB
	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	//chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	t.Run("create and configure group and group policy", func(t *testing.T) {

		//version := getICAVersion(testconfig.GetChainATag(), testconfig.GetChainBTag())
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		//msgRegisterAccount := intertxtypes.NewMsgRegisterAccount(controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID, version)
		//msgRegisterAccount := controllertypes.NewMsgRegisterAccount(ibctesting.FirstConnectionID, chainAAddress, version)
		msgRegisterAccount := controllertypes.NewMsgRegisterAccount(ibctesting.FirstConnectionID, controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), version)
		msgs := []sdk.Msg{msgRegisterAccount}

		//msgSubmitProposal, err := govtypesv1.NewMsgSubmitProposal(msgs, sdk.NewCoins(testvalues.DefaultTransferAmount(chainA.Config().Denom)), chainAAddress, "e2e")
		msgSubmitProposal, err := govtypesv1.NewMsgSubmitProposal(msgs, sdk.NewCoins(testvalues.DefaultTransferAmount(chainA.Config().Denom)), controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), "e2e")
		s.Require().NoError(err)

		resp, err := s.BroadcastMessages(ctx, chainA, chainAWallet, msgSubmitProposal)
		t.Logf("%+v", resp)
		s.AssertValidTxResponse(resp)
		s.Require().NoError(err)
	})
}

//package interchain_accounts
//
//import (
//"context"
//"testing"
//"time"
//
//"github.com/cosmos/ibc-go/e2e/testsuite"
//"github.com/cosmos/ibc-go/e2e/testvalues"
//"github.com/strangelove-ventures/ibctest/ibc"
//"github.com/stretchr/testify/suite"
//
//sdk "github.com/cosmos/cosmos-sdk/types"
//grouptypes "github.com/cosmos/cosmos-sdk/x/group"
//
//controllertypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
//icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
//ibctesting "github.com/cosmos/ibc-go/v5/testing"
//)
//
//func TestInterchainAccountsGroupsTestSuite(t *testing.T) {
//	suite.Run(t, new(InterchainAccountsGroupsTestSuite))
//}
//
//type InterchainAccountsGroupsTestSuite struct {
//	testsuite.E2ETestSuite
//}
//
//func (s *InterchainAccountsGroupsTestSuite) QueryGroupPolicyAddress(ctx context.Context, chain ibc.Chain) string {
//	queryClient := s.GetChainGRCPClients(chain).GroupsQueryClient
//	res, err := queryClient.GroupPoliciesByGroup(ctx, &grouptypes.QueryGroupPoliciesByGroupRequest{
//		GroupId: 1, // always use the initial group id
//	})
//	s.Require().NoError(err)
//
//	return res.GroupPolicies[0].Address
//}
//
//// TestInterchainAccountsGroupsIntegration_Success runs a full integration test between the x/group module and ICS27 interchain accounts.
//// 1. Create a group
//// 2. Create a group policy
//// 3. Query group policy address
//// 4. Create group proposal: MsgRegisterAccount
//// 5. Vote on proposal
//// 6. Exec propsoal
//// 7. Query interchain account address
//// 8. Fund the interchain account on chainB
//// 9. Create group proposal: MsgSubmitTx
//func (s *InterchainAccountsGroupsTestSuite) TestInterchainAccountsGroupsIntegration_Success() {
//	t := s.T()
//	ctx := context.TODO()
//
//	// setup relayers and connection-0 between two chains
//	// channel-0 is a transfer channel but it will not be used in this test case
//	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
//	chainA, chainB := s.GetChains()
//
//	_ = relayer
//	_ = chainB
//
//	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
//	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)
//
//	t.Run("create and configure group and group policy", func(t *testing.T) {
//		members := []grouptypes.MemberRequest{
//			{
//				Address: chainAAddress,
//				Weight:  "1",
//			},
//		}
//
//		decisionPolicy := grouptypes.NewThresholdDecisionPolicy("1", time.Duration(time.Minute), time.Duration(0))
//		msgCreateGroupWithPolicy, err := grouptypes.NewMsgCreateGroupWithPolicy(chainAAddress, members, "ics27-controller-group", "ics27-controller-policy", true, decisionPolicy)
//		s.Require().NoError(err)
//
//		txResp, err := s.BroadcastMessages(ctx, chainA, chainAWallet, msgCreateGroupWithPolicy)
//		s.Require().NoError(err)
//		s.AssertValidTxResponse(txResp)
//
//		groupPolicyAddr := s.QueryGroupPolicyAddress(ctx, chainA)
//		msgRegisterAccount := controllertypes.NewMsgRegisterAccount(ibctesting.FirstConnectionID, groupPolicyAddr, icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID))
//
//		msgSubmitProposal, err := grouptypes.NewMsgSubmitProposal(groupPolicyAddr, []string{chainAAddress}, []sdk.Msg{msgRegisterAccount}, "", grouptypes.Exec_EXEC_TRY)
//		s.Require().NoError(err)
//
//		s.BroadcastMessages(ctx, chainA, chainAWallet, msgSubmitProposal)
//
//		interchainAccAddr, err := s.QueryInterchainAccount(ctx, chainA, groupPolicyAddr, ibctesting.FirstConnectionID)
//		s.Require().NoError(err)
//
//		t.Logf("successfully registered interchain account via controller group: %s", interchainAccAddr)
//	})
//
//	// setup 2 accounts: controller account on chain A, a second chain B account.
//	// host account will be created when the ICA is registered
//	// controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
//	// chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
//	// var hostAccount string
//
//	// t.Run("start relayer", func(t *testing.T) {
//	// 	s.StartRelayer(relayer)
//	// })
//
//}
