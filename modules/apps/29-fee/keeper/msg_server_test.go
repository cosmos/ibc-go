package keeper_test

import (
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
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
			"success",
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
		msg := types.NewMsgRegisterCounterpartyAddress(sender, counterparty)

		_, err := suite.chainA.SendMsgs(msg)

		if tc.expPass {
			suite.Require().NoError(err) // message committed

			counterpartyAddress, _ := suite.chainA.GetSimApp().IBCFeeKeeper.GetCounterpartyAddress(ctx, suite.chainA.SenderAccount.GetAddress().String())
			suite.Require().Equal(counterparty, counterpartyAddress)
		} else {
			suite.Require().Error(err)
		}
	}
}

func (suite *KeeperTestSuite) TestPayPacketFee() {
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
	}

	for _, tc := range testCases {
		suite.SetupTest()
		suite.coordinator.SetupConnections(suite.path)
		err := SetupFeePath(suite.path)
		suite.Require().NoError(err)

		refundAcc := suite.chainA.SenderAccount.GetAddress()
		channelID := suite.path.EndpointA.ChannelID
		fee := types.Fee{
			ReceiveFee: validCoins,
			AckFee:     validCoins,
			TimeoutFee: validCoins,
		}
		msg := types.NewMsgPayPacketFee(fee, suite.path.EndpointA.ChannelConfig.PortID, channelID, refundAcc.String(), []string{})

		tc.malleate()
		_, err = suite.chainA.SendMsgs(msg)

		if tc.expPass {
			suite.Require().NoError(err) // message committed
		} else {
			suite.Require().Error(err)
		}
	}
}

func (suite *KeeperTestSuite) TestPayPacketFeeAsync() {
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
	}

	for _, tc := range testCases {
		suite.SetupTest()
		suite.coordinator.SetupConnections(suite.path)
		err := SetupFeePath(suite.path)
		suite.Require().NoError(err)

		ctxA := suite.chainA.GetContext()

		refundAcc := suite.chainA.SenderAccount.GetAddress()

		// build packetId
		channelID := suite.path.EndpointA.ChannelID
		fee := types.Fee{
			ReceiveFee: validCoins,
			AckFee:     validCoins,
			TimeoutFee: validCoins,
		}
		seq, _ := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceSend(ctxA, suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)

		// build fee
		packetId := &channeltypes.PacketId{ChannelId: channelID, PortId: suite.path.EndpointA.ChannelConfig.PortID, Sequence: seq}
		identifiedPacketFee := types.IdentifiedPacketFee{PacketId: packetId, Fee: fee, RefundAddress: refundAcc.String(), Relayers: []string{}}

		tc.malleate()

		msg := types.NewMsgPayPacketFeeAsync(identifiedPacketFee)
		_, err = suite.chainA.SendMsgs(msg)

		if tc.expPass {
			suite.Require().NoError(err) // message committed
		} else {
			suite.Require().Error(err)
		}
	}
}
