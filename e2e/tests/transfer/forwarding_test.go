//go:build !test_e2e

package transfer

import (
	"context"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
)

func TestTransferForwardingTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferForwardingTestSuite))
}

type TransferForwardingTestSuite struct {
	testsuite.E2ETestSuite
}

// TODO: replace this with actual tests https://github.com/cosmos/ibc-go/issues/6578
// this test verifies that three chains can be set up, and the relayer will relay
// packets between them as configured in the newInterchain function.
func (s *TransferForwardingTestSuite) TestThreeChainSetup() {
	ctx := context.TODO()
	t := s.T()

	// TODO: note the ThreeChainSetup fn needs to be passed to TransferChannelOptions since it calls
	// GetChains which will requires those settings. We should be able to update the function to accept
	// the chain version rather than calling GetChains.
	// https://github.com/cosmos/ibc-go/issues/6577
	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, s.TransferChannelOptions(testsuite.ThreeChainSetup()), testsuite.ThreeChainSetup())
	chains := s.GetAllChains()

	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()
	chainADenom := chainA.Config().Denom

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()
	chainBDenom := chainB.Config().Denom

	chainCWallet := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)
	chainCAddress := chainCWallet.FormattedAddress()

	t.Run("IBC transfer from A to B", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	chainBChannels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainB.Config().ChainID)
	s.Require().NoError(err)
	// channel between A and B and channel between B and C
	s.Require().Len(chainBChannels, 2)

	chainBtoCChannel := chainBChannels[0]

	t.Run("IBC transfer from B to C", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, chainBtoCChannel.PortID, chainBtoCChannel.ChannelID, testvalues.DefaultTransferCoins(chainBDenom), chainBAddress, chainCAddress, s.GetTimeoutHeight(ctx, chainC), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)
	t.Run("packets are relayed from A to B", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	chainCIBCToken := testsuite.GetIBCToken(chainBDenom, chainBtoCChannel.Counterparty.PortID, chainBtoCChannel.Counterparty.ChannelID)
	t.Run("packets are relayed from B to C", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainB, chainBtoCChannel.PortID, chainBtoCChannel.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainC, chainCAddress, chainCIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})
}
