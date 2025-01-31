package v2_test

import (
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
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
			fmt.Printf("%#v\n", packetAToB)
			// msg := channeltypesv2.NewMsgSendPacket(
			// 	suite.pathAToB.EndpointA.ClientID,
			// 	timeoutTimestamp,
			// 	suite.chainA.SenderAccount.GetAddress().String(),
			// 	payload,
			// )

			// _, err = suite.chainA.SendMsgs(msg)
			// suite.Require().NoError(err) // message committed

			// commitment := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeperV2.GetPacketCommitment(suite.chainA.GetContext(), suite.pathAToB.EndpointA.ClientID, 1)
			// fmt.Printf("commitment: %v\n", commitment)

			// check the original sendPacket logic
			escrowAddressA := types.GetEscrowAddress(types.PortID, suite.pathAToB.EndpointA.ClientID)
			// check that the balance for chainA is updated
			chainABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), coin.Denom)
			suite.Require().Equal(sdkmath.ZeroInt(), chainABalance.Amount)

			// check that module account escrow address has locked the tokens
			chainAEscrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddressA, coin.Denom)
			suite.Require().Equal(coin, chainAEscrowBalance)

			err = suite.pathAToB.EndpointB.MsgRecvPacket(packetAToB)
			suite.Require().NoError(err)

			// check the recvPacket logic with forwarding the tokens should be moved to the next hop's escrow address
			escrowAddressB := types.GetEscrowAddress(types.PortID, suite.pathBToC.EndpointA.ClientID)
			traceAToB := types.NewHop(types.PortID, suite.pathAToB.EndpointB.ClientID)
			chainBDenom := types.NewDenom(coin.Denom, traceAToB)
			chainBBalance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressB, chainBDenom.IBCDenom())
			coinSentFromAToB := sdk.NewCoin(chainBDenom.IBCDenom(), coin.Amount)
			suite.Require().Equal(coinSentFromAToB, chainBBalance)

		})
	}
}
