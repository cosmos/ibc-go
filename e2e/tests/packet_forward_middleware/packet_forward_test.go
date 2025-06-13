//go:build !test_e2e

package pfm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	pfmtypes "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
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

	relayer := s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), t.Name())
	s.StartRelayer(relayer, testName)

	chanAB := s.GetChainAToChainBChannel(testName)
	chanBC := s.GetChainBToChainCChannel(testName)
	chanCD := s.GetChainCToChainDChannel(testName)

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
	ibcTokenB := testsuite.GetIBCToken(denomA, chanAB.PortID, chanAB.ChannelID)
	ibcTokenC := testsuite.GetIBCToken(ibcTokenB.Path(), chanAB.Counterparty.PortID, chanCD.Counterparty.ChannelID)
	ibcTokenD := testsuite.GetIBCToken(ibcTokenC.Path(), chanCD.Counterparty.PortID, chanCD.Counterparty.ChannelID)

	t.Run("Multihop forward [A -> B -> C -> D]", func(_ *testing.T) {
		// Send packet from Chain A->Chain B->Chain C->Chain D
		// From A -> B will be handled by transfer msg.
		// From B -> C will be handled by firstHopMetadata.
		// From C -> D will be handled by secondHopMetadata.
		secondHopMetadata := &pfmtypes.PacketMetadata{
			Forward: &pfmtypes.ForwardMetadata{
				Receiver: userD.FormattedAddress(),
				Channel:  chanCD.ChannelID,
				Port:     chanCD.PortID,
			},
		}
		nextBz, err := json.Marshal(secondHopMetadata)
		s.Require().NoError(err)

		var next *pfmtypes.JSONObject
		json.Unmarshal(nextBz, next)
		firstHopMetadata := &pfmtypes.PacketMetadata{
			Forward: &pfmtypes.ForwardMetadata{
				Receiver: userC.FormattedAddress(),
				Channel:  chanBC.ChannelID,
				Port:     chanBC.PortID,
				Next:     next,
			},
		}

		memo, err := json.Marshal(firstHopMetadata)
		s.Require().NoError(err)

		bHeight, err := chainB.Height(ctx)
		s.Require().NoError(err)

		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, string(memo))
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParseV1PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)

		_, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, chainB.(*cosmos.CosmosChain), cosmos.DefaultEncoding().InterfaceRegistry, bHeight, bHeight+40, nil)
		s.Require().NoError(err)

		actualBalance, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)
		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)

		escrowBalAB, err := query.Balance(ctx, chainA, escrowAddrAB.String(), denomA)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, escrowBalAB.Int64())

		s.Require().Eventually(func() bool {
			_, err := query.GRPCQuery[chantypes.QueryPacketCommitmentResponse](ctx, chainA, &chantypes.QueryPacketCommitmentRequest{
				PortId:    chanAB.PortID,
				ChannelId: chanAB.ChannelID,
				Sequence:  packet.Sequence,
			})
			return err != nil && strings.Contains(err.Error(), "packet commitment hash not found")
		}, time.Second*70, time.Second)

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
		secondHopMetadata := &pfmtypes.PacketMetadata{
			Forward: &pfmtypes.ForwardMetadata{
				Receiver: userA.FormattedAddress(),
				Channel:  chanAB.Counterparty.ChannelID,
				Port:     chanAB.Counterparty.PortID,
			},
		}
		nextBz, err := json.Marshal(secondHopMetadata)
		s.Require().NoError(err)

		var next *pfmtypes.JSONObject
		json.Unmarshal(nextBz, next)

		firstHopMetadata := &pfmtypes.PacketMetadata{
			Forward: &pfmtypes.ForwardMetadata{
				Receiver: userB.FormattedAddress(),
				Channel:  chanBC.Counterparty.ChannelID,
				Port:     chanBC.Counterparty.PortID,
				Next:     next,
			},
		}

		memo, err := json.Marshal(firstHopMetadata)
		s.Require().NoError(err)

		cHeight, err := chainC.Height(ctx)
		s.Require().NoError(err)

		txResp := s.Transfer(ctx, chainD, userD, chanCD.Counterparty.PortID, chanCD.Counterparty.ChannelID, testvalues.DefaultTransferAmount(ibcTokenD.IBCDenom()), userD.FormattedAddress(), userC.FormattedAddress(), s.GetTimeoutHeight(ctx, chainD), 0, string(memo))
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParseV1PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)

		_, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, chainC.(*cosmos.CosmosChain), cosmos.DefaultEncoding().InterfaceRegistry, cHeight, cHeight+40, nil)
		s.Require().NoError(err)

		s.Require().Eventually(func() bool {
			_, err := query.GRPCQuery[chantypes.QueryPacketCommitmentResponse](ctx, chainD, &chantypes.QueryPacketCommitmentRequest{
				PortId:    chanCD.Counterparty.PortID,
				ChannelId: chanCD.Counterparty.ChannelID,
				Sequence:  packet.Sequence,
			})
			return err != nil && strings.Contains(err.Error(), "packet commitment hash not found")
		}, time.Second*70, time.Second)

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
		secondHopMetadata := &pfmtypes.PacketMetadata{
			Forward: &pfmtypes.ForwardMetadata{
				Receiver: "GurbageAddress",
				Channel:  chanCD.ChannelID,
				Port:     chanCD.PortID,
			},
		}
		nextBz, err := json.Marshal(secondHopMetadata)
		s.Require().NoError(err)

		var next *pfmtypes.JSONObject
		json.Unmarshal(nextBz, next)

		firstHopMetadata := &pfmtypes.PacketMetadata{
			Forward: &pfmtypes.ForwardMetadata{
				Receiver: userC.FormattedAddress(),
				Channel:  chanBC.ChannelID,
				Port:     chanBC.PortID,
				Next:     next,
			},
		}

		memo, err := json.Marshal(firstHopMetadata)
		s.Require().NoError(err)

		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(ibcTokenD.IBCDenom()), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, string(memo))
		s.AssertTxFailure(txResp, transfertypes.ErrDenomNotFound)

		_, err = ibctesting.ParseV1PacketFromEvents(txResp.Events)
		s.Require().ErrorContains(err, "acknowledgement event attribute not found")

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

		escrowBalBC, err = query.Balance(ctx, chainB, escrowAddrCD.String(), denomB)
		s.Require().NoError(err)
		s.Require().Equal(escrowBalBC.Int64(), testvalues.IBCTransferAmount)

		balanceA, err := query.Balance(ctx, chainA, userA.FormattedAddress(), ibcTokenA.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(balanceA.Int64(), testvalues.IBCTransferAmount)

		// Proof that unwinding happens.
		txResp = s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(ibcTokenA.IBCDenom()), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(txResp)

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

		firstHopMetadata := &pfmtypes.PacketMetadata{
			Forward: &pfmtypes.ForwardMetadata{
				Receiver: userA.FormattedAddress(),
				Channel:  chanAB.Counterparty.ChannelID,
				Port:     chanAB.Counterparty.PortID,
			},
		}

		memo, err := json.Marshal(firstHopMetadata)
		s.Require().NoError(err)

		txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, string(memo))
		s.AssertTxSuccess(txResp)

		balanceAIntAfter, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)
		balanceBIntAfter, err := s.GetChainBNativeBalance(ctx, userB)
		s.Require().NoError(err)

		s.Require().Equal(balanceAInt, balanceAIntAfter)
		s.Require().Equal(balanceBInt, balanceBIntAfter)
	})
}
