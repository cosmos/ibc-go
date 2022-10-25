package interchain_accounts

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	ibctest "github.com/strangelove-ventures/ibctest/v6"
	"github.com/strangelove-ventures/ibctest/v6/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/strangelove-ventures/ibctest/v6/test"
	"github.com/stretchr/testify/suite"
	"golang.org/x/mod/semver"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"

	"github.com/cosmos/ibc-go/e2e/testconfig"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	feetypes "github.com/cosmos/ibc-go/v6/modules/apps/29-fee/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
	simappparams "github.com/cosmos/ibc-go/v6/testing/simapp/params"
)

const (
	InitialProposalID uint64 = 1
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
	user *ibc.Wallet, portID, channelID, relayerAddr, counterpartyPayeeAddr string,
) (sdk.TxResponse, error) {
	msg := feetypes.NewMsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayeeAddr)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

// QueryModuleAccountAddress returns the sdk.AccAddress of a given module name.
func (s *InterchainAccountsTestSuite) QueryModuleAccountAddress(ctx context.Context, moduleName string, chain *cosmos.CosmosChain) (sdk.AccAddress, error) {
	authClient := s.GetChainGRCPClients(chain).AuthQueryClient

	moduleAccountsResponse, err := authClient.ModuleAccounts(ctx, &authtypes.QueryModuleAccountsRequest{})
	if err != nil {
		return nil, err
	}

	// TODO: add this to test suite with all types registered
	cfg := simappparams.MakeTestEncodingConfig()
	authtypes.RegisterInterfaces(cfg.InterfaceRegistry)

	for _, acc := range moduleAccountsResponse.Accounts {
		var account authtypes.AccountI
		err := cfg.InterfaceRegistry.UnpackAny(acc, &account)
		if err != nil {
			return nil, err
		}
		moduleAccount, ok := account.(authtypes.ModuleAccountI)
		if !ok {
			return nil, errors.New(fmt.Sprintf("failed to cast account: %T as ModuleAccount", moduleAccount))
		}

		if moduleAccount.GetName() == moduleName {
			return moduleAccount.GetAddress(), nil
		}
	}

	return nil, errors.New(fmt.Sprintf("failed to find address for module account: %s", moduleName))
}

// getICAVersion returns the version which should be used in the MsgRegisterAccount broadcast from the
// controller chain.
func getICAVersion(chainAVersion, chainBVersion string) string {
	chainBIsGreaterThanOrEqualToChainA := semver.Compare(chainAVersion, chainBVersion) <= 0
	if chainBIsGreaterThanOrEqualToChainA {
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

func (s *InterchainAccountsTestSuite) TestICARegistration_WithGovernance() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()
	chainAAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAAccount.Bech32Address(chainA.Config().Bech32Prefix)

	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	_ = chainB
	_ = chainBAccount

	govModuleAddress, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
	s.Require().NoError(err)
	s.Require().NotNil(govModuleAddress)

	t.Run("create and msg submit proposal", func(t *testing.T) {
		version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, govModuleAddress.String(), version)
		msgs := []sdk.Msg{msgRegisterAccount}
		msgSubmitProposal, err := govtypesv1.NewMsgSubmitProposal(msgs, sdk.NewCoins(sdk.NewCoin(chainA.Config().Denom, govtypesv1.DefaultMinDepositTokens)), chainAAddress, "")
		s.Require().NoError(err)

		resp, err := s.BroadcastMessages(ctx, chainA, chainAAccount, msgSubmitProposal)
		t.Logf("CREATE SUBMIT PROPOSAL: %+v", resp)
		s.AssertValidTxResponse(resp)
		s.Require().NoError(err)
	})

	//s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))
	//
	//t.Run("vote on proposal", func(t *testing.T) {
	//	msgVote := &govtypesv1.MsgVote{
	//		ProposalId: InitialProposalID,
	//		Voter:      chainAAddress,
	//		Option:     govtypesv1.VoteOption_VOTE_OPTION_YES,
	//	}
	//
	//	txResp, err := s.BroadcastMessages(ctx, chainA, chainAAccount, msgVote)
	//	s.Require().NoError(err)
	//	s.AssertValidTxResponse(txResp)
	//})
	//
	s.Require().NoError(chainA.VoteOnProposalAllValidators(ctx, "1", cosmos.ProposalVoteYes))

	time.Sleep(testvalues.VotingPeriod)
	time.Sleep(5 * time.Second)

	proposal, err := s.QueryProposalV1(ctx, chainA, 1)
	s.Require().NoError(err)
	s.Require().Equal(govtypesv1.StatusPassed, proposal.Status)

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA))

	t.Logf("gov module address: %s", govModuleAddress.String())

	//time.Sleep(100 * time.Hour)

	var hostAccount string
	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		hostAccount, err = s.QueryInterchainAccount(ctx, chainA, govModuleAddress.String(), ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(hostAccount))

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)
	})


	//
	//t.Run("interchain account executes a bank transfer on behalf of the corresponding owner account", func(t *testing.T) {
	//	t.Run("fund interchain account wallet", func(t *testing.T) {
	//		// fund the host account, so it has some $$ to send
	//		err := chainB.SendFunds(ctx, ibctest.FaucetAccountKeyName, ibc.WalletAmount{
	//			Address: hostAccount,
	//			Amount:  testvalues.StartingTokenAmount,
	//			Denom:   chainB.Config().Denom,
	//		})
	//		s.Require().NoError(err)
	//	})
	//
	//	t.Run("broadcast MsgSubmitTx", func(t *testing.T) {
	//		// assemble bank transfer message from host account to user account on host chain
	//		msgSend := &banktypes.MsgSend{
	//			FromAddress: hostAccount,
	//			ToAddress:   chainBAccount.Bech32Address(chainB.Config().Bech32Prefix),
	//			Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
	//		}
	//
	//		// assemble submitMessage tx for intertx
	//		msgSubmitTx, err := intertxtypes.NewMsgSubmitTx(
	//			msgSend,
	//			ibctesting.FirstConnectionID,
	//			govModuleAddress.String(),
	//		)
	//		s.Require().NoError(err)
	//
	//		// broadcast submitMessage tx from controller account on chain A
	//		// this message should trigger the sending of an ICA packet over channel-1 (channel created between controller and host)
	//		// this ICA packet contains the assembled bank transfer message from above, which will be executed by the host account on the host chain.
	//		resp, err := s.BroadcastMessages(
	//			ctx,
	//			chainA,
	//			controllerAccount,
	//			msgSubmitTx,
	//		)
	//
	//		s.AssertValidTxResponse(resp)
	//		s.Require().NoError(err)
	//
	//		s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))
	//	})
	//
	//	t.Run("verify tokens transferred", func(t *testing.T) {
	//		balance, err := chainB.GetBalance(ctx, chainBAccount.Bech32Address(chainB.Config().Bech32Prefix), chainB.Config().Denom)
	//		s.Require().NoError(err)
	//
	//		_, err = chainB.GetBalance(ctx, hostAccount, chainB.Config().Denom)
	//		s.Require().NoError(err)
	//
	//		expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
	//		s.Require().Equal(expected, balance)
	//	})
	//})

}
