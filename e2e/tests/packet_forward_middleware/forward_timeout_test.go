//go:build !test_e2e

package pfm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/math"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

type PFMTimeoutTestSuite struct {
	testsuite.E2ETestSuite
}

func TestForwardTransferTimeoutSuite(t *testing.T) {
	testifysuite.Run(t, new(PFMTimeoutTestSuite))
}

func (s *PFMTimeoutTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 4, nil)
}

func (s *PFMTimeoutTestSuite) TestTimeoutOnForward() {
	t := s.T()
	ctx := context.TODO()
	testName := t.Name()

	chains := s.GetAllChains()
	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	userA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	userC := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)

	relayer := s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), t.Name())

	chanAB := s.GetChainAToChainBChannel(testName)
	chanBC := s.GetChainBToChainCChannel(testName)

	ab, err := query.Channel(ctx, chainA, transfertypes.PortID, chanAB.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(ab)

	bc, err := query.Channel(ctx, chainB, transfertypes.PortID, chanBC.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(bc)

	escrowAddrAB := transfertypes.GetEscrowAddress(chanAB.PortID, chanAB.ChannelID)
	escrowAddrBC := transfertypes.GetEscrowAddress(chanBC.PortID, chanBC.ChannelID)

	denomA := chainA.Config().Denom
	ibcTokenB := testsuite.GetIBCToken(denomA, chanAB.PortID, chanAB.ChannelID)
	ibcTokenC := testsuite.GetIBCToken(ibcTokenB.Path(), chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID)

	// Send packet from a -> b -> c -> d that should timeout between b -> c
	retries := uint8(0)

	firstHopMetadata := &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: userC.FormattedAddress(),
			Channel:  chanBC.ChannelID,
			Port:     chanBC.PortID,
			Retries:  &retries,
			Timeout:  time.Second * 10, // Set a timeout too short to get picked up by the relayer in time to ensure the packet times out between b -> c
		},
	}

	memo, err := json.Marshal(firstHopMetadata)
	s.Require().NoError(err)

	opts := ibc.TransferOptions{
		Memo: string(memo),
	}

	transferAmount := math.NewInt(100_000)
	walletAmount := ibc.WalletAmount{
		Address: userB.FormattedAddress(),
		Denom:   chainA.Config().Denom,
		Amount:  transferAmount,
	}

	bHeightBeforeTransfer, err := chainB.Height(ctx)
	s.Require().NoError(err)

	transferTx, err := chainA.SendIBCTransfer(ctx, chanAB.ChannelID, userA.KeyName(), walletAmount, opts)
	s.Require().NoError(err)

	s.Require().NoError(testutil.WaitForBlocks(ctx, 5, chainA, chainB))
	err = relayer.Flush(ctx, s.GetRelayerExecReporter(), s.GetPathByChains(chainA, chainB), chanAB.ChannelID)
	s.Require().NoError(err)

	// Check that the packet was received on chainB
	_, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, chainB.(*cosmos.CosmosChain), cosmos.DefaultEncoding().InterfaceRegistry, bHeightBeforeTransfer, bHeightBeforeTransfer+20, nil)
	s.Require().NoError(err)

	time.Sleep(time.Second * 12) // Wait for timeout
	s.Require().NoError(testutil.WaitForBlocks(ctx, 1, chainA, chainB))

	// Relay the packet from chainB to chainC, which should timeout
	err = relayer.Flush(ctx, s.GetRelayerExecReporter(), s.GetPathByChains(chainB, chainC), chanBC.ChannelID)
	s.Require().NoError(err)

	bHeightAfterTimeout, err := chainB.Height(ctx)
	s.Require().NoError(err)
	aHeightAfterTimeout, err := chainA.Height(ctx)
	s.Require().NoError(err)

	// Make sure there is a MsgTimeout on chainB
	_, err = cosmos.PollForMessage[*chantypes.MsgTimeout](ctx, chainB.(*cosmos.CosmosChain), chainB.Config().EncodingConfig.InterfaceRegistry, bHeightBeforeTransfer, bHeightAfterTimeout+30, nil)
	s.Require().NoError(err)

	// Relay the ack from chainB to chainA
	err = relayer.Flush(ctx, s.GetRelayerExecReporter(), s.GetPathByChains(chainB, chainA), chanAB.Counterparty.ChannelID)
	s.Require().NoError(err)

	// Make sure there is an acknowledgment on chainA
	_, err = testutil.PollForAck(ctx, chainA, aHeightAfterTimeout, aHeightAfterTimeout+30, transferTx.Packet)
	s.Require().NoError(err)

	// Assert balances to ensure that the funds are still on the original sending chain
	chainABalance, err := chainA.GetBalance(ctx, userA.FormattedAddress(), chainA.Config().Denom)
	s.Require().NoError(err)

	chainBBalance, err := chainB.GetBalance(ctx, userB.FormattedAddress(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)

	chainCBalance, err := chainC.GetBalance(ctx, userC.FormattedAddress(), ibcTokenC.IBCDenom())
	s.Require().NoError(err)

	zeroBal := math.NewInt(0)

	s.Require().Equal(testvalues.StartingTokenAmount, chainABalance.Int64())
	s.Require().Equal(zeroBal, chainCBalance)
	s.Require().Equal(zeroBal, chainBBalance)

	escrowBalanceAB, err := chainA.GetBalance(ctx, escrowAddrAB.String(), chainA.Config().Denom)
	s.Require().NoError(err)

	escrowBalanceBC, err := chainB.GetBalance(ctx, escrowAddrBC.String(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)

	s.Require().Equal(zeroBal, escrowBalanceAB)
	s.Require().Equal(zeroBal, escrowBalanceBC)

	// Send IBC transfer from ChainA -> ChainB -> ChainC that should succeed
	err = relayer.StartRelayer(ctx, s.GetRelayerExecReporter())
	s.Require().NoError(err)

	firstHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: userC.FormattedAddress(),
			Channel:  chanBC.ChannelID,
			Port:     chanBC.PortID,
		},
	}

	memo, err = json.Marshal(firstHopMetadata)
	s.Require().NoError(err)

	opts = ibc.TransferOptions{
		Memo: string(memo),
	}

	aHeightAfterTimeout, err = chainA.Height(ctx)
	s.Require().NoError(err)

	transferTx, err = chainA.SendIBCTransfer(ctx, chanAB.ChannelID, userA.KeyName(), walletAmount, opts)
	s.Require().NoError(err)

	_, err = testutil.PollForAck(ctx, chainA, aHeightAfterTimeout, aHeightAfterTimeout+30, transferTx.Packet)
	s.Require().NoError(err)

	err = testutil.WaitForBlocks(ctx, 10, chainA)
	s.Require().NoError(err)

	// Assert balances are updated to reflect tokens now being on ChainD
	chainABalance, err = chainA.GetBalance(ctx, userA.FormattedAddress(), chainA.Config().Denom)
	s.Require().NoError(err)

	chainBBalance, err = chainB.GetBalance(ctx, userB.FormattedAddress(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)

	chainCBalance, err = chainC.GetBalance(ctx, userC.FormattedAddress(), ibcTokenC.IBCDenom())
	s.Require().NoError(err)

	s.Require().Equal(testvalues.StartingTokenAmount-transferAmount.Int64(), chainABalance.Int64())
	s.Require().Equal(zeroBal, chainBBalance)
	s.Require().Equal(transferAmount, chainCBalance)

	escrowBalanceAB, err = chainA.GetBalance(ctx, escrowAddrAB.String(), chainA.Config().Denom)
	s.Require().NoError(err)

	escrowBalanceBC, err = chainB.GetBalance(ctx, escrowAddrBC.String(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)

	s.Require().Equal(transferAmount, escrowBalanceAB)
	s.Require().Equal(transferAmount, escrowBalanceBC)
}

// TODO: Try to replace this with PFM's own version of this struct #8360
type PacketMetadata struct {
	Forward *ForwardMetadata `json:"forward"`
}

type ForwardMetadata struct {
	Receiver       string        `json:"receiver"`
	Port           string        `json:"port"`
	Channel        string        `json:"channel"`
	Timeout        time.Duration `json:"timeout"`
	Retries        *uint8        `json:"retries,omitempty"`
	Next           *string       `json:"next,omitempty"`
	RefundSequence *uint64       `json:"refund_sequence,omitempty"`
}
