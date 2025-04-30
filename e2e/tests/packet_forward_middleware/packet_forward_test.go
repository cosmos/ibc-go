//go:build !test_e2e

package pfm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	chantypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	testifysuite "github.com/stretchr/testify/suite"
)

type PFMTestSuite struct {
	testsuite.E2ETestSuite
}

func TestForwardTransferSuite(t *testing.T) {
	testifysuite.Run(t, new(PFMTestSuite))
}

func (s *PFMTestSuite) TestForwardPacket() {
	t := s.T()
	ctx := context.TODO()
	testName := t.Name()

	chains := s.GetAllChains()
	chainA, chainB, chainC, chainD := chains[0], chains[1], chains[2], chains[3]

	// channelVersion := transfertypes.V1

	denomA := chainA.Config().Denom

	userA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	userC := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)
	userD := s.CreateUserOnChainD(ctx, testvalues.StartingTokenAmount)

	relayer := s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), t.Name())
	s.StartRelayer(relayer, testName)

	chanAB := s.ChanAToB(testName)
	chanBC := s.ChanBToC(testName)
	chanCD := s.ChanCToD(testName)

	ab, err := query.Channel(ctx, chainA, transfertypes.PortID, chanAB.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(ab)

	bc, err := query.Channel(ctx, chainB, transfertypes.PortID, chanBC.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(bc)

	cd, err := query.Channel(ctx, chainC, transfertypes.PortID, chanCD.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(cd)

	// Send packet from Chain A->Chain B->Chain C->Chain D
	// From A -> B will be handled by transfer msg.
	// From B -> C will be handled by firstHopMetadata.
	// From C -> D will be handled by secondHopMetadata.
	secondHopMetadata := &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: userD.FormattedAddress(),
			Channel:  chanCD.ChannelID,
			Port:     chanCD.PortID,
		},
	}
	nextBz, err := json.Marshal(secondHopMetadata)
	s.Require().NoError(err)
	next := string(nextBz)

	firstHopMetadata := &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: userC.FormattedAddress(),
			Channel:  chanBC.ChannelID,
			Port:     chanBC.PortID,
			Next:     &next,
		},
	}

	memo, err := json.Marshal(firstHopMetadata)
	s.Require().NoError(err)

	bHeight, err := chainB.Height(ctx)
	s.Require().NoError(err)

	txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, string(memo))
	s.AssertTxSuccess(txResp)

	packet, err := ibctesting.ParsePacketFromEvents(txResp.Events)
	s.Require().NoError(err)
	s.Require().NotNil(packet)

	// Poll for MsgRecvPacket on chainB
	_, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, chainB.(*cosmos.CosmosChain), cosmos.DefaultEncoding().InterfaceRegistry, bHeight, bHeight+40, nil)
	s.Require().NoError(err)

	actualBalance, err := s.GetChainANativeBalance(ctx, userA)
	s.Require().NoError(err)
	expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
	s.Require().Equal(expected, actualBalance)

	escrowAddrA := transfertypes.GetEscrowAddress(chanAB.PortID, chanAB.ChannelID)
	escrowAddrB := transfertypes.GetEscrowAddress(chanCD.PortID, chanCD.ChannelID)
	escrowAddrC := escrowAddrB
	escrowAddrD := userD.FormattedAddress()

	escrowBalA, err := query.Balance(ctx, chainA, escrowAddrA.String(), denomA)
	s.Require().NoError(err)
	s.Require().Equal(testvalues.IBCTransferAmount, escrowBalA.Int64())
	// })

	time.Sleep(60 * time.Second)

	ibcTokenB := testsuite.GetIBCToken(denomA, chanAB.PortID, chanAB.ChannelID)
	escrowBalB, err := query.Balance(ctx, chainB, escrowAddrB.String(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)
	s.Require().Equal(testvalues.IBCTransferAmount, escrowBalB.Int64())

	ibcTokenC := testsuite.GetIBCToken(ibcTokenB.Path(), chanAB.Counterparty.PortID, chanCD.Counterparty.ChannelID)
	escrowBalC, err := query.Balance(ctx, chainC, escrowAddrC.String(), ibcTokenC.IBCDenom())
	s.Require().NoError(err)
	s.Require().Equal(testvalues.IBCTransferAmount, escrowBalC.Int64())

	ibcTokenD := testsuite.GetIBCToken(ibcTokenC.Path(), chanCD.Counterparty.PortID, chanCD.Counterparty.ChannelID)
	balanceD, err := query.Balance(ctx, chainD, escrowAddrD, ibcTokenD.IBCDenom())
	s.Require().NoError(err)
	s.Require().Equal(testvalues.IBCTransferAmount, balanceD.Int64())

	// t.Run("recv packet ibc transfer", func(t *testing.T) {

	// Assart Packet relayed
	s.Require().Eventually(func() bool {
		_, err := query.GRPCQuery[chantypes.QueryPacketCommitmentResponse](ctx, chainA, &chantypes.QueryPacketCommitmentRequest{
			PortId:    chanAB.PortID,
			ChannelId: chanAB.ChannelID,
			Sequence:  packet.Sequence,
		})
		return strings.Contains(err.Error(), "packet commitment hash not found")
	}, time.Second*60, time.Second)

	versionB := chainB.Config().Images[0].Version
	if testvalues.TokenMetadataFeatureReleases.IsSupported(versionB) {
		// t.Run("metadata for IBC denomination exists on chainB", func(t *testing.T) {
		s.AssertHumanReadableDenom(ctx, chainB, denomA, chanAB)
		// })
	}

	// s.Require().Equal(expected, balanceC.Int64())

	// })

	/*

		t.Run("non-native IBC token transfer from chainB to chainA, receiver is source of tokens", func(t *testing.T) {
			transferTxResp := s.Transfer(ctx, chainB, chainBWallet, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(chainBIBCToken.IBCDenom()), chainBAddress, chainAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
			s.AssertTxSuccess(transferTxResp)
		})

		t.Run("tokens are escrowed", func(t *testing.T) {
			actualBalance, err := query.Balance(ctx, chainB, chainBAddress, chainBIBCToken.IBCDenom())
			s.Require().NoError(err)

			s.Require().Equal(sdkmath.ZeroInt(), actualBalance)

			// https://github.com/cosmos/ibc-go/issues/6742
			// if testvalues.TotalEscrowFeatureReleases.IsSupported(chainBVersion) {
			//	actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainB, chainBIBCToken.IBCDenom())
			//	s.Require().NoError(err)
			//	s.Require().Equal(sdk.NewCoin(chainBIBCToken.IBCDenom(), sdkmath.NewInt(0)), actualTotalEscrow) // total escrow is zero because sending chain is not source for tokens
			// }
		})

		s.Require().NoError(test.WaitForBlocks(ctx, 5, chainA, chainB), "failed to wait for blocks")

		t.Run("packets are relayed", func(t *testing.T) {
			s.AssertPacketRelayed(ctx, chainB, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

			actualBalance, err := s.GetChainANativeBalance(ctx, chainAWallet)
			s.Require().NoError(err)

			expected := testvalues.StartingTokenAmount
			s.Require().Equal(expected, actualBalance)
		})

	*/

}
