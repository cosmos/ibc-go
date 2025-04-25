//go:build !test_e2e

package pfm

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"

	sdk "github.com/cosmos/cosmos-sdk/types"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
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
	chainA, chainB, chainC := chains[0], chains[1], chains[2]

	// channelVersion := transfertypes.V1

	denomA := chainA.Config().Denom

	userA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	userC := s.CreateUserOnChainC(ctx, testvalues.StartingTokenAmount)
	userD := s.CreateUserOnChainD(ctx, testvalues.StartingTokenAmount)

	relayer := s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), t.Name())
	s.StartRelayer(relayer, testName)

	chanAB := s.GetChainAChannelForTest(testName)
	chanBC := s.GetChainBChannelForTest(testName)
	chanCD := s.GetChainCChannelForTest(testName)

	// t.Run("query localhost transfer channel ends", func(t *testing.T) {
	channelEndA, err := query.Channel(ctx, chainA, transfertypes.PortID, chanAB.ChannelID)
	s.Require().NoError(err)
	s.Require().NotNil(channelEndA)
	// })

	// t.Run("send packet localhost ibc transfer", func(t *testing.T) {
	// Send packet from Chain A->Chain B->Chain C->Chain D
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
			Receiver: userD.FormattedAddress(),
			Channel:  chanBC.ChannelID,
			Port:     chanBC.PortID,
			Next:     &next,
		},
	}

	memo, err := json.Marshal(firstHopMetadata)
	s.Require().NoError(err)

	// amount := ibc.WalletAmount{
	// 	Address: userB.FormattedAddress(),
	// 	Denom:   chainA.Config().Denom,
	// 	Amount:  math.NewInt(testvalues.IBCTransferAmount),
	// }

	// // heightA, err := chainA.Height(ctx)
	// // s.Require().NoError(err)
	// opts := ibc.TransferOptions{
	// 	Memo: string(memo),
	// }

	// bHeight, err := chainB.Height(ctx)
	// s.Require().NoError(err)

	// txResp, err := chainA.SendIBCTransfer(ctx, chanAB.ChannelID, userA.KeyName(), amount, opts)
	// s.Require().NoError(err)

	// fmt.Printf("Packet.Data %s\n", string(txResp.Packet.Data))
	txResp := s.Transfer(ctx, chainA, userA, chanAB.PortID, chanAB.ChannelID, testvalues.DefaultTransferAmount(denomA), userA.FormattedAddress(), userB.FormattedAddress(), s.GetTimeoutHeight(ctx, chainA), 0, string(memo))
	s.AssertTxSuccess(txResp)

	packet, err := ibctesting.ParsePacketFromEvents(txResp.Events)
	s.Require().NoError(err)
	s.Require().NotNil(packet)

	s.Require().NotNil(txResp)

	// })

	// t.Run("tokens are escrowed", func(t *testing.T) {
	actualBalance, err := s.GetChainANativeBalance(ctx, userA)
	s.Require().NoError(err)

	// Poll for MsgRecvPacket on chainB
	// _, err = cosmos.PollForMessage[*chantypes.MsgRecvPacket](ctx, chainB.(*cosmos.CosmosChain), cosmos.DefaultEncoding().InterfaceRegistry, bHeight, bHeight+30, nil)
	//s.Require().NoError(err)

	expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
	fmt.Printf("Balance of UserA: %v\n", actualBalance)
	// s.Require().Equal(expected, actualBalance)
	// })

	time.Sleep(5 * time.Second)
	// t.Run("recv packet ibc transfer", func(t *testing.T) {
	s.AssertPacketRelayed(ctx, chainA, chanAB.PortID, chanAB.ChannelID, packet.Sequence)
	ibcTokenB := testsuite.GetIBCToken(denomA, chanAB.Counterparty.PortID, chanAB.Counterparty.ChannelID)
	balanceB, err := query.Balance(ctx, chainB, userB.FormattedAddress(), ibcTokenB.IBCDenom())
	fmt.Printf("Balance of Balance B: %s\n", balanceB)
	s.Require().NoError(err)
	expected = testvalues.IBCTransferAmount
	//s.Require().Equal(expected, balanceB.Int64())

	versionB := chainB.Config().Images[0].Version
	if testvalues.TokenMetadataFeatureReleases.IsSupported(versionB) {
		// t.Run("metadata for IBC denomination exists on chainB", func(t *testing.T) {
		s.AssertHumanReadableDenom(ctx, chainB, denomA, chanAB)
		// })
	}

	// append port and channel from this chain to denom
	// trace := []transfertypes.Hop{transfertypes.NewHop(port, channel)}
	// denom.Trace = append(trace, denom.Trace...)

	fmt.Printf("Token B Path: %s\n", ibcTokenB.Path())

	// ibcTokenC := testsuite.GetIBCToken(ibcTokenB.IBCDenom(), chanBC.PortID, chanBC.ChannelID)

	trace := []transfertypes.Hop{transfertypes.NewHop(chanCD.PortID, chanCD.ChannelID)}
	ibcTokenC := transfertypes.ExtractDenomFromPath(ibcTokenB.Path())
	ibcTokenC.Trace = append(trace, ibcTokenC.Trace...)
	fmt.Printf("Token C Path: %s\n", ibcTokenC.Path())
	// balanceC, err := query.Balance(ctx, chainC, userC.FormattedAddress(), ibcTokenC.IBCDenom())
	// s.Require().NoError(err)

	// fmt.Printf("Balance On ChainC: %s\n", balanceC)

	chainABalances, err := query.GRPCQuery[banktypes.QueryAllBalancesResponse](ctx, chainA, &banktypes.QueryAllBalancesRequest{
		Address: userA.FormattedAddress(),
	})
	s.Require().NoError(err)

	fmt.Printf("ChainA Original balances: %s\n", chainABalances.Balances.String())

	firstHopEscrowAccount := sdk.MustBech32ifyAddressBytes(chainA.Config().Bech32Prefix, transfertypes.GetEscrowAddress(chanAB.PortID, chanAB.ChannelID))

	chainABalances, err = query.GRPCQuery[banktypes.QueryAllBalancesResponse](ctx, chainA, &banktypes.QueryAllBalancesRequest{
		Address: firstHopEscrowAccount,
	})
	s.Require().NoError(err)

	fmt.Printf("ChainA ESCROW balances: %s\n", chainABalances.Balances.String())

	chainBBalances, err := query.GRPCQuery[banktypes.QueryAllBalancesResponse](ctx, chainB, &banktypes.QueryAllBalancesRequest{
		Address: userB.FormattedAddress(),
	})
	s.Require().NoError(err)

	fmt.Printf("ChainB Original balances: %s\n", chainBBalances.Balances.String())

	secondHopEscrowAccount := sdk.MustBech32ifyAddressBytes(chainB.Config().Bech32Prefix, transfertypes.GetEscrowAddress(chanBC.PortID, chanBC.ChannelID))
	chainBBalances, err = query.GRPCQuery[banktypes.QueryAllBalancesResponse](ctx, chainB, &banktypes.QueryAllBalancesRequest{
		Address: secondHopEscrowAccount,
	})
	s.Require().NoError(err)

	fmt.Printf("ChainB ESCROW balances: %s\n", chainBBalances.Balances.String())

	chainCBalances, err := query.GRPCQuery[banktypes.QueryAllBalancesResponse](ctx, chainC, &banktypes.QueryAllBalancesRequest{
		Address: userC.FormattedAddress(),
	})
	s.Require().NoError(err)

	fmt.Printf("ChainC Original balances: %s\n", chainCBalances.Balances.String())

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
	// t.Run("acknowledge packet ibc transfer", func(t *testing.T) {
	// msgAcknowledgement := channeltypes.NewMsgAcknowledgement(packet, ack, localhost.SentinelProof, clienttypes.ZeroHeight(), userA.FormattedAddress())

	// txResp = s.BroadcastMessages(ctx, chainA, userA, msgAcknowledgement)
	// s.AssertTxSuccess(txResp)
	// })

	// t.Run("verify tokens transferred", func(t *testing.T) {
	s.AssertPacketRelayed(ctx, chainA, transfertypes.PortID, chanAB.ChannelID, 1)

	ibcToken := testsuite.GetIBCToken(denomA, transfertypes.PortID, chanAB.ChannelID)
	actualBalance_, err := query.Balance(ctx, chainA, userD.FormattedAddress(), ibcToken.IBCDenom())

	s.Require().NoError(err)

	expected = testvalues.IBCTransferAmount
	s.Require().Equal(expected, actualBalance_.Int64())
	// })
}

// ChainA 499999990000atoma
// ChainB 500000000000atomb, 10000ibc/7AF52A5722E76D21F64C0D8F4E676B096D922BDFFDD930BC57EDCD184D6A7220
// ChainC 500000000000atomc
