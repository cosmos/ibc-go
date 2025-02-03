package v2_test

import (
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *TransferTestSuite) TestFullEurekaForwardPath() {
	testCases := []struct {
		name     string
		receiver string
		hops     []types.Hop
	}{
		{
			name:     "success: 1 hop",
			receiver: suite.chainC.SenderAccount.GetAddress().String(),
			hops:     []types.Hop{types.Hop{PortId: types.PortID, ChannelId: suite.pathBToC.EndpointA.ClientID}},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			coin := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
			tokens := make([]types.Token, 1)
			var err error
			tokens[0], err = suite.chainA.GetSimApp().TransferKeeper.TokenFromCoin(suite.chainA.GetContext(), coin)
			suite.Require().NoError(err)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			transferData := types.FungibleTokenPacketDataV2{
				Tokens:     tokens,
				Sender:     suite.chainA.SenderAccount.GetAddress().String(),
				Receiver:   tc.receiver,
				Memo:       "",
				Forwarding: types.NewForwardingPacketData("", tc.hops...),
			}
			bz := suite.chainA.Codec.MustMarshal(&transferData)
			payload := channeltypesv2.NewPayload(
				types.PortID, types.PortID, types.V2,
				types.EncodingProtobuf, bz,
			)
			packetAToB, err := suite.pathAToB.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
			suite.Require().NoError(err)

			// check the original sendPacket logic
			escrowAddressA := types.GetEscrowAddress(types.PortID, suite.pathAToB.EndpointA.ClientID)
			// check that the balance for chainA is updated
			chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
			suite.Require().Equal(sdkmath.ZeroInt(), chainABalance.Amount)

			// check that module account escrow address has locked the tokens
			chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddressA, coin.Denom)
			suite.Require().Equal(coin, chainAEscrowBalance)

			res, err := suite.pathAToB.EndpointB.MsgRecvPacketWithResult(packetAToB)
			suite.Require().NoError(err)

			// check the recvPacket logic with forwarding the tokens should be moved to the next hop's escrow address
			escrowAddressB := types.GetEscrowAddress(types.PortID, suite.pathBToC.EndpointA.ClientID)
			traceAToB := types.NewHop(types.PortID, suite.pathAToB.EndpointB.ClientID)
			chainBDenom := types.NewDenom(coin.Denom, traceAToB)
			chainBBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressB, chainBDenom.IBCDenom())
			coinSentFromAToB := sdk.NewCoin(chainBDenom.IBCDenom(), coin.Amount)
			suite.Require().Equal(coinSentFromAToB, chainBBalance)

			packetBToC, err := ibctesting.ParsePacketV2FromEvents(res.Events)
			suite.Require().NoError(err)

			// check that the packet has been sent from B to C
			packetBToCCommitment := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeperV2.GetPacketCommitment(suite.chainB.GetContext(), suite.pathBToC.EndpointA.ClientID, 1)
			suite.Require().Equal(channeltypesv2.CommitPacket(packetBToC), packetBToCCommitment)

			// check that acknowledgement on chainB for packet A to B does not exist yet
			acknowledgementBToC := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeperV2.GetPacketAcknowledgement(suite.chainB.GetContext(), suite.pathAToB.EndpointA.ClientID, 1)
			suite.Require().Nil(acknowledgementBToC)

			// update the chainB client on chainC
			err = suite.pathBToC.EndpointB.UpdateClient()
			suite.Require().NoError(err)

			// recvPacket packetBToC on chain C
			res, err = suite.pathBToC.EndpointB.MsgRecvPacketWithResult(packetBToC)
			suite.Require().NoError(err)

			// check that the receiver has received final tokens on chainC
			traceBToC := types.NewHop(types.PortID, suite.pathBToC.EndpointB.ClientID)
			chainCDenom := types.NewDenom(coin.Denom, traceBToC, traceAToB)
			chainCBalance := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccount.GetAddress(), chainCDenom.IBCDenom())
			coinSentFromBToC := sdk.NewCoin(chainCDenom.IBCDenom(), coin.Amount)
			suite.Require().Equal(coinSentFromBToC, chainCBalance)

			// check that the final hop has written an acknowledgement
			ack, err := ibctesting.ParseAckV2FromEvents(res.Events)
			suite.Require().NoError(err)

			res, err = suite.pathBToC.EndpointA.MsgAcknowledgePacketWithResult(packetBToC, *ack)
			suite.Require().NoError(err)

			// check that the middle hop has now written its async acknowledgement
			ack, err = ibctesting.ParseAckV2FromEvents(res.Events)
			suite.Require().NoError(err)

			// update chainB client on chainA
			err = suite.pathAToB.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			err = suite.pathAToB.EndpointA.MsgAcknowledgePacket(packetAToB, *ack)
			suite.Require().NoError(err)
		})
	}
}

