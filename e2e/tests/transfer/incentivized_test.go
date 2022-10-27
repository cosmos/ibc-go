package transfer

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/strangelove-ventures/ibctest/v6/test"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testvalues"
	feetypes "github.com/cosmos/ibc-go/v6/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
)

type IncentivizedTransferTestSuite struct {
	TransferTestSuite
}

func TestIncentivizedTransferTestSuite(t *testing.T) {
	suite.Run(t, new(IncentivizedTransferTestSuite))
}

func (s *IncentivizedTransferTestSuite) TestMsgPayPacketFee_AsyncSingleSender_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
	chainA, chainB := s.GetChains()

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
		resp, err := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.Address, chainARelayerWallet.Address)
		s.Require().NoError(err)
		s.AssertValidTxResponse(resp)
	})

	t.Run("verify counterparty payee", func(t *testing.T) {
		address, err := s.QueryCounterPartyPayee(ctx, chainB, chainBRelayerWallet.Address, channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.Address, address)
	})

	walletAmount := ibc.WalletAmount{
		Address: chainAWallet.Bech32Address(chainB.Config().Bech32Prefix), // destination address
		Denom:   chainADenom,
		Amount:  testvalues.IBCTransferAmount,
	}

	t.Run("send IBC transfer", func(t *testing.T) {
		chainATx, err = chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName, walletAmount, nil)
		s.Require().NoError(err)
		s.Require().NoError(chainATx.Validate(), "chain-a ibc transfer tx is invalid")
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - walletAmount.Amount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("pay packet fee", func(t *testing.T) {
		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		packetId := channeltypes.NewPacketID(channelA.PortID, channelA.ChannelID, chainATx.Packet.Sequence)
		packetFee := feetypes.NewPacketFee(testFee, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix), nil)

		t.Run("should succeed", func(t *testing.T) {
			payPacketFeeTxResp, err = s.PayPacketFeeAsync(ctx, chainA, chainAWallet, packetId, packetFee)
			s.Require().NoError(err)
			s.AssertValidTxResponse(payPacketFeeTxResp)
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.IsEqual(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.IsEqual(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.IsEqual(testFee.TimeoutFee))
		})

		t.Run("balance should be lowered by sum of recv ack and timeout", func(t *testing.T) {
			// The balance should be lowered by the sum of the recv, ack and timeout fees.
			actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - walletAmount.Amount - testFee.Total().AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})

	t.Run("timeout fee is refunded", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := testvalues.StartingTokenAmount - walletAmount.Amount - testFee.AckFee.AmountOf(chainADenom).Int64() - testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})
}

func (s *IncentivizedTransferTestSuite) TestMsgPayPacketFee_InvalidReceiverAccount() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
	chainA, chainB := s.GetChains()

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
		resp, err := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.Address, chainARelayerWallet.Address)
		s.Require().NoError(err)
		s.AssertValidTxResponse(resp)
	})

	t.Run("verify counterparty payee", func(t *testing.T) {
		address, err := s.QueryCounterPartyPayee(ctx, chainB, chainBRelayerWallet.Address, channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.Address, address)
	})

	transferAmount := testvalues.DefaultTransferAmount(chainADenom)

	t.Run("send IBC transfer", func(t *testing.T) {
		transferMsg := transfertypes.NewMsgTransfer(channelA.PortID, channelA.ChannelID, transferAmount, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix), testvalues.InvalidAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		txResp, err := s.BroadcastMessages(ctx, chainA, chainAWallet, transferMsg)
		// this message should be successful, as receiver account is not validated on the sending chain.
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - transferAmount.Amount.Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("pay packet fee", func(t *testing.T) {
		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		packetId := channeltypes.NewPacketID(channelA.PortID, channelA.ChannelID, 1)
		packetFee := feetypes.NewPacketFee(testFee, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix), nil)

		t.Run("should succeed", func(t *testing.T) {
			payPacketFeeTxResp, err = s.PayPacketFeeAsync(ctx, chainA, chainAWallet, packetId, packetFee)
			s.Require().NoError(err)
			s.AssertValidTxResponse(payPacketFeeTxResp)
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.IsEqual(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.IsEqual(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.IsEqual(testFee.TimeoutFee))
		})

		t.Run("balance should be lowered by sum of recv, ack and timeout", func(t *testing.T) {
			// The balance should be lowered by the sum of the recv, ack and timeout fees.
			actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - transferAmount.Amount.Int64() - testFee.Total().AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
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

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())

	chainA, chainB := s.GetChains()

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
		multiMsgTxResponse, err = s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.Address, chainARelayerWallet.Address)
		s.Require().NoError(err)
		s.AssertValidTxResponse(multiMsgTxResponse)
	})

	t.Run("verify counterparty payee", func(t *testing.T) {
		address, err := s.QueryCounterPartyPayee(ctx, chainB, chainBRelayerWallet.Address, channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.Address, address)
	})

	t.Run("no incentivized packets", func(t *testing.T) {
		packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})

	payPacketFeeMsg := feetypes.NewMsgPayPacketFee(testFee, channelA.PortID, channelA.ChannelID, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix), nil)
	transferMsg := transfertypes.NewMsgTransfer(channelA.PortID, channelA.ChannelID, transferAmount, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix), chainBWallet.Bech32Address(chainB.Config().Bech32Prefix), s.GetTimeoutHeight(ctx, chainB), 0, "")
	resp, err := s.BroadcastMessages(ctx, chainA, chainAWallet, payPacketFeeMsg, transferMsg)

	t.Run("transfer successful", func(t *testing.T) {
		s.AssertValidTxResponse(resp)
		s.Require().NoError(err)
	})

	t.Run("there should be incentivized packets", func(t *testing.T) {
		packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Len(packets, 1)
		actualFee := packets[0].PacketFees[0].Fee

		s.Require().True(actualFee.RecvFee.IsEqual(testFee.RecvFee))
		s.Require().True(actualFee.AckFee.IsEqual(testFee.AckFee))
		s.Require().True(actualFee.TimeoutFee.IsEqual(testFee.TimeoutFee))
	})

	t.Run("balance should be lowered by sum of recv ack and timeout", func(t *testing.T) {
		// The balance should be lowered by the sum of the recv, ack and timeout fees.
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount - testFee.Total().AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
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

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
	chainA, chainB := s.GetChains()

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
		resp, err := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.Address, chainARelayerWallet.Address)
		s.Require().NoError(err)
		s.AssertValidTxResponse(resp)
	})

	t.Run("verify counterparty payee", func(t *testing.T) {
		address, err := s.QueryCounterPartyPayee(ctx, chainB, chainBRelayerWallet.Address, channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.Address, address)
	})

	chainBWalletAmount := ibc.WalletAmount{
		Address: chainBWallet.Bech32Address(chainB.Config().Bech32Prefix), // destination address
		Denom:   chainA.Config().Denom,
		Amount:  testvalues.IBCTransferAmount,
	}

	t.Run("Send IBC transfer", func(t *testing.T) {
		chainATx, err = chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName, chainBWalletAmount, testvalues.ImmediatelyTimeout())
		s.Require().NoError(err)
		s.Require().NoError(chainATx.Validate(), "source ibc transfer tx is invalid")
		time.Sleep(time.Nanosecond * 1) // want it to timeout immediately
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("pay packet fee", func(t *testing.T) {
		packetId := channeltypes.NewPacketID(channelA.PortID, channelA.ChannelID, chainATx.Packet.Sequence)
		packetFee := feetypes.NewPacketFee(testFee, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix), nil)

		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		t.Run("should succeed", func(t *testing.T) {
			payPacketFeeTxResp, err = s.PayPacketFeeAsync(ctx, chainA, chainAWallet, packetId, packetFee)
			s.Require().NoError(err)
			s.AssertValidTxResponse(payPacketFeeTxResp)
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.IsEqual(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.IsEqual(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.IsEqual(testFee.TimeoutFee))
		})

		t.Run("balance should be lowered by sum of recv ack and timeout", func(t *testing.T) {
			// The balance should be lowered by the sum of the recv, ack and timeout fees.
			actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount - testFee.Total().AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
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

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
	chainA, chainB := s.GetChains()

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
		Address: chainAWallet.Bech32Address(chainB.Config().Bech32Prefix), // destination address
		Denom:   chainADenom,
		Amount:  testvalues.IBCTransferAmount,
	}

	t.Run("send IBC transfer", func(t *testing.T) {
		var err error
		chainATx, err = chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName, chainBWalletAmount, nil)
		s.Require().NoError(err)
		s.Require().NoError(chainATx.Validate(), "source ibc transfer tx is invalid")
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("pay packet fee", func(t *testing.T) {
		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		packetId := channeltypes.NewPacketID(channelA.PortID, channelA.ChannelID, chainATx.Packet.Sequence)
		packetFee := feetypes.NewPacketFee(testFee, chainAWallet.Bech32Address(chainA.Config().Bech32Prefix), nil)

		t.Run("should succeed", func(t *testing.T) {
			var err error
			payPacketFeeTxResp, err = s.PayPacketFeeAsync(ctx, chainA, chainAWallet, packetId, packetFee)
			s.Require().NoError(err)
			s.AssertValidTxResponse(payPacketFeeTxResp)
		})

		t.Run("should be incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.IsEqual(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.IsEqual(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.IsEqual(testFee.TimeoutFee))
		})
	})

	t.Run("balance should be lowered by sum of recv, ack and timeout", func(t *testing.T) {
		// The balance should be lowered by the sum of the recv, ack and timeout fees.
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount - testFee.Total().AmountOf(chainADenom).Int64()
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("with no counterparty address", func(t *testing.T) {
		t.Run("packets are relayed", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		t.Run("timeout and recv fee are refunded", func(t *testing.T) {
			actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
			s.Require().NoError(err)

			// once the relayer has relayed the packets, the timeout and recv fee should be refunded.
			expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount - testFee.AckFee.AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})
	})
}

func (s *IncentivizedTransferTestSuite) TestMsgPayPacketFee_AsyncMultipleSenders_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
	chainA, chainB := s.GetChains()

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
		resp, err := s.RegisterCounterPartyPayee(ctx, chainB, chainBRelayerUser, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, chainBRelayerWallet.Address, chainARelayerWallet.Address)
		s.Require().NoError(err)
		s.AssertValidTxResponse(resp)
	})

	t.Run("verify counterparty payee", func(t *testing.T) {
		address, err := s.QueryCounterPartyPayee(ctx, chainB, chainBRelayerWallet.Address, channelA.Counterparty.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(chainARelayerWallet.Address, address)
	})

	walletAmount1 := ibc.WalletAmount{
		Address: chainAWallet1.Bech32Address(chainB.Config().Bech32Prefix), // destination address
		Denom:   chainADenom,
		Amount:  testvalues.IBCTransferAmount,
	}

	t.Run("send IBC transfer", func(t *testing.T) {
		chainATx, err = chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet1.KeyName, walletAmount1, nil)
		s.Require().NoError(err)
		s.Require().NoError(chainATx.Validate(), "chain-a ibc transfer tx is invalid")
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet1)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - walletAmount1.Amount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("pay packet fee", func(t *testing.T) {
		t.Run("no incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Empty(packets)
		})

		packetId := channeltypes.NewPacketID(channelA.PortID, channelA.ChannelID, chainATx.Packet.Sequence)
		packetFee1 := feetypes.NewPacketFee(testFee, chainAWallet1.Bech32Address(chainA.Config().Bech32Prefix), nil)
		packetFee2 := feetypes.NewPacketFee(testFee, chainAWallet2.Bech32Address(chainA.Config().Bech32Prefix), nil)

		t.Run("paying packetFee1 should succeed", func(t *testing.T) {
			payPacketFeeTxResp, err = s.PayPacketFeeAsync(ctx, chainA, chainAWallet1, packetId, packetFee1)
			s.Require().NoError(err)
			s.AssertValidTxResponse(payPacketFeeTxResp)
		})
		t.Run("paying packetFee2 should succeed", func(t *testing.T) {
			payPacketFeeTxResp, err = s.PayPacketFeeAsync(ctx, chainA, chainAWallet2, packetId, packetFee2)
			s.Require().NoError(err)
			s.AssertValidTxResponse(payPacketFeeTxResp)
		})

		t.Run("there should be incentivized packets", func(t *testing.T) {
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee1 := packets[0].PacketFees[0].Fee
			actualFee2 := packets[0].PacketFees[1].Fee
			s.Require().Len(packets[0].PacketFees, 2)

			s.Require().True(actualFee1.RecvFee.IsEqual(testFee.RecvFee))
			s.Require().True(actualFee1.AckFee.IsEqual(testFee.AckFee))
			s.Require().True(actualFee1.TimeoutFee.IsEqual(testFee.TimeoutFee))

			s.Require().True(actualFee2.RecvFee.IsEqual(testFee.RecvFee))
			s.Require().True(actualFee2.AckFee.IsEqual(testFee.AckFee))
			s.Require().True(actualFee2.TimeoutFee.IsEqual(testFee.TimeoutFee))
		})

		t.Run("balance of chainAWallet1 should be lowered by sum of recv ack and timeout", func(t *testing.T) {
			actualBalance1, err := s.GetChainANativeBalance(ctx, chainAWallet1)
			s.Require().NoError(err)

			expected1 := testvalues.StartingTokenAmount - walletAmount1.Amount - testFee.Total().AmountOf(chainADenom).Int64()
			s.Require().Equal(expected1, actualBalance1)
		})

		t.Run("balance of chainAWallet2 should be lowered by sum of recv ack and timeout", func(t *testing.T) {
			actualBalance2, err := s.GetChainANativeBalance(ctx, chainAWallet2)
			s.Require().NoError(err)

			expected2 := testvalues.StartingTokenAmount - testFee.Total().AmountOf(chainADenom).Int64()
			s.Require().Equal(expected2, actualBalance2)
		})
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
		s.Require().NoError(err)
		s.Require().Empty(packets)
	})

	t.Run("timeout fee is refunded", func(t *testing.T) {
		actualBalance1, err := s.GetChainANativeBalance(ctx, chainAWallet1)
		s.Require().NoError(err)

		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected1 := testvalues.StartingTokenAmount - walletAmount1.Amount - testFee.AckFee.AmountOf(chainADenom).Int64() - testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected1, actualBalance1)

		actualBalance2, err := s.GetChainANativeBalance(ctx, chainAWallet2)
		s.Require().NoError(err)

		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected2 := testvalues.StartingTokenAmount - testFee.AckFee.AmountOf(chainADenom).Int64() - testFee.RecvFee.AmountOf(chainADenom).Int64()
		s.Require().Equal(expected2, actualBalance2)
	})
}

// feeMiddlewareChannelOptions configures both of the chains to have fee middleware enabled.
func feeMiddlewareChannelOptions() func(options *ibc.CreateChannelOptions) {
	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}"
		opts.DestPortName = "transfer"
		opts.SourcePortName = "transfer"
	}
}
