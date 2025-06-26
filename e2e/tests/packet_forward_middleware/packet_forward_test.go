//go:build !test_e2e

package pfm

import (
	"context"
	"testing"

	"github.com/cosmos/interchaintest/v10/ibc"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	pfmtypes "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

type PFMTestSuite struct {
	testsuite.E2ETestSuite
}

func TestForwardTransferSuite(t *testing.T) {
	testifysuite.Run(t, new(PFMTestSuite))
}

func (s *PFMTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 4, nil)
}

func (s *PFMTestSuite) TestForwardPacket() {
	t := s.T()
	ctx := context.TODO()
	testName := t.Name()

	chains := s.GetAllChains()
	chainA, chainB, chainC, chainD := chains[0], chains[1], chains[2], chains[3]

	userA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	userC := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)
	userD := s.CreateUserOnChainD(ctx, testvalues.StartingTokenAmount)

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), t.Name())
	relayer := s.GetRelayerForTest(t.Name())
	s.StartRelayer(relayer, testName)

	chanAB := s.GetChannelBetweenChains(testName, chainA, chainB)
	chanBC := s.GetChannelBetweenChains(testName, chainB, chainC)
	chanCD := s.GetChannelBetweenChains(testName, chainC, chainD)

	ab, err := query.Channel(ctx, chainA, transfertypes.PortID, chanAB.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(ab)

	bc, err := query.Channel(ctx, chainB, transfertypes.PortID, chanBC.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(bc)

	cd, err := query.Channel(ctx, chainC, transfertypes.PortID, chanCD.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(cd)

	escrowAddrAB := transfertypes.GetEscrowAddress(chanAB.PortID, chanAB.ChannelID)
	escrowAddrBC := transfertypes.GetEscrowAddress(chanBC.PortID, chanBC.ChannelID)
	escrowAddrCD := transfertypes.GetEscrowAddress(chanCD.PortID, chanCD.ChannelID)

	denomA := chainA.Config().Denom
	ibcTokenB := testsuite.GetIBCToken(denomA, chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID)
	ibcTokenC := testsuite.GetIBCToken(ibcTokenB.Path(), chanBC.Counterparty.PortID, chanBC.Counterparty.ChannelID)
	ibcTokenD := testsuite.GetIBCToken(ibcTokenC.Path(), chanCD.Counterparty.PortID, chanCD.Counterparty.ChannelID)

	t.Run("Multihop forward [A -> B -> C -> D]", func(_ *testing.T) {
		// Send packet from Chain A->Chain B->Chain C->Chain D
		// From A -> B will be handled by transfer msg.
		// From B -> C will be handled by firstHopMetadata.
		// From C -> D will be handled by secondHopMetadata.
		secondHopMetadata := pfmtypes.PacketMetadata{
			Forward: pfmtypes.ForwardMetadata{
				Receiver: userD.FormattedAddress(),
				Channel:  chanCD.ChannelID,
				Port:     chanCD.PortID,
			},
		}

		firstHopMetadata := pfmtypes.PacketMetadata{
			Forward: pfmtypes.ForwardMetadata{
				Receiver: userC.FormattedAddress(),
				Channel:  chanBC.ChannelID,
				Port:     chanBC.PortID,
				Next:     &secondHopMetadata,
			},
		}

		memo, err := firstHopMetadata.ToMemo()
		s.Require().NoError(err)

		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, memo)
		s.AssertTxSuccess(txResp)

		s.FlushPackets(ctx, relayer, []ibc.Chain{chainA, chainB, chainC, chainD})

		actualBalance, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)
		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)

		escrowBalAB, err := query.Balance(ctx, chainA, escrowAddrAB.String(), denomA)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, escrowBalAB.Int64())

		versionB := chainB.Config().Images[0].Version
		if testvalues.TokenMetadataFeatureReleases.IsSupported(versionB) {
			s.AssertHumanReadableDenom(ctx, chainB, denomA, chanAB)
		}

		escrowBalBC, err := query.Balance(ctx, chainB, escrowAddrBC.String(), ibcTokenB.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, escrowBalBC.Int64())

		escrowBalCD, err := query.Balance(ctx, chainC, escrowAddrCD.String(), ibcTokenC.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, escrowBalCD.Int64())

		balanceD, err := query.Balance(ctx, chainD, userD.FormattedAddress(), ibcTokenD.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, balanceD.Int64())
	})

	t.Run("Packet forwarded [D -> C -> B -> A]", func(_ *testing.T) {
		secondHopMetadata := pfmtypes.PacketMetadata{
			Forward: pfmtypes.ForwardMetadata{
				Receiver: userA.FormattedAddress(),
				Channel:  chanAB.Counterparty.ChannelID,
				Port:     chanAB.Counterparty.PortID,
			},
		}

		firstHopMetadata := pfmtypes.PacketMetadata{
			Forward: pfmtypes.ForwardMetadata{
				Receiver: userB.FormattedAddress(),
				Channel:  chanBC.Counterparty.ChannelID,
				Port:     chanBC.Counterparty.PortID,
				Next:     &secondHopMetadata,
			},
		}

		memo, err := firstHopMetadata.ToMemo()
		s.Require().NoError(err)

		txResp := s.Transfer(ctx, chainD, userD, chanCD.Counterparty.PortID, chanCD.Counterparty.ChannelID, testvalues.DefaultTransferAmount(ibcTokenD.IBCDenom()), userD.FormattedAddress(), userC.FormattedAddress(), s.GetTimeoutHeight(ctx, chainD), 0, memo)
		s.AssertTxSuccess(txResp)

		// Flush the packet all the way back to Chain A and then the acknowledgement back to Chain D
		s.FlushPackets(ctx, relayer, []ibc.Chain{chainA, chainB, chainC, chainD})

		// All escrow accounts have been cleared
		escrowBalAB, err := query.Balance(ctx, chainA, escrowAddrAB.String(), denomA)
		s.Require().NoError(err)
		s.Require().Zero(escrowBalAB.Int64())

		escrowBalBC, err := query.Balance(ctx, chainB, escrowAddrBC.String(), ibcTokenB.IBCDenom())
		s.Require().NoError(err)
		s.Require().Zero(escrowBalBC.Int64())

		escrowBalCD, err := query.Balance(ctx, chainC, escrowAddrCD.String(), ibcTokenC.IBCDenom())
		s.Require().NoError(err)
		s.Require().Zero(escrowBalCD.Int64())

		userDBalance, err := query.Balance(ctx, chainD, userD.FormattedAddress(), ibcTokenD.IBCDenom())
		s.Require().NoError(err)
		s.Require().Zero(userDBalance.Int64())

		// User A has his asset back
		balance, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, balance)
	})

	t.Run("Error while forwarding: Refund ok [A -> B -> C ->X D]", func(_ *testing.T) {
		secondHopMetadata := pfmtypes.PacketMetadata{
			Forward: pfmtypes.ForwardMetadata{
				Receiver: "GurbageAddress",
				Channel:  chanCD.ChannelID,
				Port:     chanCD.PortID,
			},
		}

		firstHopMetadata := pfmtypes.PacketMetadata{
			Forward: pfmtypes.ForwardMetadata{
				Receiver: userC.FormattedAddress(),
				Channel:  chanBC.ChannelID,
				Port:     chanBC.PortID,
				Next:     &secondHopMetadata,
			},
		}

		memo, err := firstHopMetadata.ToMemo()
		s.Require().NoError(err)

		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(ibcTokenD.IBCDenom()), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, memo)
		s.AssertTxFailure(txResp, transfertypes.ErrDenomNotFound)

		// Flush the packet all the way back to Chain D and then the acknowledgement back to Chain A
		s.FlushPackets(ctx, relayer, []ibc.Chain{chainA, chainB, chainC, chainD})

		// C -> D should not happen.
		// Refunded UserA on chain A.
		escrowBalAB, err := query.Balance(ctx, chainA, escrowAddrAB.String(), denomA)
		s.Require().NoError(err)
		s.Require().Zero(escrowBalAB.Int64())

		escrowBalBC, err := query.Balance(ctx, chainB, escrowAddrBC.String(), ibcTokenB.IBCDenom())
		s.Require().NoError(err)
		s.Require().Zero(escrowBalBC.Int64())

		escrowBalCD, err := query.Balance(ctx, chainC, escrowAddrCD.String(), ibcTokenC.IBCDenom())
		s.Require().NoError(err)
		s.Require().Zero(escrowBalCD.Int64())

		userDBalance, err := query.Balance(ctx, chainD, userD.FormattedAddress(), ibcTokenD.IBCDenom())
		s.Require().NoError(err)
		s.Require().Zero(userDBalance.Int64())

		// User A has his asset back
		balance, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, balance)

		// send normal IBC transfer from B->A to get funds in IBC denom, then do multihop A->B(native)->C->D
		// this lets us test the burn from escrow account on chain C and the escrow to escrow transfer on chain B.

		denomB := chainB.Config().Denom
		ibcTokenA := testsuite.GetIBCToken(denomB, chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID)
		escrowAddrCD = transfertypes.GetEscrowAddress(chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID)

		txResp = s.Transfer(ctx, chainB, userB, chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID, testvalues.DefaultTransferAmount(denomB), userB.FormattedAddress(), userA.FormattedAddress(), s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(txResp)

		s.FlushPackets(ctx, relayer, []ibc.Chain{chainB, chainA})

		escrowBalBC, err = query.Balance(ctx, chainB, escrowAddrCD.String(), denomB)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, escrowBalBC.Int64())

		balanceA, err := query.Balance(ctx, chainA, userA.FormattedAddress(), ibcTokenA.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, balanceA.Int64())

		// Proof that unwinding happens.
		txResp = s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(ibcTokenA.IBCDenom()), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(txResp)

		s.FlushPackets(ctx, relayer, []ibc.Chain{chainA, chainB})

		// Escrow account is cleared on chain B
		escrowBalBC, err = query.Balance(ctx, chainB, escrowAddrCD.String(), denomB)
		s.Require().NoError(err)
		s.Require().Zero(escrowBalBC.Int64())

		// ChainB user now has the same amount he started with
		balanceB, err := s.GetChainBNativeBalance(ctx, userB)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, balanceB)
	})

	// A -> B -> A Nothing changes
	t.Run("A -> B -> A", func(_ *testing.T) {
		balanceAInt, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)
		balanceBInt, err := s.GetChainBNativeBalance(ctx, userB)
		s.Require().NoError(err)

		firstHopMetadata := pfmtypes.PacketMetadata{
			Forward: pfmtypes.ForwardMetadata{
				Receiver: userA.FormattedAddress(),
				Channel:  chanAB.Counterparty.ChannelID,
				Port:     chanAB.Counterparty.PortID,
			},
		}

		memo, err := firstHopMetadata.ToMemo()
		s.Require().NoError(err)

		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, memo)
		s.AssertTxSuccess(txResp)

		s.FlushPackets(ctx, relayer, []ibc.Chain{chainA, chainB})
		s.FlushPackets(ctx, relayer, []ibc.Chain{chainB, chainA})

		balanceAIntAfter, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)
		balanceBIntAfter, err := s.GetChainBNativeBalance(ctx, userB)
		s.Require().NoError(err)

		s.Require().Equal(balanceAInt, balanceAIntAfter)
		s.Require().Equal(balanceBInt, balanceBIntAfter)
	})
}