func (suite *TransferTestSuite) TestFullEurekaForwardFailedAck() {
	testCases := []struct {
		name     string
		receiver string
		hops     []types.Hop
	}{
		{
			name:     "success: 1 hop",
			receiver: suite.chainC.SenderAccount.GetAddress().String(),
			hops:     []types.Hop{types.Hop{PortId: types.PortID, ChannelId: suite.pathBToC.EndpointA.ClientID}},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			coin := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
			tokens := make([]types.Token, 1)
			var err error
			tokens[0], err = suite.chainA.GetSimApp().TransferKeeper.TokenFromCoin(suite.chainA.GetContext(), coin)
			suite.Require().NoError(err)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			transferData := types.FungibleTokenPacketDataV2{
				Tokens:     tokens,
				Sender:     suite.chainA.SenderAccount.GetAddress().String(),
				Receiver:   tc.receiver,
				Memo:       "",
				Forwarding: types.NewForwardingPacketData("", tc.hops...),
			}
			bz := suite.chainA.Codec.MustMarshal(&transferData)
			payload := channeltypesv2.NewPayload(
				types.PortID, types.PortID, types.V2,
				types.EncodingProtobuf, bz,
			)
			packetAToB, err := suite.pathAToB.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
			suite.Require().NoError(err)

			// check the original sendPacket logic
			escrowAddressA := types.GetEscrowAddress(types.PortID, suite.pathAToB.EndpointA.ClientID)
			// check that the balance for chainA is updated
			chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
			suite.Require().Equal(sdkmath.ZeroInt(), chainABalance.Amount)

			// check that module account escrow address has locked the tokens
			chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddressA, coin.Denom)
			suite.Require().Equal(coin, chainAEscrowBalance)

			res, err := suite.pathAToB.EndpointB.MsgRecvPacketWithResult(packetAToB)
			suite.Require().NoError(err)

			// check the recvPacket logic with forwarding the tokens should be moved to the next hop's escrow address
			escrowAddressB := types.GetEscrowAddress(types.PortID, suite.pathBToC.EndpointA.ClientID)
			traceAToB := types.NewHop(types.PortID, suite.pathAToB.EndpointB.ClientID)
			chainBDenom := types.NewDenom(coin.Denom, traceAToB)
			chainBBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressB, chainBDenom.IBCDenom())
			coinSentFromAToB := sdk.NewCoin(chainBDenom.IBCDenom(), coin.Amount)
			suite.Require().Equal(coinSentFromAToB, chainBBalance)

			packetBToC, err := ibctesting.ParsePacketV2FromEvents(res.Events)
			suite.Require().NoError(err)

			// check that the packet has been sent from B to C
			packetBToCCommitment := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeperV2.GetPacketCommitment(suite.chainB.GetContext(), suite.pathBToC.EndpointA.ClientID, 1)
			suite.Require().Equal(channeltypesv2.CommitPacket(packetBToC), packetBToCCommitment)

			// check that acknowledgement on chainB for packet A to B does not exist yet
			acknowledgementBToC := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeperV2.GetPacketAcknowledgement(suite.chainB.GetContext(), suite.pathAToB.EndpointA.ClientID, 1)
			suite.Require().Nil(acknowledgementBToC)

			// update the chainB client on chainC
			err = suite.pathBToC.EndpointB.UpdateClient()
			suite.Require().NoError(err)

			// turn off receive on chain C to trigger an error
			suite.chainC.GetSimApp().TransferKeeper.SetParams(suite.chainC.GetContext(), types.Params{
				SendEnabled:    true,
				ReceiveEnabled: false,
			})

			// recvPacket packetBToC on chain C
			res, err = suite.pathBToC.EndpointB.MsgRecvPacketWithResult(packetBToC)
			suite.Require().NoError(err)

			// update the chainC client on chain B
			err = suite.pathBToC.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			// check that the final hop has written an acknowledgement
			ack, err := ibctesting.ParseAckV2FromEvents(res.Events)
			suite.Require().NoError(err)

			res, err = suite.pathBToC.EndpointA.MsgAcknowledgePacketWithResult(packetBToC, *ack)
			suite.Require().NoError(err)

			// check that the middle hop has now written its async acknowledgement
			ack, err = ibctesting.ParseAckV2FromEvents(res.Events)
			suite.Require().NoError(err)

			// update chainB client on chainA
			err = suite.pathAToB.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			err = suite.pathAToB.EndpointA.MsgAcknowledgePacket(packetAToB, *ack)
			suite.Require().NoError(err)

			// check that the tokens have been refunded on original sender
			chainABalance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
			suite.Require().Equal(coin, chainABalance)
		})
	}
}

