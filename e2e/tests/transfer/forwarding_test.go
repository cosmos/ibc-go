//go:build !test_e2e

package transfer

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

func TestTransferForwardingTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TransferForwardingTestSuite))
}

type TransferForwardingTestSuite struct {
	testsuite.E2ETestSuite
}

// SetupSuite explicitly sets up three chains for this test suite.
func (s *TransferForwardingTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), nil, testsuite.ThreeChainSetup())
}

// TODO: replace this with actual tests https://github.com/cosmos/ibc-go/issues/6578
// this test verifies that three chains can be set up, and the relayer will relay
// packets between them as configured in the newInterchain function.
func (s *TransferForwardingTestSuite) TestThreeChainSetup() {
	ctx := context.TODO()
	t := s.T()

	relayer, chains := s.GetRelayer(), s.GetAllChains()

	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	chainAChannels := s.GetChannels(chainA)
	chainBChannels := s.GetChannels(chainB)
	chainCChannels := s.GetChannels(chainC)

	s.Require().Len(chainAChannels, 1, "expected 1 channels on chain A")
	s.Require().Len(chainBChannels, 2, "expected 2 channels on chain B")
	s.Require().Len(chainCChannels, 1, "expected 1 channels on chain C")

	channelAToB := chainAChannels[0]
	channelBToC := chainBChannels[1]

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()
	chainADenom := chainA.Config().Denom

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()
	chainBDenom := chainB.Config().Denom

	chainCWallet := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)
	chainCAddress := chainCWallet.FormattedAddress()

	t.Run("IBC transfer from A to B", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelAToB.PortID, channelAToB.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("IBC transfer from B to C", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelBToC.PortID, channelBToC.ChannelID, testvalues.DefaultTransferCoins(chainBDenom), chainBAddress, chainCAddress, s.GetTimeoutHeight(ctx, chainC), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelAToB.Counterparty.PortID, channelAToB.Counterparty.ChannelID)
	t.Run("packets are relayed from A to B", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelAToB.PortID, channelAToB.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	chainCIBCToken := testsuite.GetIBCToken(chainBDenom, channelBToC.Counterparty.PortID, channelBToC.Counterparty.ChannelID)
	t.Run("packets are relayed from B to C", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainB, channelBToC.PortID, channelBToC.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainC, chainCAddress, chainCIBCToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})
}

func (s *TransferForwardingTestSuite) TestForwarding_WithLastChainBeingICS20v1_Succeeds() {
	ctx := context.TODO()
	t := s.T()

	threeChainSetup := testsuite.ThreeChainSetup()
	chainCChainID := "chainC-1" // Setting manually to ensure chainID is set to chainC-1
	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, func(options *ibc.CreateChannelOptions, chain1 ibc.Chain, chain2 ibc.Chain) {
		if chain1.Config().ChainID == chainCChainID || chain2.Config().ChainID == chainCChainID {
			options.Version = transfertypes.V1
		}
	}, func(options *testsuite.ChainOptions) {
		threeChainSetup(options)

		options.ChainSpecs[2].ChainID = chainCChainID
	})
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

	chainBChannels, err := relayer.GetChannels(ctx, s.GetRelayerExecReporter(), chainB.Config().ChainID)
	s.Require().NoError(err)
	// channel between A and B and channel between B and C
	s.Require().Len(chainBChannels, 2)

	chainBtoCChannel := chainBChannels[0]
	s.Require().Equal(transfertypes.V1, chainBtoCChannel.Version, "the channel version is not ics20-1")

	t.Run("IBC transfer from A to B", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, chainAWallet, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

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
