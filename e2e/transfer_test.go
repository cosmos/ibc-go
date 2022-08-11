package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
)

func TestTransferTestSuite(t *testing.T) {
	suite.Run(t, new(TransferTestSuite))
}

type TransferTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *TransferTestSuite) TestMsgTransfer_Timeout_Nonincentivized() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()
	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	chainBWalletAmount := ibc.WalletAmount{
		Address: chainBWallet.Bech32Address(chainB.Config().Bech32Prefix), // destination address
		Denom:   chainA.Config().Denom,
		Amount:  testvalues.IBCTransferAmount,
	}

	t.Run("IBC transfer packet timesout", func(t *testing.T) {
		tx, err := chainA.SendIBCTransfer(ctx, channelA.ChannelID, chainAWallet.KeyName, chainBWalletAmount, testvalues.ImmediatelyTimeout())
		s.Require().NoError(err)
		s.Require().NoError(tx.Validate(), "source ibc transfer tx is invalid")
		time.Sleep(time.Nanosecond * 1) // want it to timeout immediately
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("ensure escrowed tokens have been refunded to sender due to timeout", func(t *testing.T) {
		// ensure destination address did not recieve any tokens
		bal, err := s.GetChainBNativeBalance(ctx, chainBWallet)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, bal)

		// ensure that the sender address has been successfully refunded the full amount
		bal, err = s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, bal)
	})
}

// transferChannelOptions configures both of the chains to have non-incentivized transfer channels.
func transferChannelOptions() func(options *ibc.CreateChannelOptions) {
	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = "ics20-1"
		opts.SourcePortName = "transfer"
		opts.DestPortName = "transfer"
	}
}
