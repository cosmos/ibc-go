package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *KeeperTestSuite) TestRegisterCounterpartyAddress() {
	var (
		sender       string
		counterparty string
	)

	testCases := []struct {
		name     string
		expPass  bool
		malleate func()
	}{
		{
			"success",
			true,
			func() {},
		},
		{
			"counterparty is an arbitrary string",
			true,
			func() { counterparty = "arbitrary-string" },
		},
	}

	for _, tc := range testCases {
		suite.SetupTest()
		ctx := suite.chainA.GetContext()

		sender = suite.chainA.SenderAccount.GetAddress().String()
		counterparty = suite.chainB.SenderAccount.GetAddress().String()
		tc.malleate()
		msg := types.NewMsgRegisterCounterpartyAddress(sender, counterparty, ibctesting.FirstChannelID)

		_, err := suite.chainA.SendMsgs(msg)

		if tc.expPass {
			suite.Require().NoError(err) // message committed

			counterpartyAddress, _ := suite.chainA.GetSimApp().IBCFeeKeeper.GetCounterpartyAddress(ctx, suite.chainA.SenderAccount.GetAddress().String(), ibctesting.FirstChannelID)
			suite.Require().Equal(counterparty, counterpartyAddress)
		} else {
			suite.Require().Error(err)
		}
	}
}

func (suite *KeeperTestSuite) TestPayPacketFee() {
	var (
		msg *types.MsgPayPacketFee
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"success with existing packet fees in escrow",
			func() {
				fee := types.NewFee(defaultReceiveFee, defaultAckFee, defaultTimeoutFee)

				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
				packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string{})
				feesInEscrow := types.NewPacketFees([]types.PacketFee{packetFee})

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, feesInEscrow)
			},
			true,
		},
		{
			"fee module is locked",
			func() {
				lockFeeModule(suite.chainA)
			},
			false,
		},
		{
			"fee module disabled on channel",
			func() {
				msg.SourcePortId = "invalid-port"
				msg.SourceChannelId = "invalid-channel"
			},
			false,
		},
		{
			"invalid refund address",
			func() {
				msg.Signer = "invalid-address"
			},
			false,
		},
		{
			"refund account does not exist",
			func() {
				msg.Signer = suite.chainB.SenderAccount.GetAddress().String()
			},
			false,
		},
		{
			"acknowledgement fee balance not found",
			func() {
				msg.Fee.AckFee = invalidCoins
			},
			false,
		},
		{
			"receive fee balance not found",
			func() {
				msg.Fee.RecvFee = invalidCoins
			},
			false,
		},
		{
			"timeout fee balance not found",
			func() {
				msg.Fee.TimeoutFee = invalidCoins
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.Setup(suite.path) // setup channel

			fee := types.NewFee(defaultReceiveFee, defaultAckFee, defaultTimeoutFee)
			msg = types.NewMsgPayPacketFee(
				fee,
				suite.path.EndpointA.ChannelConfig.PortID,
				suite.path.EndpointA.ChannelID,
				suite.chainA.SenderAccount.GetAddress().String(),
				[]string{},
			)

			tc.malleate()

			_, err := suite.chainA.GetSimApp().IBCFeeKeeper.PayPacketFee(sdk.WrapSDKContext(suite.chainA.GetContext()), msg)

			if tc.expPass {
				suite.Require().NoError(err) // message committed
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestPayPacketFeeAsync() {
	var (
		msg *types.MsgPayPacketFeeAsync
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"success with existing packet fees in escrow",
			func() {
				fee := types.NewFee(defaultReceiveFee, defaultAckFee, defaultTimeoutFee)

				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
				packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string{})
				feesInEscrow := types.NewPacketFees([]types.PacketFee{packetFee})

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, feesInEscrow)
			},
			true,
		},
		{
			"fee module is locked",
			func() {
				lockFeeModule(suite.chainA)
			},
			false,
		},
		{
			"fee module disabled on channel",
			func() {
				msg.PacketId.PortId = "invalid-port"
				msg.PacketId.ChannelId = "invalid-channel"
			},
			false,
		},
		{
			"invalid refund address",
			func() {
				msg.PacketFee.RefundAddress = "invalid-address"
			},
			false,
		},
		{
			"refund account does not exist",
			func() {
				msg.PacketFee.RefundAddress = suite.chainB.SenderAccount.GetAddress().String()
			},
			false,
		},
		{
			"acknowledgement fee balance not found",
			func() {
				msg.PacketFee.Fee.AckFee = invalidCoins
			},
			false,
		},
		{
			"receive fee balance not found",
			func() {
				msg.PacketFee.Fee.RecvFee = invalidCoins
			},
			false,
		},
		{
			"timeout fee balance not found",
			func() {
				msg.PacketFee.Fee.TimeoutFee = invalidCoins
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.Setup(suite.path) // setup channel

			packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
			fee := types.NewFee(defaultReceiveFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), nil)

			msg = types.NewMsgPayPacketFeeAsync(packetID, packetFee)

			tc.malleate()

			_, err := suite.chainA.GetSimApp().IBCFeeKeeper.PayPacketFeeAsync(sdk.WrapSDKContext(suite.chainA.GetContext()), msg)

			if tc.expPass {
				suite.Require().NoError(err) // message committed
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
