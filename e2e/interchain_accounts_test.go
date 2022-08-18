package e2e

import (
	"context"
	"testing"

	ibctest "github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"

	"github.com/cosmos/ibc-go/e2e/testconfig"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"

	feetypes "github.com/cosmos/ibc-go/v5/modules/apps/29-fee/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
)

func TestInterchainAccountsTestSuite(t *testing.T) {
	// NOTE: this is a temporary mechanism to enable this test to run alongside the simd tests.
	// This will be removed in a follow up PR and properly parameterized in a github workflow.
	testconfig.SetChainBinaryVersions(
		"ghcr.io/cosmos/ibc-go-icad", "master", "icad", "ghcr.io/cosmos/ibc-go-icad", "master",
	)
	suite.Run(t, new(InterchainAccountsTestSuite))
}

type InterchainAccountsTestSuite struct {
	testsuite.E2ETestSuite
}

// RegisterInterchainAccount will attempt to register an interchain account on the counterparty chain.
func (s *InterchainAccountsTestSuite) RegisterInterchainAccount(ctx context.Context, chain *cosmos.CosmosChain, user *ibctest.User, msgRegisterAccount *intertxtypes.MsgRegisterAccount) error {
	txResp, err := s.BroadcastMessages(ctx, chain, user, msgRegisterAccount)
	s.AssertValidTxResponse(txResp)
	return err
}

// RegisterCounterPartyPayee broadcasts a MsgRegisterCounterpartyPayee message.
func (s *InterchainAccountsTestSuite) RegisterCounterPartyPayee(ctx context.Context, chain *cosmos.CosmosChain,
	user *ibctest.User, portID, channelID, relayerAddr, counterpartyPayeeAddr string) (sdk.TxResponse, error) {
	msg := feetypes.NewMsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayeeAddr)
	return s.BroadcastMessages(ctx, chain, user, msg)
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
		version := "" // allow app to handle the version as appropriate.
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
		version := "" // allow app to handle the version as appropriate.
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

func (s *InterchainAccountsTestSuite) TestMsgSubmitTx_SuccessfulBankSend_Incentivized() {
	t := s.T()
	ctx := context.TODO()

	// setup relayers and connection-0 between two chains
	// channel-0 is a transfer channel but it will not be used in this test case
	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	var (
		chainADenom   = chainA.Config().Denom
		interchainAcc = ""
		testFee       = testvalues.DefaultFee(chainADenom)
	)

	t.Run("relayer wallets recovered", func(t *testing.T) {
		err := s.RecoverRelayerWallets(ctx, relayer)
		s.Require().NoError(err)
	})

	chainARelayerWallet, chainBRelayerWallet, err := s.GetRelayerWallets(relayer)
	t.Run("relayer wallets fetched", func(t *testing.T) {
		s.Require().NoError(err)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

	chainARelayerUser, chainBRelayerUser := s.GetRelayerUsers(ctx)
	relayerAStartingBalance, err := s.GetChainANativeBalance(ctx, chainARelayerUser)
	s.Require().NoError(err)
	t.Logf("relayer A user starting with balance: %d", relayerAStartingBalance)

	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	t.Run("register interchain account", func(t *testing.T) {
		version := "" // allow app to handle the version as appropriate.
		msgRegisterAccount := intertxtypes.NewMsgRegisterAccount(controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID, version)
		err := s.RegisterInterchainAccount(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.Require().NoError(err)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	var channelOutput ibc.ChannelOutput
	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		interchainAcc, err = s.QueryInterchainAccount(ctx, chainA, controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(interchainAcc))

		channels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainA.Config().ChainID)
		s.Require().NoError(err)
		s.Require().Equal(len(channels), 2)

		// interchain accounts channel at index: 0
		channelOutput = channels[0]
	})

	t.Run("execute interchain account bank send through controller", func(t *testing.T) {
		t.Run("fund interchain account wallet on host chainB", func(t *testing.T) {
			// fund the interchain account so it has some $$ to send
			err := chainB.SendFunds(ctx, ibctest.FaucetAccountKeyName, ibc.WalletAmount{
				Address: interchainAcc,
				Amount:  testvalues.StartingTokenAmount,
				Denom:   chainB.Config().Denom,
			})
			s.Require().NoError(err)
		})

		t.Run("register counterparty payee", func(t *testing.T) {
			resp, err := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelOutput.Counterparty.PortID, channelOutput.Counterparty.ChannelID, chainBRelayerWallet.Address, chainARelayerWallet.Address)
			s.Require().NoError(err)
			s.AssertValidTxResponse(resp)
		})

		t.Run("verify counterparty payee", func(t *testing.T) {
			address, err := s.QueryCounterPartyPayee(ctx, chainB, chainBRelayerWallet.Address, channelOutput.Counterparty.ChannelID)
			s.Require().NoError(err)
			s.Require().Equal(chainARelayerWallet.Address, address)
		})

		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelOutput.PortID, channelOutput.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		t.Run("stop relayer", func(t *testing.T) {
			err := relayer.StopRelayer(ctx, s.GetRelayerExecReporter())
			s.Require().NoError(err)
		})

		t.Run("broadcast incentivized MsgSubmitTx", func(t *testing.T) {
			msgPayPacketFee := &feetypes.MsgPayPacketFee{
				Fee:             testvalues.DefaultFee(chainADenom),
				SourcePortId:    channelOutput.PortID,
				SourceChannelId: channelOutput.ChannelID,
				Signer:          controllerAccount.Bech32Address(chainA.Config().Bech32Prefix),
			}

			msgSend := &banktypes.MsgSend{
				FromAddress: interchainAcc,
				ToAddress:   chainBAccount.Bech32Address(chainB.Config().Bech32Prefix),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(chainB.Config().Denom)),
			}

			msgSubmitTx, err := intertxtypes.NewMsgSubmitTx(msgSend, ibctesting.FirstConnectionID, controllerAccount.Bech32Address(chainA.Config().Bech32Prefix))
			s.Require().NoError(err)

			resp, err := s.BroadcastMessages(ctx, chainA, controllerAccount, msgPayPacketFee, msgSubmitTx)
			s.AssertValidTxResponse(resp)
			s.Require().NoError(err)

			s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB))
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelOutput.PortID, channelOutput.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.IsEqual(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.IsEqual(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.IsEqual(testFee.TimeoutFee))
		})

		t.Run("start relayer", func(t *testing.T) {
			s.StartRelayer(relayer)
		})

		t.Run("packets are relayed", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelOutput.PortID, channelOutput.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		t.Run("verify interchain account sent tokens", func(t *testing.T) {
			balance, err := chainB.GetBalance(ctx, chainBAccount.Bech32Address(chainB.Config().Bech32Prefix), chainB.Config().Denom)
			s.Require().NoError(err)

			_, err = chainB.GetBalance(ctx, interchainAcc, chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance)
		})

		t.Run("timeout fee is refunded", func(t *testing.T) {
			actualBalance, err := s.GetChainANativeBalance(ctx, controllerAccount)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - testFee.AckFee.AmountOf(chainADenom).Int64() - testFee.RecvFee.AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})

		t.Run("relayerA is paid ack and recv fee", func(t *testing.T) {
			actualBalance, err := s.GetChainANativeBalance(ctx, chainARelayerUser)
			s.Require().NoError(err)

			expected := relayerAStartingBalance + testFee.AckFee.AmountOf(chainADenom).Int64() + testFee.RecvFee.AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})
	})
}
