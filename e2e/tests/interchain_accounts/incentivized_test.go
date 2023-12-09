//go:build !test_e2e

package interchainaccounts

import (
	"context"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	controllertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	feetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func TestIncentivizeInterchainAccountsTestSuite(t *testing.T) {
	testifysuite.Run(t, new(IncentivizeInterchainAccountsTestSuite))
}

type IncentivizeInterchainAccountsTestSuite struct {
	testsuite.E2ETestSuite
	chainA ibc.Chain
	chainB ibc.Chain
	rly    ibc.Relayer
}

func (s *IncentivizeInterchainAccountsTestSuite) SetupTest() {
	ctx := context.TODO()
	s.chainA, s.chainB = s.GetChains()
	s.rly = s.SetupRelayer(ctx, nil, s.chainA, s.chainB)
}

func (s *IncentivizeInterchainAccountsTestSuite) TestMsgSendTx_SuccessfulBankSend_Incentivized() {
	t := s.T()
	ctx := context.TODO()

	var (
		chainADenom   = s.chainA.Config().Denom
		interchainAcc = ""
		testFee       = testvalues.DefaultFee(chainADenom)
	)

	// t.Run("relayer wallets recovered", func(t *testing.T) {
	// 	err := s.RecoverRelayerWallets(ctx, s.rly, s.chainA, s.chainB)
	// 	s.Require().NoError(err)
	// })

	chainARelayerWallet, chainBRelayerWallet, err := s.GetRelayerWallets(s.rly, s.chainA, s.chainB)
	t.Run("relayer wallets fetched", func(t *testing.T) {
		s.Require().NoError(err)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, s.chainA, s.chainB), "failed to wait for blocks")

	chainARelayerUser, chainBRelayerUser := s.GetRelayerUsers(ctx, s.chainA, s.chainB)
	relayerAStartingBalance, err := s.GetChainANativeBalance(ctx, chainARelayerUser)
	s.Require().NoError(err)
	t.Logf("relayer A user starting with balance: %d", relayerAStartingBalance)

	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, s.chainA)
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount, s.chainB)

	t.Run("broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		version := "" // allow version to be specified by the controller chain since both chains should support incentivized channels
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAccount.FormattedAddress(), version)

		txResp := s.BroadcastMessages(ctx, s.chainA, controllerAccount, s.chainA, msgRegisterAccount)
		s.AssertTxSuccess(txResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(s.rly)
	})

	var channelOutput ibc.ChannelOutput
	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		interchainAcc, err = s.QueryInterchainAccount(ctx, s.chainA, controllerAccount.FormattedAddress(), ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(interchainAcc))

		channels, err := s.rly.GetChannels(ctx, s.GetRelayerExecReporter(), s.chainA.Config().ChainID)
		chanNumber++
		s.Require().NoError(err)
		s.Require().Equal(len(channels), chanNumber)

		// interchain accounts channel at index: 0
		channelOutput = channels[0]

		s.Require().NoError(test.WaitForBlocks(ctx, 2, s.chainA, s.chainB))
	})

	t.Run("execute interchain account bank send through controller", func(t *testing.T) {
		t.Run("fund interchain account wallet on host chainB", func(t *testing.T) {
			// fund the interchain account so it has some $$ to send
			err := s.chainB.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
				Address: interchainAcc,
				Amount:  sdkmath.NewInt(testvalues.StartingTokenAmount),
				Denom:   s.chainB.Config().Denom,
			})
			s.Require().NoError(err)
		})

		t.Run("register counterparty payee", func(t *testing.T) {
			resp := s.RegisterCounterPartyPayee(ctx, s.chainB, chainBRelayerUser, channelOutput.Counterparty.PortID, channelOutput.Counterparty.ChannelID, chainBRelayerWallet.FormattedAddress(), chainARelayerWallet.FormattedAddress(), s.chainA)
			s.AssertTxSuccess(resp)
		})

		t.Run("verify counterparty payee", func(t *testing.T) {
			address, err := s.QueryCounterPartyPayee(ctx, s.chainB, chainBRelayerWallet.FormattedAddress(), channelOutput.Counterparty.ChannelID)
			s.Require().NoError(err)
			s.Require().Equal(chainARelayerWallet.FormattedAddress(), address)
		})

		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, s.chainA, channelOutput.PortID, channelOutput.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		t.Run("stop relayer", func(t *testing.T) {
			s.StopRelayer(ctx, s.rly)
		})

		t.Run("broadcast incentivized MsgSendTx", func(t *testing.T) {
			msgPayPacketFee := &feetypes.MsgPayPacketFee{
				Fee:             testvalues.DefaultFee(chainADenom),
				SourcePortId:    channelOutput.PortID,
				SourceChannelId: channelOutput.ChannelID,
				Signer:          controllerAccount.FormattedAddress(),
			}

			msgSend := &banktypes.MsgSend{
				FromAddress: interchainAcc,
				ToAddress:   chainBAccount.FormattedAddress(),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(s.chainB.Config().Denom)),
			}

			cdc := testsuite.Codec()
			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(controllerAccount.FormattedAddress(), ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

			resp := s.BroadcastMessages(ctx, s.chainA, controllerAccount, s.chainB, msgPayPacketFee, msgSendTx)
			s.AssertTxSuccess(resp)

			s.Require().NoError(test.WaitForBlocks(ctx, 1, s.chainA, s.chainB))
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, s.chainA, channelOutput.PortID, channelOutput.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.Equal(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.Equal(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.Equal(testFee.TimeoutFee))
		})

		t.Run("start relayer", func(t *testing.T) {
			s.StartRelayer(s.rly)
		})

		t.Run("packets are relayed", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, s.chainA, channelOutput.PortID, channelOutput.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		t.Run("verify interchain account sent tokens", func(t *testing.T) {
			balance, err := s.QueryBalance(ctx, s.chainB, chainBAccount.FormattedAddress(), s.chainB.Config().Denom)
			s.Require().NoError(err)

			_, err = s.QueryBalance(ctx, s.chainB, interchainAcc, s.chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.IBCTransferAmount + testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance.Int64())
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

func (s *IncentivizeInterchainAccountsTestSuite) TestMsgSendTx_FailedBankSend_Incentivized() {
	t := s.T()
	ctx := context.TODO()

	var (
		chainADenom   = s.chainA.Config().Denom
		interchainAcc = ""
		testFee       = testvalues.DefaultFee(chainADenom)
	)

	t.Run("relayer wallets recovered", func(t *testing.T) {
		err := s.RecoverRelayerWallets(ctx, s.rly, s.chainA, s.chainB)
		s.Require().NoError(err)
	})

	chainARelayerWallet, chainBRelayerWallet, err := s.GetRelayerWallets(s.rly, s.chainA, s.chainB)
	t.Run("relayer wallets fetched", func(t *testing.T) {
		s.Require().NoError(err)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 5, s.chainA, s.chainB), "failed to wait for blocks")

	chainARelayerUser, chainBRelayerUser := s.GetRelayerUsers(ctx, s.chainA, s.chainB)
	relayerAStartingBalance, err := s.GetChainANativeBalance(ctx, chainARelayerUser)
	s.Require().NoError(err)
	t.Logf("relayer A user starting with balance: %d", relayerAStartingBalance)

	controllerAccount := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, s.chainA)
	chainBAccount := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount, s.chainB)

	t.Run("broadcast MsgRegisterInterchainAccount", func(t *testing.T) {
		version := "" // allow version to be specified by the controller chain since both chains should support incentivized channels
		msgRegisterAccount := controllertypes.NewMsgRegisterInterchainAccount(ibctesting.FirstConnectionID, controllerAccount.FormattedAddress(), version)

		txResp := s.BroadcastMessages(ctx, s.chainA, controllerAccount, s.chainB, msgRegisterAccount)
		s.AssertTxSuccess(txResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(s.rly)
	})

	var channelOutput ibc.ChannelOutput
	t.Run("verify interchain account", func(t *testing.T) {
		var err error
		interchainAcc, err = s.QueryInterchainAccount(ctx, s.chainA, controllerAccount.FormattedAddress(), ibctesting.FirstConnectionID)
		s.Require().NoError(err)
		s.Require().NotZero(len(interchainAcc))

		channels, err := s.rly.GetChannels(ctx, s.GetRelayerExecReporter(), s.chainA.Config().ChainID)
		chanNumber++
		s.Require().NoError(err)
		s.Require().Equal(len(channels), chanNumber)

		// interchain accounts channel at index: 0
		channelOutput = channels[0]

		s.Require().NoError(test.WaitForBlocks(ctx, 2, s.chainA, s.chainB))
	})

	t.Run("execute interchain account bank send through controller", func(t *testing.T) {
		t.Run("register counterparty payee", func(t *testing.T) {
			resp := s.RegisterCounterPartyPayee(ctx, s.chainB, chainBRelayerUser, channelOutput.Counterparty.PortID, channelOutput.Counterparty.ChannelID, chainBRelayerWallet.FormattedAddress(), chainARelayerWallet.FormattedAddress(), s.chainA)
			s.AssertTxSuccess(resp)
		})

		t.Run("verify counterparty payee", func(t *testing.T) {
			address, err := s.QueryCounterPartyPayee(ctx, s.chainB, chainBRelayerWallet.FormattedAddress(), channelOutput.Counterparty.ChannelID)
			s.Require().NoError(err)
			s.Require().Equal(chainARelayerWallet.FormattedAddress(), address)
		})

		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, s.chainA, channelOutput.PortID, channelOutput.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		t.Run("stop relayer", func(t *testing.T) {
			err := s.rly.StopRelayer(ctx, s.GetRelayerExecReporter())
			s.Require().NoError(err)
		})

		t.Run("broadcast incentivized MsgSendTx", func(t *testing.T) {
			msgPayPacketFee := &feetypes.MsgPayPacketFee{
				Fee:             testvalues.DefaultFee(chainADenom),
				SourcePortId:    channelOutput.PortID,
				SourceChannelId: channelOutput.ChannelID,
				Signer:          controllerAccount.FormattedAddress(),
			}

			msgSend := &banktypes.MsgSend{
				FromAddress: interchainAcc,
				ToAddress:   chainBAccount.FormattedAddress(),
				Amount:      sdk.NewCoins(testvalues.DefaultTransferAmount(s.chainB.Config().Denom)),
			}

			cdc := testsuite.Codec()
			bz, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msgSend}, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			packetData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: bz,
				Memo: "e2e",
			}

			msgSendTx := controllertypes.NewMsgSendTx(controllerAccount.FormattedAddress(), ibctesting.FirstConnectionID, uint64(time.Hour.Nanoseconds()), packetData)

			resp := s.BroadcastMessages(ctx, s.chainA, controllerAccount, s.chainB, msgPayPacketFee, msgSendTx)
			s.AssertTxSuccess(resp)

			s.Require().NoError(test.WaitForBlocks(ctx, 1, s.chainA, s.chainB))
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, s.chainA, channelOutput.PortID, channelOutput.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.Equal(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.Equal(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.Equal(testFee.TimeoutFee))
		})

		t.Run("start relayer", func(t *testing.T) {
			s.StartRelayer(s.rly)
		})

		t.Run("packets are relayed", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, s.chainA, channelOutput.PortID, channelOutput.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		t.Run("verify interchain account did not send tokens", func(t *testing.T) {
			balance, err := s.QueryBalance(ctx, s.chainB, chainBAccount.FormattedAddress(), s.chainB.Config().Denom)
			s.Require().NoError(err)

			_, err = s.QueryBalance(ctx, s.chainB, interchainAcc, s.chainB.Config().Denom)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount
			s.Require().Equal(expected, balance.Int64(), "tokens should not have been sent as interchain account was not funded")
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
