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
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

// compatibility:from_version: v9.0.0
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

	testName := t.Name()
	t.Parallel()
	relayer := s.CreateDefaultPaths(testName)

	chains := s.GetAllChains()

	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	channelAtoB := s.GetChainAChannelForTest(testName)

	s.Require().Len(s.GetChannelsForTest(chainA, testName), 1, "expected one channel on chain A")
	s.Require().Len(s.GetChannelsForTest(chainB, testName), 2, "expected two channels on chain B")
	s.Require().Len(s.GetChannelsForTest(chainC, testName), 1, "expected one channel on chain C")

	var channelBtoC ibc.ChannelOutput
	if lastChainVersion == transfertypes.V2 {
		channelBtoC = s.GetChannelsForTest(chainB, testName)[1]
		s.Require().Equal(transfertypes.V2, channelBtoC.Version, "the channel version is not ics20-2")
	} else {
		opts := s.TransferChannelOptions()
		opts.Version = transfertypes.V1
		channelBtoC, _ = s.CreatePath(ctx, relayer, chainB, chainC, ibc.DefaultClientOpts(), opts, testName)
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
		resp := s.Transfer(ctx, chainA, chainAWallet, channelAtoB.PortID, channelAtoB.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainCAddress, clienttypes.ZeroHeight(), uint64(inFiveMinutes), "", forwarding)
		s.AssertTxSuccess(resp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
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

		// packet from B to C is acknowledged on chain C
		s.AssertPacketAcknowledged(ctx, chainC, channelBtoC.Counterparty.PortID, channelBtoC.Counterparty.ChannelID, 1)
		// packet from A to B is acknowledged on chain B
		s.AssertPacketAcknowledged(ctx, chainB, channelAtoB.Counterparty.PortID, channelAtoB.Counterparty.ChannelID, 1)
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
	t.Parallel()
	testName := t.Name()
	relayer := s.CreateDefaultPaths(testName)

	chains := s.GetAllChains()

	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	channelAtoB := s.GetChainAChannelForTest(testName)
	channelBtoC := s.GetChannelsForTest(chainB, testName)[1]

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()
	chainADenom := chainA.Config().Denom

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	chainBAddress := chainBWallet.FormattedAddress()

	chainCWallet := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)
	chainCAddress := chainCWallet.FormattedAddress()

	t.Run("IBC transfer from A to B", func(t *testing.T) {
		inFiveMinutes := time.Now().Add(5 * time.Minute).UnixNano()
		resp := s.Transfer(ctx, chainA, chainAWallet, channelAtoB.PortID, channelAtoB.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainBAddress, clienttypes.ZeroHeight(), uint64(inFiveMinutes), "", nil)
		s.AssertTxSuccess(resp)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	chainBDenom := transfertypes.NewDenom(chainADenom, transfertypes.NewHop(channelAtoB.Counterparty.PortID, channelAtoB.Counterparty.ChannelID))
	t.Run("packet has reached B", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelAtoB.PortID, channelAtoB.ChannelID, 1)
		s.AssertPacketAcknowledged(ctx, chainB, channelAtoB.Counterparty.PortID, channelAtoB.Counterparty.ChannelID, 1)

		balance, err := query.Balance(ctx, chainB, chainBAddress, chainBDenom.IBCDenom())
		s.Require().NoError(err)

		s.Require().Equal(testvalues.IBCTransferAmount, balance.Int64())
	})

	t.Run("IBC transfer from B (unwind) to A and forwarded to C through B", func(t *testing.T) {
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
		// packet from B to C is relayed
		s.AssertPacketRelayed(ctx, chainB, channelBtoC.PortID, channelBtoC.ChannelID, 1)
		// packet from B to C is acknowledged on chain C
		s.AssertPacketAcknowledged(ctx, chainC, channelBtoC.Counterparty.PortID, channelBtoC.Counterparty.ChannelID, 1)
		// packet from A to B is acknowledged on chain B
		s.AssertPacketAcknowledged(ctx, chainB, channelAtoB.Counterparty.PortID, channelAtoB.Counterparty.ChannelID, 2)
	})
}

