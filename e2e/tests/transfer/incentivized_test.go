//go:build !test_e2e

package transfer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	feetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

const (
	legacyMessage           = "balance should be lowered by sum of recv_fee, ack_fee and timeout_fee"
	capitalEfficientMessage = "balance should be lowered by max(recv_fee + ack_fee, timeout_fee)"
)

type IncentivizedTransferTestSuite struct {
	TransferTestSuite
}

func TestIncentivizedTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(IncentivizedTransferTestSuite))
}

func (s *IncentivizedTransferTestSuite) TestMsgPayPacketFee_AsyncSingleSender_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.FeeMiddlewareChannelOptions())
	chainA, chainB := s.GetChains()
	chainAVersion := chainA.Config().Images[0].Version

	var (
		chainADenom        = chainA.Config().Denom
		testFee            = testvalues.DefaultFee(chainADenom)
		chainATx           ibc.Tx
		payPacketFeeTxResp sdk.TxResponse
	)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	t.Run("relayer wallets recovered", func(t *testing.T) {
		err := s.RecoverRelayerWallets(ctx, relayer)
		s.Require().NoError(err)
	})

	chainARelayerWallet, chainBRelayerWallet, err := s.GetRelayerWallets(relayer)
	t.Run("relayer wallets fetched", func(t *testing.T) {
		s.Require().NoError(err)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	_, chainBRelayerUser := s.GetRelayerUsers(ctx)

	t.Run("register counterparty payee", func(t *testing.T) {
		resp := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.FormattedAddress(), chainARelayerWallet.FormattedAddress())
		s.AssertTxSuccess(resp)
	})

	t.Run("verify counterparty payee", func(t *testing.T) {
		address, err := query.CounterPartyPayee(ctx, chainB, chainBRelayerWallet.FormattedAddress(), channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.FormattedAddress(), address)
	})

	walletAmount := ibc.WalletAmount{
		Address: chainAWallet.FormattedAddress(), // destination address
		Denom:   chainADenom,
		Amount:  sdkmath.NewInt(testvalues.IBCTransferAmount),
	}

	t.Run("send IBC transfer", func(t *testing.T) {
		chainATx, err = chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName(), walletAmount, ibc.TransferOptions{})
		s.Require().NoError(err)
		s.Require().NoError(chainATx.Validate(), "chain-a ibc transfer tx is invalid")
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - walletAmount.Amount.Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("pay packet fee", func(t *testing.T) {
		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		packetID := channeltypes.NewPacketID(channelA.PortID, channelA.ChannelID, chainATx.Packet.Sequence)
		packetFee := feetypes.NewPacketFee(testFee, chainAWallet.FormattedAddress(), nil)

		t.Run("should succeed", func(t *testing.T) {
			payPacketFeeTxResp = s.PayPacketFeeAsync(ctx, chainA, chainAWallet, packetID, packetFee)
			s.AssertTxSuccess(payPacketFeeTxResp)
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.Equal(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.Equal(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.Equal(testFee.TimeoutFee))
		})

		msg, escrowTotalFee := getMessageAndFee(testFee, chainAVersion)
		t.Run(msg, func(t *testing.T) {
			actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - walletAmount.Amount.Int64() - escrowTotalFee.AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})

	t.Run("timeout fee is refunded", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := testvalues.StartingTokenAmount - walletAmount.Amount.Int64() - testFee.AckFee.AmountOf(chainADenom).Int64() - testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})
}

func (s *IncentivizedTransferTestSuite) TestMsgPayPacketFee_InvalidReceiverAccount() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.FeeMiddlewareChannelOptions())
	chainA, chainB := s.GetChains()
	chainAVersion := chainA.Config().Images[0].Version

	var (
		chainADenom        = chainA.Config().Denom
		testFee            = testvalues.DefaultFee(chainADenom)
		payPacketFeeTxResp sdk.TxResponse
	)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	t.Run("relayer wallets recovered", func(t *testing.T) {
		err := s.RecoverRelayerWallets(ctx, relayer)
		s.Require().NoError(err)
	})

	chainARelayerWallet, chainBRelayerWallet, err := s.GetRelayerWallets(relayer)
	t.Run("relayer wallets fetched", func(t *testing.T) {
		s.Require().NoError(err)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	_, chainBRelayerUser := s.GetRelayerUsers(ctx)

	t.Run("register counterparty payee", func(t *testing.T) {
		resp := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.FormattedAddress(), chainARelayerWallet.FormattedAddress())
		s.AssertTxSuccess(resp)
	})

	t.Run("verify counterparty payee", func(t *testing.T) {
		address, err := query.CounterPartyPayee(ctx, chainB, chainBRelayerWallet.FormattedAddress(), channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.FormattedAddress(), address)
	})

	transferAmount := testvalues.DefaultTransferAmount(chainADenom)

	t.Run("send IBC transfer", func(t *testing.T) {
		version, err := feetypes.MetadataFromVersion(channelA.Version)
		s.Require().NoError(err)

		msgTransfer := testsuite.GetMsgTransfer(
			channelA.PortID,
			channelA.ChannelID,
			version.AppVersion,
			sdk.NewCoins(transferAmount),
			chainAWallet.FormattedAddress(),
			testvalues.InvalidAddress,
			s.GetTimeoutHeight(ctx, chainB),
			0,
			"",
		)
		txResp := s.BroadcastMessages(ctx, chainA, chainAWallet, msgTransfer)
		// this message should be successful, as receiver account is not validated on the sending chain.
		s.AssertTxSuccess(txResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - transferAmount.Amount.Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("pay packet fee", func(t *testing.T) {
		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		packetID := channeltypes.NewPacketID(channelA.PortID, channelA.ChannelID, 1)
		packetFee := feetypes.NewPacketFee(testFee, chainAWallet.FormattedAddress(), nil)

		t.Run("should succeed", func(t *testing.T) {
			payPacketFeeTxResp = s.PayPacketFeeAsync(ctx, chainA, chainAWallet, packetID, packetFee)
			s.AssertTxSuccess(payPacketFeeTxResp)
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.Equal(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.Equal(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.Equal(testFee.TimeoutFee))
		})

		msg, escrowTotalFee := getMessageAndFee(testFee, chainAVersion)
		t.Run(msg, func(t *testing.T) {
			actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - transferAmount.Amount.Int64() - escrowTotalFee.AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})
	t.Run("timeout fee and transfer amount refunded", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		// once the relayer has relayed the packets, the timeout fee should be refunded.
		// the address was invalid so the amount sent should be unescrowed.
		expected := testvalues.StartingTokenAmount - testFee.AckFee.AmountOf(chainADenom).Int64() - testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance, "the amount sent and timeout fee should have been refunded as there was an invalid receiver address provided")
	})
}

func (s *IncentivizedTransferTestSuite) TestMultiMsg_MsgPayPacketFeeSingleSender() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.FeeMiddlewareChannelOptions())
	chainA, chainB := s.GetChains()
	chainAVersion := chainA.Config().Images[0].Version

	var (
		chainADenom        = chainA.Config().Denom
		testFee            = testvalues.DefaultFee(chainADenom)
		multiMsgTxResponse sdk.TxResponse
	)

	transferAmount := testvalues.DefaultTransferAmount(chainA.Config().Denom)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	t.Run("relayer wallets recovered", func(t *testing.T) {
		err := s.RecoverRelayerWallets(ctx, relayer)
		s.Require().NoError(err)
	})

	chainARelayerWallet, chainBRelayerWallet, err := s.GetRelayerWallets(relayer)
	t.Run("relayer wallets fetched", func(t *testing.T) {
		s.Require().NoError(err)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	chainARelayerUser, chainBRelayerUser := s.GetRelayerUsers(ctx)

	relayerAStartingBalance, err := s.GetChainANativeBalance(ctx, chainARelayerUser)
	s.Require().NoError(err)
	t.Logf("relayer A user starting with balance: %d", relayerAStartingBalance)

	t.Run("register counterparty payee", func(t *testing.T) {
		multiMsgTxResponse = s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.FormattedAddress(), chainARelayerWallet.FormattedAddress())
		s.AssertTxSuccess(multiMsgTxResponse)
	})

	t.Run("verify counterparty payee", func(t *testing.T) {
		address, err := query.CounterPartyPayee(ctx, chainB, chainBRelayerWallet.FormattedAddress(), channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.FormattedAddress(), address)
	})

	t.Run("no incentivized packets", func(t *testing.T) {
		packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})

	version, err := feetypes.MetadataFromVersion(channelA.Version)
	s.Require().NoError(err)

	msgPayPacketFee := feetypes.NewMsgPayPacketFee(testFee, channelA.PortID, channelA.ChannelID, chainAWallet.FormattedAddress(), nil)
	msgTransfer := testsuite.GetMsgTransfer(
		channelA.PortID,
		channelA.ChannelID,
		version.AppVersion,
		sdk.NewCoins(transferAmount),
		chainAWallet.FormattedAddress(),
		chainBWallet.FormattedAddress(),
		s.GetTimeoutHeight(ctx, chainB),
		0,
		"",
	)
	resp := s.BroadcastMessages(ctx, chainA, chainAWallet, msgPayPacketFee, msgTransfer)
	s.AssertTxSuccess(resp)

	t.Run("there should be incentivized packets", func(t *testing.T) {
		packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Len(packets, 1)
		actualFee := packets[0].PacketFees[0].Fee

		s.Require().True(actualFee.RecvFee.Equal(testFee.RecvFee))
		s.Require().True(actualFee.AckFee.Equal(testFee.AckFee))
		s.Require().True(actualFee.TimeoutFee.Equal(testFee.TimeoutFee))
	})

	msg, escrowTotalFee := getMessageAndFee(testFee, chainAVersion)
	t.Run(msg, func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount - escrowTotalFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})

	t.Run("timeout fee is refunded", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount - testFee.AckFee.AmountOf(chainADenom).Int64() - testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("relayerA is paid ack and recv fee", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainARelayerUser)
		s.Require().NoError(err)
		expected := relayerAStartingBalance + testFee.AckFee.AmountOf(chainADenom).Int64() + testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})
}

func (s *IncentivizedTransferTestSuite) TestMsgPayPacketFee_SingleSender_TimesOut() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.FeeMiddlewareChannelOptions())
	chainA, chainB := s.GetChains()
	chainAVersion := chainA.Config().Images[0].Version

	var (
		chainADenom        = chainA.Config().Denom
		testFee            = testvalues.DefaultFee(chainADenom)
		chainATx           ibc.Tx
		payPacketFeeTxResp sdk.TxResponse
	)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	t.Run("relayer wallets recovered", func(t *testing.T) {
		s.Require().NoError(s.RecoverRelayerWallets(ctx, relayer))
	})

	chainARelayerWallet, chainBRelayerWallet, err := s.GetRelayerWallets(relayer)
	t.Run("relayer wallets fetched", func(t *testing.T) {
		s.Require().NoError(err)
	})

	_, chainBRelayerUser := s.GetRelayerUsers(ctx)

	t.Run("register counterparty payee", func(t *testing.T) {
		resp := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.FormattedAddress(), chainARelayerWallet.FormattedAddress())
		s.AssertTxSuccess(resp)
	})

	t.Run("verify counterparty payee", func(t *testing.T) {
		address, err := query.CounterPartyPayee(ctx, chainB, chainBRelayerWallet.FormattedAddress(), channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.FormattedAddress(), address)
	})

	chainBWalletAmount := ibc.WalletAmount{
		Address: chainBWallet.FormattedAddress(), // destination address
		Denom:   chainA.Config().Denom,
		Amount:  sdkmath.NewInt(testvalues.IBCTransferAmount),
	}

	t.Run("Send IBC transfer", func(t *testing.T) {
		chainATx, err = chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName(), chainBWalletAmount, ibc.TransferOptions{Timeout: testvalues.ImmediatelyTimeout()})
		s.Require().NoError(err)
		s.Require().NoError(chainATx.Validate(), "source ibc transfer tx is invalid")
		time.Sleep(time.Nanosecond * 1) // want it to timeout immediately
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount.Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("pay packet fee", func(t *testing.T) {
		packetID := channeltypes.NewPacketID(channelA.PortID, channelA.ChannelID, chainATx.Packet.Sequence)
		packetFee := feetypes.NewPacketFee(testFee, chainAWallet.FormattedAddress(), nil)

		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		t.Run("should succeed", func(t *testing.T) {
			payPacketFeeTxResp = s.PayPacketFeeAsync(ctx, chainA, chainAWallet, packetID, packetFee)
			s.AssertTxSuccess(payPacketFeeTxResp)
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.Equal(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.Equal(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.Equal(testFee.TimeoutFee))
		})

		msg, escrowTotalFee := getMessageAndFee(testFee, chainAVersion)
		t.Run(msg, func(t *testing.T) {
			actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount.Int64() - escrowTotalFee.AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})

	t.Run("recv and ack should be refunded", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testFee.TimeoutFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})
}

func (s *IncentivizedTransferTestSuite) TestPayPacketFeeAsync_SingleSender_NoCounterPartyAddress() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.FeeMiddlewareChannelOptions())
	chainA, _ := s.GetChains()
	chainAVersion := chainA.Config().Images[0].Version

	var (
		chainADenom        = chainA.Config().Denom
		testFee            = testvalues.DefaultFee(chainADenom)
		chainATx           ibc.Tx
		payPacketFeeTxResp sdk.TxResponse
	)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	t.Run("relayer wallets recovered", func(t *testing.T) {
		err := s.RecoverRelayerWallets(ctx, relayer)
		s.Require().NoError(err)
	})

	chainBWalletAmount := ibc.WalletAmount{
		Address: chainAWallet.FormattedAddress(), // destination address
		Denom:   chainADenom,
		Amount:  sdkmath.NewInt(testvalues.IBCTransferAmount),
	}

	t.Run("send IBC transfer", func(t *testing.T) {
		var err error
		chainATx, err = chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName(), chainBWalletAmount, ibc.TransferOptions{})
		s.Require().NoError(err)
		s.Require().NoError(chainATx.Validate(), "source ibc transfer tx is invalid")
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount.Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("pay packet fee", func(t *testing.T) {
		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		packetID := channeltypes.NewPacketID(channelA.PortID, channelA.ChannelID, chainATx.Packet.Sequence)
		packetFee := feetypes.NewPacketFee(testFee, chainAWallet.FormattedAddress(), nil)

		t.Run("should succeed", func(t *testing.T) {
			payPacketFeeTxResp = s.PayPacketFeeAsync(ctx, chainA, chainAWallet, packetID, packetFee)
			s.AssertTxSuccess(payPacketFeeTxResp)
		})

		t.Run("should be incentivized packets", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.Equal(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.Equal(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.Equal(testFee.TimeoutFee))
		})
	})

	msg, escrowTotalFee := getMessageAndFee(testFee, chainAVersion)
	t.Run(msg, func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount.Int64() - escrowTotalFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("with no counterparty address", func(t *testing.T) {
		t.Run("packets are relayed", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		t.Run("timeout and recv fee are refunded", func(t *testing.T) {
			actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
			s.Require().NoError(err)

			// once the relayer has relayed the packets, the timeout and recv fee should be refunded.
			expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount.Int64() - testFee.AckFee.AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})
	})
}

func (s *IncentivizedTransferTestSuite) TestMsgPayPacketFee_AsyncMultipleSenders_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.FeeMiddlewareChannelOptions())
	chainA, chainB := s.GetChains()
	chainAVersion := chainA.Config().Images[0].Version

	var (
		chainADenom        = chainA.Config().Denom
		testFee            = testvalues.DefaultFee(chainADenom)
		chainATx           ibc.Tx
		payPacketFeeTxResp sdk.TxResponse
	)

	chainAWallet1 := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAWallet2 := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	t.Run("relayer wallets recovered", func(t *testing.T) {
		err := s.RecoverRelayerWallets(ctx, relayer)
		s.Require().NoError(err)
	})

	chainARelayerWallet, chainBRelayerWallet, err := s.GetRelayerWallets(relayer)
	s.Require().NoError(err)
	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	_, chainBRelayerUser := s.GetRelayerUsers(ctx)

	t.Run("register counterparty payee", func(t *testing.T) {
		resp := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.FormattedAddress(), chainARelayerWallet.FormattedAddress())
		s.AssertTxSuccess(resp)
	})

	t.Run("verify counterparty payee", func(t *testing.T) {
		address, err := query.CounterPartyPayee(ctx, chainB, chainBRelayerWallet.FormattedAddress(), channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.FormattedAddress(), address)
	})

	walletAmount1 := ibc.WalletAmount{
		Address: chainAWallet1.FormattedAddress(), // destination address
		Denom:   chainADenom,
		Amount:  sdkmath.NewInt(testvalues.IBCTransferAmount),
	}

	t.Run("send IBC transfer", func(t *testing.T) {
		chainATx, err = chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet1.KeyName(), walletAmount1, ibc.TransferOptions{})
		s.Require().NoError(err)
		s.Require().NoError(chainATx.Validate(), "chain-a ibc transfer tx is invalid")
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet1)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - walletAmount1.Amount.Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("pay packet fee", func(t *testing.T) {
		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		packetID := channeltypes.NewPacketID(channelA.PortID, channelA.ChannelID, chainATx.Packet.Sequence)
		packetFee1 := feetypes.NewPacketFee(testFee, chainAWallet1.FormattedAddress(), nil)
		packetFee2 := feetypes.NewPacketFee(testFee, chainAWallet2.FormattedAddress(), nil)

		t.Run("paying packetFee1 should succeed", func(t *testing.T) {
			payPacketFeeTxResp = s.PayPacketFeeAsync(ctx, chainA, chainAWallet1, packetID, packetFee1)
			s.AssertTxSuccess(payPacketFeeTxResp)
		})
		t.Run("paying packetFee2 should succeed", func(t *testing.T) {
			payPacketFeeTxResp = s.PayPacketFeeAsync(ctx, chainA, chainAWallet2, packetID, packetFee2)
			s.AssertTxSuccess(payPacketFeeTxResp)
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee1 := packets[0].PacketFees[0].Fee
			actualFee2 := packets[0].PacketFees[1].Fee
			s.Require().Len(packets[0].PacketFees, 2)

			s.Require().True(actualFee1.RecvFee.Equal(testFee.RecvFee))
			s.Require().True(actualFee1.AckFee.Equal(testFee.AckFee))
			s.Require().True(actualFee1.TimeoutFee.Equal(testFee.TimeoutFee))

			s.Require().True(actualFee2.RecvFee.Equal(testFee.RecvFee))
			s.Require().True(actualFee2.AckFee.Equal(testFee.AckFee))
			s.Require().True(actualFee2.TimeoutFee.Equal(testFee.TimeoutFee))
		})

		msgFn := func(wallet string) string {
			return fmt.Sprintf("balance of %s should be lowered by max(recv_fee + ack_fee, timeout_fee)", wallet)
		}
		escrowTotalFee := testFee.Total()
		if !testvalues.CapitalEfficientFeeEscrowFeatureReleases.IsSupported(chainAVersion) {
			msgFn = func(wallet string) string {
				return fmt.Sprintf("balance of %s should be lowered by sum of recv_fee, ack_fee and timeout_fee", wallet)
			}
			escrowTotalFee = testFee.RecvFee.Add(testFee.AckFee...).Add(testFee.TimeoutFee...)
		}
		t.Run(msgFn("chainAWallet1"), func(t *testing.T) {
			actualBalance1, err := s.GetChainANativeBalance(ctx, chainAWallet1)
			s.Require().NoError(err)

			expected1 := testvalues.StartingTokenAmount - walletAmount1.Amount.Int64() - escrowTotalFee.AmountOf(chainADenom).Int64()
			s.Require().Equal(expected1, actualBalance1)
		})

		t.Run(msgFn("chainAWallet2"), func(t *testing.T) {
			actualBalance2, err := s.GetChainANativeBalance(ctx, chainAWallet2)
			s.Require().NoError(err)

			expected2 := testvalues.StartingTokenAmount - escrowTotalFee.AmountOf(chainADenom).Int64()
			s.Require().Equal(expected2, actualBalance2)
		})
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := query.IncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})

	t.Run("timeout fee is refunded", func(t *testing.T) {
		actualBalance1, err := s.GetChainANativeBalance(ctx, chainAWallet1)
		s.Require().NoError(err)

		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected1 := testvalues.StartingTokenAmount - walletAmount1.Amount.Int64() - testFee.AckFee.AmountOf(chainADenom).Int64() - testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected1, actualBalance1)

		actualBalance2, err := s.GetChainANativeBalance(ctx, chainAWallet2)
		s.Require().NoError(err)

		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected2 := testvalues.StartingTokenAmount - testFee.AckFee.AmountOf(chainADenom).Int64() - testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected2, actualBalance2)
	})
}

// getMessageAndFee returns the message that should be used for t.Run as well as the expected fee based on
// whether or not capital efficient fee escrow is supported.
func getMessageAndFee(fee feetypes.Fee, chainVersion string) (string, sdk.Coins) {
	msg := capitalEfficientMessage
	escrowTotalFee := fee.Total()
	if !testvalues.CapitalEfficientFeeEscrowFeatureReleases.IsSupported(chainVersion) {
		msg = legacyMessage
		escrowTotalFee = fee.RecvFee.Add(fee.AckFee...).Add(fee.TimeoutFee...)
	}
	return msg, escrowTotalFee
}
