//go:build !test_e2e

package pfm

import (
	"context"
	"testing"
	"time"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/math"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	pfmtypes "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
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
	s.SetupChains(context.TODO(), 3, nil)
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

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), t.Name())
	relayer := s.GetRelayerForTest(t.Name())

	chanAB := s.GetChannelBetweenChains(testName, chainA, chainB)
	chanBC := s.GetChannelBetweenChains(testName, chainB, chainC)

	escrowAddrAB := transfertypes.GetEscrowAddress(chanAB.PortID, chanAB.ChannelID)
	escrowAddrBC := transfertypes.GetEscrowAddress(chanBC.PortID, chanBC.ChannelID)

	denomA := chainA.Config().Denom
	ibcTokenB := testsuite.GetIBCToken(denomA, chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID)
	ibcTokenC := testsuite.GetIBCToken(ibcTokenB.Path(), chanBC.Counterparty.PortID, chanBC.Counterparty.ChannelID)

	zeroBal := math.NewInt(0)

	// Send packet from a -> b -> c that should timeout between b -> c
	retries := uint8(0)

	bToCMetadata := pfmtypes.PacketMetadata{
		Forward: pfmtypes.ForwardMetadata{
			Receiver: userC.FormattedAddress(),
			Channel:  chanBC.ChannelID,
			Port:     chanBC.PortID,
			Retries:  &retries,
			Timeout:  time.Second * 10, // Short timeout
		},
	}

	memo, err := bToCMetadata.ToMemo()
	s.Require().NoError(err)

	opts := ibc.TransferOptions{
		Memo: memo,
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

	// Verify that the users funds are still in escrow on chainA and chainB before we relay the timeout between chainB and chainC
	userABalance, err := chainA.GetBalance(ctx, userA.FormattedAddress(), chainA.Config().Denom)
	s.Require().NoError(err)
	userBBalance, err := chainB.GetBalance(ctx, userB.FormattedAddress(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)

	s.Require().Equal(testvalues.StartingTokenAmount-transferAmount.Int64(), userABalance.Int64())
	s.Require().Equal(zeroBal, userBBalance)

	escrowBalanceAB, err := chainA.GetBalance(ctx, escrowAddrAB.String(), chainA.Config().Denom)
	s.Require().NoError(err)
	escrowBalanceBC, err := chainB.GetBalance(ctx, escrowAddrBC.String(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)

	s.Require().Equal(transferAmount, escrowBalanceAB)
	s.Require().Equal(transferAmount, escrowBalanceBC)

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

	// Verify that the users funds have been returned to userA on chainA, and that all escrow balances are zero
	userABalance, err = chainA.GetBalance(ctx, userA.FormattedAddress(), chainA.Config().Denom)
	s.Require().NoError(err)

	userBBalance, err = chainB.GetBalance(ctx, userB.FormattedAddress(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)

	userCBalance, err := chainC.GetBalance(ctx, userC.FormattedAddress(), ibcTokenC.IBCDenom())
	s.Require().NoError(err)

	s.Require().Equal(testvalues.StartingTokenAmount, userABalance.Int64())
	s.Require().Equal(zeroBal, userCBalance)
	s.Require().Equal(zeroBal, userBBalance)

	escrowBalanceAB, err = chainA.GetBalance(ctx, escrowAddrAB.String(), chainA.Config().Denom)
	s.Require().NoError(err)

	escrowBalanceBC, err = chainB.GetBalance(ctx, escrowAddrBC.String(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)

	s.Require().Equal(zeroBal, escrowBalanceAB)
	s.Require().Equal(zeroBal, escrowBalanceBC)

	// Send IBC transfer from ChainA -> ChainB -> ChainC that should succeed
	err = relayer.StartRelayer(ctx, s.GetRelayerExecReporter())
	s.Require().NoError(err)

	bToCMetadata = pfmtypes.PacketMetadata{
		Forward: pfmtypes.ForwardMetadata{
			Receiver: userC.FormattedAddress(),
			Channel:  chanBC.ChannelID,
			Port:     chanBC.PortID,
		},
	}

	memo, err = bToCMetadata.ToMemo()
	s.Require().NoError(err)

	opts = ibc.TransferOptions{
		Memo: memo,
	}

	aHeightBeforeTransfer, err := chainA.Height(ctx)
	s.Require().NoError(err)

	transferTx, err = chainA.SendIBCTransfer(ctx, chanAB.ChannelID, userA.KeyName(), walletAmount, opts)
	s.Require().NoError(err)

	s.FlushPackets(ctx, relayer, []ibc.Chain{chainA, chainB, chainC})

	// Verify that the ack has come all the way back to chainA (only happens after the entire packet lifecycle is complete)
	_, err = testutil.PollForAck(ctx, chainA, aHeightBeforeTransfer, aHeightAfterTimeout+30, transferTx.Packet)
	s.Require().NoError(err)

	err = testutil.WaitForBlocks(ctx, 10, chainA)
	s.Require().NoError(err)

	// Verify that the users funds have been forwarded to userC on chainC, and that the escrow balances are correct
	userABalance, err = chainA.GetBalance(ctx, userA.FormattedAddress(), chainA.Config().Denom)
	s.Require().NoError(err)

	userBBalance, err = chainB.GetBalance(ctx, userB.FormattedAddress(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)

	userCBalance, err = chainC.GetBalance(ctx, userC.FormattedAddress(), ibcTokenC.IBCDenom())
	s.Require().NoError(err)

	s.Require().Equal(testvalues.StartingTokenAmount-transferAmount.Int64(), userABalance.Int64())
	s.Require().Equal(zeroBal, userBBalance)
	s.Require().Equal(transferAmount, userCBalance)

	escrowBalanceAB, err = chainA.GetBalance(ctx, escrowAddrAB.String(), chainA.Config().Denom)
	s.Require().NoError(err)

	escrowBalanceBC, err = chainB.GetBalance(ctx, escrowAddrBC.String(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)

	s.Require().Equal(transferAmount, escrowBalanceAB)
	s.Require().Equal(transferAmount, escrowBalanceBC)
}
