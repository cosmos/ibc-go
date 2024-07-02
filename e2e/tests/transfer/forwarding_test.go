//go:build !test_e2e

package transfer

import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
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
	ctx := context.TODO()
	t := s.T()

	relayer, chains := s.GetRelayer(), s.GetAllChains()

	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	channelAtoB := s.GetChainAChannel()

	// Creating a new path between chain B and chain C with a ICS20-v1 channel
	opts := s.TransferChannelOptions()
	opts.Version = transfertypes.V1
	channelBtoC, _ := s.CreatePath(ctx, chainB, chainC, ibc.DefaultClientOpts(), opts)
	s.Require().Equal(transfertypes.V1, channelBtoC.Version, "the channel version is not ics20-1")

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
