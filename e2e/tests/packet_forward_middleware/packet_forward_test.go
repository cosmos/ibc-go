//go:build !test_e2e

package pfm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
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

	// Store B's height to poll for ack later.
	bHeight, err := chainB.Height(ctx)
	s.Require().NoError(err)

	denomA := chainA.Config().Denom
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

	// Assart Packet relayed
	s.Require().Eventually(func() bool {
		_, err := query.GRPCQuery[chantypes.QueryPacketCommitmentResponse](ctx, chainA, &chantypes.QueryPacketCommitmentRequest{
			PortId:    chanAB.PortID,
			ChannelId: chanAB.ChannelID,
			Sequence:  packet.Sequence,
		})
		return err != nil && strings.Contains(err.Error(), "packet commitment hash not found")
	}, time.Second*70, time.Second)

	// Assert human readable denom.
	versionB := chainB.Config().Images[0].Version
	if testvalues.TokenMetadataFeatureReleases.IsSupported(versionB) {
		s.AssertHumanReadableDenom(ctx, chainB, denomA, chanAB)
	}

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

	// fmt.Printf("Chain A Escrow Addr: %s\n", escrowAddrA.String())
	// s.printChainBalances(ctx, chainA, userA.FormattedAddress())
	// fmt.Printf("Chain B Escrow Addr: %s\n", escrowAddrB.String())
	// s.printChainBalances(ctx, chainB, userB.FormattedAddress())
	// fmt.Printf("Chain C Escrow Addr: %s\n", escrowAddrC.String())
	// s.printChainBalances(ctx, chainC, userC.FormattedAddress())
	// fmt.Printf("Chain D Escrow Addr: %s\n", userD.FormattedAddress())
	// s.printChainBalances(ctx, chainD, userD.FormattedAddress())

	// fmt.Printf("------------------------\n---------------------\n")

	// Send from D -> C -> B -> A
	secondHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: userA.FormattedAddress(),
			Channel:  chanAB.Counterparty.ChannelID,
			Port:     chanAB.Counterparty.PortID,
		},
	}
	nextBz, err = json.Marshal(secondHopMetadata)
	s.Require().NoError(err)
	next = string(nextBz)

	firstHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: userB.FormattedAddress(),
			Channel:  chanBC.Counterparty.ChannelID,
			Port:     chanBC.Counterparty.PortID,
			Next:     &next,
		},
	}

	memo, err = json.Marshal(firstHopMetadata)
	s.Require().NoError(err)

	// Store height for later use.
	cHeight, err := chainC.Height(ctx)
	s.Require().NoError(err)

	txResp = s.Transfer(ctx, chainD, userD, chanCD.Counterparty.PortID, chanCD.Counterparty.ChannelID, testvalues.DefaultTransferAmount(ibcTokenD.IBCDenom()), userD.FormattedAddress(), userC.FormattedAddress(), s.GetTimeoutHeight(ctx, chainD), 0, string(memo))
	s.AssertTxSuccess(txResp)

	packet, err = ibctesting.ParsePacketFromEvents(txResp.Events)
	s.Require().NoError(err)
	s.Require().NotNil(packet)

	// fmt.Printf("D -> A Packer Sequence: %v\n", packet.Sequence)

	// Poll for MsgRecvPacket on chainC
	_, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, chainC.(*cosmos.CosmosChain), cosmos.DefaultEncoding().InterfaceRegistry, cHeight, cHeight+40, nil)
	s.Require().NoError(err)

	// fmt.Printf("Chain A Escrow Addr: %s\n", escrowAddrA.String())
	// s.printChainBalances(ctx, chainA, userA.FormattedAddress())
	// fmt.Printf("Chain B Escrow Addr: %s\n", escrowAddrB.String())
	// s.printChainBalances(ctx, chainB, userB.FormattedAddress())
	// fmt.Printf("Chain C Escrow Addr: %s\n", escrowAddrC.String())
	// s.printChainBalances(ctx, chainC, userC.FormattedAddress())
	// fmt.Printf("Chain D Escrow Addr: %s\n", userD.FormattedAddress())
	// s.printChainBalances(ctx, chainD, userD.FormattedAddress())

	// Assart Packet relayed
	s.Require().Eventually(func() bool {
		_, err := query.GRPCQuery[chantypes.QueryPacketCommitmentResponse](ctx, chainD, &chantypes.QueryPacketCommitmentRequest{
			PortId:    chanCD.Counterparty.PortID,
			ChannelId: chanCD.Counterparty.ChannelID,
			Sequence:  packet.Sequence,
		})
		return err != nil && strings.Contains(err.Error(), "packet commitment hash not found")
	}, time.Second*70, time.Second)

	// All escrow accounts have been cleared
	escrowBalA, err = query.Balance(ctx, chainA, escrowAddrA.String(), denomA)
	s.Require().NoError(err)
	s.Require().Zero(escrowBalA.Int64())

	escrowBalB, err = query.Balance(ctx, chainB, escrowAddrB.String(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)
	s.Require().Zero(escrowBalB.Int64())

	escrowBalC, err = query.Balance(ctx, chainC, escrowAddrC.String(), ibcTokenC.IBCDenom())
	s.Require().NoError(err)
	s.Require().Zero(escrowBalC.Int64())

	escrowBalD, err := query.Balance(ctx, chainD, userD.FormattedAddress(), ibcTokenD.IBCDenom())
	s.Require().NoError(err)
	s.Require().Zero(escrowBalD.Int64())

	// User A has his asset back
	balance, err := s.GetChainANativeBalance(ctx, userA)
	s.Require().NoError(err)
	s.Require().Equal(testvalues.StartingTokenAmount, balance)

	// Error in forwarding: Refunded
	// Send from A -> B -> C -< D
	//
	secondHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: "GurbageAddress",
			Channel:  chanCD.ChannelID,
			Port:     chanCD.PortID,
		},
	}
	nextBz, err = json.Marshal(secondHopMetadata)
	s.Require().NoError(err)
	next = string(nextBz)

	firstHopMetadata = &PacketMetadata{
		Forward: &ForwardMetadata{
			Receiver: userC.FormattedAddress(),
			Channel:  chanBC.ChannelID,
			Port:     chanBC.PortID,
			Next:     &next,
		},
	}

	memo, err = json.Marshal(firstHopMetadata)
	s.Require().NoError(err)

	txResp = s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(ibcTokenD.IBCDenom()), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, string(memo))
	s.AssertTxFailure(txResp, transfertypes.ErrDenomNotFound)

	packet, err = ibctesting.ParsePacketFromEvents(txResp.Events)
	s.Require().ErrorContains(err, "acknowledgement event attribute not found")

	// C -> D should not happen.
	// Refunded UserA on chain A.
	escrowBalA, err = query.Balance(ctx, chainA, escrowAddrA.String(), denomA)
	s.Require().NoError(err)
	s.Require().Zero(escrowBalA.Int64())

	escrowBalB, err = query.Balance(ctx, chainB, escrowAddrB.String(), ibcTokenB.IBCDenom())
	s.Require().NoError(err)
	s.Require().Zero(escrowBalB.Int64())

	escrowBalC, err = query.Balance(ctx, chainC, escrowAddrC.String(), ibcTokenC.IBCDenom())
	s.Require().NoError(err)
	s.Require().Zero(escrowBalC.Int64())

	escrowBalD, err = query.Balance(ctx, chainD, userD.FormattedAddress(), ibcTokenD.IBCDenom())
	s.Require().NoError(err)
	s.Require().Zero(escrowBalD.Int64())

	// User A has his asset back
	balance, err = s.GetChainANativeBalance(ctx, userA)
	s.Require().NoError(err)
	s.Require().Equal(testvalues.StartingTokenAmount, balance)

	// send normal IBC transfer from B->A to get funds in IBC denom, then do multihop A->B(native)->C->D
	// this lets us test the burn from escrow account on chain C and the escrow to escrow transfer on chain B.

	// Compose the prefixed denoms and ibc denom for asserting balances
	denomB := chainB.Config().Denom
	ibcTokenA := testsuite.GetIBCToken(denomB, chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID)
	escrowAddrB = transfertypes.GetEscrowAddress(chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID)

	txResp = s.Transfer(ctx, chainB, userB, chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID, testvalues.DefaultTransferAmount(denomB), userB.FormattedAddress(), userA.FormattedAddress(), s.GetTimeoutHeight(ctx, chainB), 0, "")
	s.AssertTxSuccess(txResp)

	s.printChainBalances(ctx, chainB, userB.FormattedAddress())
	escrowBalB, err = query.Balance(ctx, chainB, escrowAddrB.String(), denomB)
	s.Require().NoError(err)
	s.Require().Equal(escrowBalB.Int64(), testvalues.IBCTransferAmount)

	s.printChainBalances(ctx, chainA, userA.FormattedAddress())
	balanceA, err := query.Balance(ctx, chainA, userA.FormattedAddress(), ibcTokenA.IBCDenom())
	s.Require().NoError(err)
	s.Require().Equal(balanceA.Int64(), testvalues.IBCTransferAmount)

	// Proof that unwinding happenes.
	txResp = s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(ibcTokenA.IBCDenom()), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, "")
	s.AssertTxSuccess(txResp)

	// Escrow account is cleared on chain B
	escrowBalB, err = query.Balance(ctx, chainB, escrowAddrB.String(), denomB)
	s.Require().NoError(err)
	s.Require().Zero(escrowBalB.Int64())

	// ChainB user now has the same amount he started with
	balanceB, err := s.GetChainBNativeBalance(ctx, userB)
	s.Require().NoError(err)
	s.Require().Equal(testvalues.StartingTokenAmount, balanceB)
}

func (s *PFMTestSuite) printChainBalances(ctx context.Context, chain ibc.Chain, userAddr string) {
	fmt.Printf("Chain: %s\n", chain.Config().ChainID)
	resp, err := query.GRPCQuery[authtypes.QueryAccountsResponse](ctx, chain, &authtypes.QueryAccountsRequest{})
	s.Require().NoError(err)
	// Chain B addresses
	fmt.Printf("Native User: %s\n", userAddr)
	for _, acc := range resp.GetAccounts() {
		if acc.TypeUrl != "/cosmos.auth.v1beta1.BaseAccount" {
			continue
		}
		var account sdk.AccountI
		err := chain.Config().EncodingConfig.InterfaceRegistry.UnpackAny(acc, &account)
		if err != nil {
			fmt.Printf("UnpackAny Error: %s\n", err)
		}
		bal, err := query.GRPCQuery[banktypes.QueryAllBalancesResponse](ctx, chain, &banktypes.QueryAllBalancesRequest{
			Address: account.GetAddress().String(),
		})
		s.Require().NoError(err)
		if bal.Balances.String() != "" {
			fmt.Printf("Address: %s\n", account.GetAddress())
			fmt.Printf("	Balances: %s\n", bal.Balances.String())
		}
	}
}