func (suite *TransferTestSuite) TestFullEurekaForwardTimeout() {
	testCases := []struct {
		name     string
		receiver string
		hops     []types.Hop
	}{
		{
			name:     "success: 1 hop",
			receiver: suite.chainC.SenderAccount.GetAddress().String(),
			hops:     []types.Hop{types.Hop{PortId: types.PortID, ChannelId: suite.pathBToC.EndpointA.ClientID}},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			coin := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
			tokens := make([]types.Token, 1)
			var err error
			tokens[0], err = suite.chainA.GetSimApp().TransferKeeper.TokenFromCoin(suite.chainA.GetContext(), coin)
			suite.Require().NoError(err)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			transferData := types.FungibleTokenPacketDataV2{
				Tokens:     tokens,
				Sender:     suite.chainA.SenderAccount.GetAddress().String(),
				Receiver:   tc.receiver,
				Memo:       "",
				Forwarding: types.NewForwardingPacketData("", tc.hops...),
			}
			bz := suite.chainA.Codec.MustMarshal(&transferData)
			payload := channeltypesv2.NewPayload(
				types.PortID, types.PortID, types.V2,
				types.EncodingProtobuf, bz,
			)
			packetAToB, err := suite.pathAToB.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
			suite.Require().NoError(err)

			// check the original sendPacket logic
			escrowAddressA := types.GetEscrowAddress(types.PortID, suite.pathAToB.EndpointA.ClientID)
			// check that the balance for chainA is updated
			chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
			suite.Require().Equal(sdkmath.ZeroInt(), chainABalance.Amount)

			// check that module account escrow address has locked the tokens
			chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddressA, coin.Denom)
			suite.Require().Equal(coin, chainAEscrowBalance)

			res, err := suite.pathAToB.EndpointB.MsgRecvPacketWithResult(packetAToB)
			suite.Require().NoError(err)

			// check the recvPacket logic with forwarding the tokens should be moved to the next hop's escrow address
			escrowAddressB := types.GetEscrowAddress(types.PortID, suite.pathBToC.EndpointA.ClientID)
			traceAToB := types.NewHop(types.PortID, suite.pathAToB.EndpointB.ClientID)
			chainBDenom := types.NewDenom(coin.Denom, traceAToB)
			chainBBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressB, chainBDenom.IBCDenom())
			coinSentFromAToB := sdk.NewCoin(chainBDenom.IBCDenom(), coin.Amount)
			suite.Require().Equal(coinSentFromAToB, chainBBalance)

			packetBToC, err := ibctesting.ParsePacketV2FromEvents(res.Events)
			suite.Require().NoError(err)

			// check that the packet has been sent from B to C
			packetBToCCommitment := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeperV2.GetPacketCommitment(suite.chainB.GetContext(), suite.pathBToC.EndpointA.ClientID, 1)
			suite.Require().Equal(channeltypesv2.CommitPacket(packetBToC), packetBToCCommitment)

			// check that acknowledgement on chainB for packet A to B does not exist yet
			acknowledgementBToC := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeperV2.GetPacketAcknowledgement(suite.chainB.GetContext(), suite.pathAToB.EndpointA.ClientID, 1)
			suite.Require().Nil(acknowledgementBToC)

			// update the chainB client on chainC
			err = suite.pathBToC.EndpointB.UpdateClient()
			suite.Require().NoError(err)

			// Time out packet
			suite.coordinator.IncrementTimeBy(time.Hour * 5)
			err = suite.pathBToC.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			res, err = suite.pathBToC.EndpointA.MsgTimeoutPacketWithResult(packetBToC)
			suite.Require().NoError(err)
			ack, err := ibctesting.ParseAckV2FromEvents(res.Events)
			suite.Require().NoError(err)

			err = suite.pathAToB.EndpointA.UpdateClient()
			suite.Require().NoError(err)
			err = suite.pathAToB.EndpointA.MsgAcknowledgePacket(packetAToB, *ack)
			suite.Require().NoError(err)

			// check that the tokens have been refunded on original sender
			chainABalance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
			suite.Require().Equal(coin, chainABalance)
		})
	}
}
