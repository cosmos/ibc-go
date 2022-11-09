package interchain_accounts

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/gogo/protobuf/proto"
	ibctest "github.com/strangelove-ventures/ibctest/v6"
	"github.com/strangelove-ventures/ibctest/v6/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/strangelove-ventures/ibctest/v6/test"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testconfig"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	feetypes "github.com/cosmos/ibc-go/v6/modules/apps/29-fee/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

func TestIncentivizedInterchainAccountsTestSuite(t *testing.T) {
	suite.Run(t, new(IncentivizedInterchainAccountsTestSuite))
}

type IncentivizedInterchainAccountsTestSuite struct {
	InterchainAccountsTestSuite
}

// RegisterCounterPartyPayee broadcasts a MsgRegisterCounterpartyPayee message.
func (s *IncentivizedInterchainAccountsTestSuite) RegisterCounterPartyPayee(ctx context.Context, chain *cosmos.CosmosChain,
	user *ibc.Wallet, portID, channelID, relayerAddr, counterpartyPayeeAddr string,
) (sdk.TxResponse, error) {
	msg := feetypes.NewMsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayeeAddr)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

func (s *IncentivizedInterchainAccountsTestSuite) TestMsgSendTx_SuccessfulBankSend_Incentivized() {
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

	t.Run("broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		version := getICAVersion(testconfig.GetChainATag(), testconfig.GetChainBTag())
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), version)

		txResp, err := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
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

		s.Require().NoError(test.WaitForBlocks(ctx, 2, chainA, chainB))
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
			s.StopRelayer(ctx, relayer)
		})

		t.Run("broadcast incentivized MsgSendTx", func(t *testing.T) {
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

			cdc := testsuite.Codec()
			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend})
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

			resp, err := s.BroadcastMessages(ctx, chainA, controllerAccount, msgPayPacketFee, msgSendTx)
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

func (s *IncentivizedInterchainAccountsTestSuite) TestMsgSendTx_FailedBankSend_Incentivized() {
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

	t.Run("broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		version := getICAVersion(testconfig.GetChainATag(), testconfig.GetChainBTag())
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), version)

		txResp, err := s.BroadcastMessages(ctx, chainA, controllerAccount, msgRegisterAccount)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
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

		s.Require().NoError(test.WaitForBlocks(ctx, 2, chainA, chainB))
	})

	t.Run("execute interchain account bank send through controller", func(t *testing.T) {
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

		t.Run("broadcast incentivized MsgSendTx", func(t *testing.T) {
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

			cdc := testsuite.Codec()
			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend})
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(controllerAccount.Bech32Address(chainA.Config().Bech32Prefix), ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

			resp, err := s.BroadcastMessages(ctx, chainA, controllerAccount, msgPayPacketFee, msgSendTx)
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

		t.Run("verify interchain account did not send tokens", func(t *testing.T) {
			balance, err := chainB.GetBalance(ctx, chainBAccount.Bech32Address(chainB.Config().Bech32Prefix), chainB.Config().Denom)
			s.Require().NoError(err)

			_, err = chainB.GetBalance(ctx, interchainAcc, chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance, "tokens should not have been sent as interchain account was not funded")
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
