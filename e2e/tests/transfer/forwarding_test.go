//go:build !test_e2e

package transfer

import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
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

// TestForwarding_WithLastChainBeingICS20v1_Succeeds tests the case where a token is forwarded and successfully
// received on a destination chain that is on ics20-v1 version.
func (s *TransferForwardingTestSuite) TestForwarding_WithLastChainBeingICS20v1_Succeeds() {
	s.testForwardingThreeChains(transfertypes.V1)
}

// TestForwarding_Succeeds tests the case where a token is forwarded and successfully
// received on a destination chain.
func (s *TransferForwardingTestSuite) TestForwarding_Succeeds() {
	s.testForwardingThreeChains(transfertypes.V2)
}

func (s *TransferForwardingTestSuite) testForwardingThreeChains(lastChainVersion string) {
	ctx := context.TODO()
	t := s.T()

	relayer, chains := s.GetRelayer(), s.GetAllChains()

	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	var channelBtoC ibc.ChannelOutput
	channelAtoB := s.GetChainAChannel()
	if lastChainVersion == transfertypes.V2 {
		channelBtoC = s.GetChainChannel(testsuite.ChainChannelPair{ChainIdx: 1, ChannelIdx: 1})
	} else {
		opts := s.TransferChannelOptions()
		opts.Version = transfertypes.V1
		chains := s.GetAllChains()
		channelBtoC, _ = s.CreatePath(ctx, chains[1], chains[2], ibc.DefaultClientOpts(), opts)
		s.Require().Equal(transfertypes.V1, channelBtoC.Version, "the channel version is not ics20-1")
	}

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()
	chainADenom := chainA.Config().Denom

	chainCWallet := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)
	chainCAddress := chainCWallet.FormattedAddress()

	t.Run("IBC transfer from A to C with forwarding through B", func(t *testing.T) {
		inFiveMinutes := time.Now().Add(5 * time.Minute).UnixNano()
		forwarding := transfertypes.NewForwarding(false, transfertypes.NewHop(channelBtoC.PortID, channelBtoC.ChannelID))

		msgTransfer := testsuite.GetMsgTransfer(
			channelAtoB.PortID,
			channelAtoB.ChannelID,
			transfertypes.V2,
			testvalues.DefaultTransferCoins(chainADenom),
			chainAAddress,
			chainCAddress,
			clienttypes.ZeroHeight(),
			uint64(inFiveMinutes),
			"",
			forwarding)
		resp := s.BroadcastMessages(ctx, chainA, chainAWallet, msgTransfer)
		s.AssertTxSuccess(resp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	t.Run("packets are relayed from A to B to C", func(t *testing.T) {
		chainCDenom := transfertypes.NewDenom(chainADenom,
			transfertypes.NewHop(channelBtoC.Counterparty.PortID, channelBtoC.Counterparty.ChannelID),
			transfertypes.NewHop(channelAtoB.Counterparty.PortID, channelAtoB.Counterparty.ChannelID),
		)

		s.AssertPacketRelayed(ctx, chainA, channelAtoB.PortID, channelAtoB.ChannelID, 1)
		s.AssertPacketRelayed(ctx, chainB, channelBtoC.PortID, channelBtoC.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainC, chainCAddress, chainCDenom.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})
}

// TestForwardingWithUnwindSucceeds tests the forwarding scenario in which
// a packet is sent from A to B, then unwound back to A and forwarded to C
// The overall flow of the packet is:
// A ---> B
// B --(unwind)-->A --(forward)-->B --(forward)--> C
func (s *TransferForwardingTestSuite) TestForwardingWithUnwindSucceeds() {
	t := s.T()
	ctx := context.TODO()
	relayer, chains := s.GetRelayer(), s.GetAllChains()

	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	channelAtoB := s.GetChainAChannel()
	channelBtoC := s.GetChainChannel(testsuite.ChainChannelPair{ChainIdx: 1, ChannelIdx: 1})

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()
	chainADenom := chainA.Config().Denom

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	chainCWallet := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)
	chainCAddress := chainCWallet.FormattedAddress()

	t.Run("IBC transfer from A to B", func(t *testing.T) {
		inFiveMinutes := time.Now().Add(5 * time.Minute).UnixNano()

		msgTransfer := testsuite.GetMsgTransfer(
			channelAtoB.PortID,
			channelAtoB.ChannelID,
			transfertypes.V2,
			testvalues.DefaultTransferCoins(chainADenom),
			chainAAddress,
			chainBAddress,
			clienttypes.ZeroHeight(),
			uint64(inFiveMinutes),
			"",
			nil)
		resp := s.BroadcastMessages(ctx, chainA, chainAWallet, msgTransfer)
		s.AssertTxSuccess(resp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	chainBDenom := transfertypes.NewDenom(chainADenom, transfertypes.NewHop(channelAtoB.Counterparty.PortID, channelAtoB.Counterparty.ChannelID))
	t.Run("packet has reached B", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelAtoB.PortID, channelAtoB.ChannelID, 1)

		balance, err := query.Balance(ctx, chainB, chainBAddress, chainBDenom.IBCDenom())
		s.Require().NoError(err)

		s.Require().Equal(testvalues.IBCTransferAmount, balance.Int64())
	})

	t.Run("IBC transfer from B (unwind) to C through A", func(t *testing.T) {
		inFiveMinutes := time.Now().Add(5 * time.Minute).UnixNano()

		forwarding := transfertypes.NewForwarding(
			true,
			transfertypes.NewHop(channelAtoB.PortID, channelAtoB.ChannelID),
			transfertypes.NewHop(channelBtoC.PortID, channelBtoC.ChannelID),
		)
		msgTransfer := testsuite.GetMsgTransfer(
			"",
			"",
			transfertypes.V2,
			testvalues.DefaultTransferCoins(chainBDenom.IBCDenom()),
			chainBAddress,
			chainCAddress,
			clienttypes.ZeroHeight(),
			uint64(inFiveMinutes),
			"",
			forwarding)
		resp := s.BroadcastMessages(ctx, chainB, chainBWallet, msgTransfer)
		s.AssertTxSuccess(resp)
	})
	t.Run("packet has reached C", func(t *testing.T) {
		chainCDenom := transfertypes.NewDenom(chainADenom,
			transfertypes.NewHop(channelAtoB.Counterparty.PortID, channelAtoB.Counterparty.ChannelID),
			transfertypes.NewHop(channelBtoC.Counterparty.PortID, channelBtoC.Counterparty.ChannelID),
		)

		err := test.WaitForCondition(time.Minute*10, time.Second*30, func() (bool, error) {
			balance, err := query.Balance(ctx, chainC, chainCAddress, chainCDenom.IBCDenom())
			if err != nil {
				return false, err
			}
			return balance.Int64() == testvalues.IBCTransferAmount, nil
		})
		s.Require().NoError(err)
	})
}