func (s *TransferForwardingTestSuite) TestChannelUpgradeForwarding_Succeeds() {
	ctx := context.TODO()
	t := s.T()
	testName := t.Name()
	t.Parallel()

	relayer := s.CreateDefaultPaths(testName)
	chains := s.GetAllChains()

	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	opts := s.TransferChannelOptions()
	opts.Version = transfertypes.V1

	channelAtoB, _ := s.CreatePath(ctx, relayer, chains[0], chains[1], ibc.DefaultClientOpts(), opts, testName)
	s.Require().Equal(transfertypes.V1, channelAtoB.Version, "the channel version is not ics20-1")

	channelBtoC, _ := s.CreatePath(ctx, relayer, chains[1], chains[2], ibc.DefaultClientOpts(), opts, testName)
	s.Require().Equal(transfertypes.V1, channelBtoC.Version, "the channel version is not ics20-1")

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()
	chainADenom := chainA.Config().Denom

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	chainCWallet := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)
	chainCAddress := chainCWallet.FormattedAddress()

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})
	t.Run("execute gov proposal to initiate channel upgrade", func(t *testing.T) {
		chA, err := query.Channel(ctx, chainA, channelAtoB.PortID, channelAtoB.ChannelID)
		s.Require().NoError(err)

		upgradeFields := channeltypes.NewUpgradeFields(chA.Ordering, chA.ConnectionHops, transfertypes.V2)
		s.InitiateChannelUpgrade(ctx, chainA, chainAWallet, channelAtoB.PortID, channelAtoB.ChannelID, upgradeFields)

		chB, err := query.Channel(ctx, chainB, channelBtoC.PortID, channelBtoC.ChannelID)
		s.Require().NoError(err)

		upgradeFields = channeltypes.NewUpgradeFields(chB.Ordering, chB.ConnectionHops, transfertypes.V2)
		s.InitiateChannelUpgrade(ctx, chainB, chainBWallet, channelBtoC.PortID, channelBtoC.ChannelID, upgradeFields)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB), "failed to wait for blocks")

	t.Run("verify channel A upgraded and transfer version is ics20-2", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainA, channelAtoB.PortID, channelAtoB.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(transfertypes.V2, channel.Version, "the channel version is not ics20-2")
	})

	t.Run("verify channel B upgraded and transfer version is ics20-2", func(t *testing.T) {
		channel, err := query.Channel(ctx, chainB, channelBtoC.PortID, channelBtoC.ChannelID)
		s.Require().NoError(err)
		s.Require().Equal(transfertypes.V2, channel.Version, "the channel version is not ics20-2")
	})

	t.Run("IBC transfer from A to C with forwarding through B", func(t *testing.T) {
		inFiveMinutes := time.Now().Add(5 * time.Minute).UnixNano()
		forwarding := transfertypes.NewForwarding(false, transfertypes.NewHop(channelBtoC.PortID, channelBtoC.ChannelID))
		resp := s.Transfer(ctx, chainA, chainAWallet, channelAtoB.PortID, channelAtoB.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, chainCAddress, clienttypes.ZeroHeight(), uint64(inFiveMinutes), "", forwarding)
		s.AssertTxSuccess(resp)
	})

	t.Run("packets are relayed from A to B to C", func(t *testing.T) {
		chainCDenom := transfertypes.NewDenom(chainADenom,
			transfertypes.NewHop(channelBtoC.Counterparty.PortID, channelBtoC.Counterparty.ChannelID),
			transfertypes.NewHop(channelAtoB.Counterparty.PortID, channelAtoB.Counterparty.ChannelID),
		)

		actualBalance, err := query.Balance(ctx, chainC, chainCAddress, chainCDenom.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})
}

// TestFailedForwarding tests the scenario in which the packet is sent from
// A to C (through B) but it can't reach C (we use an invalid address).
func (s *TransferForwardingTestSuite) TestFailedForwarding() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	t.Parallel()
	relayer := s.CreateDefaultPaths(testName)
	chains := s.GetAllChains()

	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	chainADenom := chainA.Config().Denom

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.FormattedAddress()

	chainBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	chainCWallet := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)

	channelAtoB := s.GetChainAChannelForTest(testName)
	channelBtoC := s.GetChannelsForTest(chainB, testName)[1]

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("native IBC token transfer from chainA to invalid address through B", func(t *testing.T) {
		inFiveMinutes := time.Now().Add(5 * time.Minute).UnixNano()
		forwarding := transfertypes.NewForwarding(false, transfertypes.NewHop(channelBtoC.PortID, channelBtoC.ChannelID))
		resp := s.Transfer(ctx, chainA, chainAWallet, channelAtoB.PortID, channelAtoB.ChannelID, testvalues.DefaultTransferCoins(chainADenom), chainAAddress, testvalues.InvalidAddress, clienttypes.ZeroHeight(), uint64(inFiveMinutes), "", forwarding)
		s.AssertTxSuccess(resp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	t.Run("packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, channelAtoB.PortID, channelAtoB.ChannelID, 1)
	})

	t.Run("token transfer amount unescrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("balances for B and C have not changed", func(t *testing.T) {
		chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelAtoB.Counterparty.PortID, channelAtoB.Counterparty.ChannelID) // IBC token sent to chainB
		chainBBalance, err := testsuite.GetChainBalanceForDenom(ctx, chainB, chainBIBCToken.IBCDenom(), chainBWallet)
		s.Require().NoError(err)
		s.Require().Zero(chainBBalance)

		chainCIBCToken := testsuite.GetIBCToken(chainBIBCToken.IBCDenom(), channelBtoC.Counterparty.PortID, channelBtoC.Counterparty.ChannelID) // IBC token sent to chainC
		chainCBalance, err := testsuite.GetChainBalanceForDenom(ctx, chainC, chainCIBCToken.IBCDenom(), chainCWallet)
		s.Require().NoError(err)
		s.Require().Zero(chainCBalance)
	})
}
