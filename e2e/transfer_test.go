package e2e

import (
	"context"
	"github.com/cosmos/ibc-go/v3/e2e/testsuite"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/stretchr/testify/suite"
	"testing"
)

const (
	pollHeightMax = uint64(50)
)

func TestTransferTestSuite(t *testing.T) {
	suite.Run(t, new(TransferTestSuite))
}

type TransferTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *TransferTestSuite) TestBasicIBCTransfer() {
	t := s.T()
	ctx := context.TODO()

	relayer, srcChainChannelInfo := s.CreateChainsRelayerAndChannel(ctx)
	srcChain, dstChain := s.GetChains()

	t.Run("test IBC transfer", func(t *testing.T) {

		startingTokenAmount := int64(10_000_000)
		srcUser := s.CreateUserOnSourceChain(ctx, startingTokenAmount)
		dstUser := s.CreateUserOnDestinationChain(ctx, startingTokenAmount)

		srcToDestWallet := ibc.WalletAmount{
			Address: srcUser.Bech32Address(dstChain.Config().Bech32Prefix), // destination address
			Denom:   srcChain.Config().Denom,
			Amount:  30000,
		}

		dstToSrcWallet := ibc.WalletAmount{
			Address: dstUser.Bech32Address(srcChain.Config().Bech32Prefix), // destination address
			Denom:   dstChain.Config().Denom,
			Amount:  20000,
		}
		var srcTx ibc.Tx
		var dstTx ibc.Tx

		t.Run("source to destination", func(t *testing.T) {
			t.Run("transfer succeeds", func(t *testing.T) {
				var err error
				srcTx, err = srcChain.SendIBCTransfer(ctx, srcChainChannelInfo.ChannelID, srcUser.KeyName, srcToDestWallet, nil)
				s.Req.NoError(err)
				s.Req.NoError(srcTx.Validate(), "source ibc transfer tx is invalid")
			})

			expected := startingTokenAmount - srcToDestWallet.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
			t.Run("tokens have been escrowed", s.AssertChainNativeBalance(ctx, srcChain, srcUser, expected))
		})

		t.Run("destination to source", func(t *testing.T) {
			t.Run("destination to source", func(t *testing.T) {
				var err error
				dstTx, err = dstChain.SendIBCTransfer(ctx, srcChainChannelInfo.Counterparty.ChannelID, dstUser.KeyName, dstToSrcWallet, nil)
				s.Req.NoError(err)
				s.Req.NoError(dstTx.Validate(), "source ibc transfer tx is invalid")
			})

			dstExpected := startingTokenAmount - dstToSrcWallet.Amount - dstChain.GetGasFeesInNativeDenom(dstTx.GasSpent)
			t.Run("tokens have been escrowed", s.AssertChainNativeBalance(ctx, dstChain, dstUser, dstExpected))
		})

		t.Run("Start relayer", func(t *testing.T) {
			s.StartRelayer(relayer)
		})

		t.Run("source chain user has expected amounts", func(t *testing.T) {
			srcAck, err := test.PollForAck(ctx, srcChain, srcTx.Height, srcTx.Height+pollHeightMax, srcTx.Packet)
			s.Req.NoError(err, "failed to get acknowledgement on source chain")
			s.Req.NoError(srcAck.Validate(), "invalid acknowledgement on source chain")

			expectedSourceUserNativeBalance := startingTokenAmount - srcToDestWallet.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
			t.Run("correct amount on source chain", s.AssertChainNativeBalance(ctx, srcChain, srcUser, expectedSourceUserNativeBalance))
			t.Run("correct amount on destination chain", func(t *testing.T) {
				bal, err := testsuite.GetCounterPartyChainBalance(ctx, srcChain, dstChain, srcUser, srcChainChannelInfo.Counterparty.PortID, srcChainChannelInfo.Counterparty.ChannelID)
				s.Req.NoError(err)
				s.Req.Equal(srcToDestWallet.Amount, bal)
			})
		})

		t.Run("destination chain user has expected amounts", func(t *testing.T) {
			dstAck, err := test.PollForAck(ctx, dstChain, dstTx.Height, dstTx.Height+pollHeightMax, dstTx.Packet)
			s.Req.NoError(err, "failed to get acknowledgement on source chain")
			s.Req.NoError(dstAck.Validate(), "invalid acknowledgement on source chain")

			t.Run("correct amount on source chain", func(t *testing.T) {
				bal, err := testsuite.GetCounterPartyChainBalance(ctx, dstChain, srcChain, dstUser, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID)
				s.Req.NoError(err)
				s.Req.Equal(dstToSrcWallet.Amount, bal)
			})

			expectedDstUserNativeBalance := startingTokenAmount - dstToSrcWallet.Amount - dstChain.GetGasFeesInNativeDenom(dstTx.GasSpent)
			t.Run("correct amount on destination chain", s.AssertChainNativeBalance(ctx, dstChain, dstUser, expectedDstUserNativeBalance))
		})
	})
}
