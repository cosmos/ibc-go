//go:build !test_e2e

package transfer

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	feetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

func TestNoCounterPartyIncentivizedTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(NoCounterPartyIncentivizedTransferTestSuite))
}

type NoCounterPartyIncentivizedTransferTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *NoCounterPartyIncentivizedTransferTestSuite) SetupSuite() {
	chainA, chainB := s.GetChains()
	s.SetChainsIntoSuite(chainA, chainB)
}

func (s *NoCounterPartyIncentivizedTransferTestSuite) TestPayPacketFeeAsync_SingleSender_NoCounterPartyAddress() {
	t := s.T()
	t.Parallel()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	relayer, channelA := s.SetupRelayer(ctx, feeMiddlewareChannelOptions(), chainA, chainB)

	var (
		chainADenom        = chainA.Config().Denom
		testFee            = testvalues.DefaultFee(chainADenom)
		chainATx           ibc.Tx
		payPacketFeeTxResp sdk.TxResponse
	)

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

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
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
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
			packets, err := s.QueryIncentivizedPacketsForChannel(ctx, chainA, channelA.PortID, channelA.ChannelID)
			s.Require().NoError(err)
			s.Require().Len(packets, 1)
			actualFee := packets[0].PacketFees[0].Fee

			s.Require().True(actualFee.RecvFee.Equal(testFee.RecvFee))
			s.Require().True(actualFee.AckFee.Equal(testFee.AckFee))
			s.Require().True(actualFee.TimeoutFee.Equal(testFee.TimeoutFee))
		})
	})

	t.Run("balance should be lowered by sum of recv, ack and timeout", func(t *testing.T) {
		// The balance should be lowered by the sum of the recv, ack and timeout fees.
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount.Int64() - testFee.Total().AmountOf(chainADenom).Int64()
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
			expected := testvalues.StartingTokenAmount - chainBWalletAmount.Amount.Int64() - testFee.AckFee.AmountOf(chainADenom).Int64()
			s.Require().Equal(expected, actualBalance)
		})
	})
}
