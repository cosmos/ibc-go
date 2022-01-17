package keeper_test

import (
	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
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
		suite.coordinator.Setup(suite.path) // setup channel

		refundAcc := suite.chainA.SenderAccount.GetAddress()
		channelID := suite.path.EndpointA.ChannelID
		fee := types.Fee{
			RecvFee:    defaultReceiveFee,
			AckFee:     defaultAckFee,
			TimeoutFee: defaultTimeoutFee,
		}
		msg := types.NewMsgPayPacketFee(fee, suite.path.EndpointA.ChannelConfig.PortID, channelID, refundAcc.String(), []string{})

		tc.malleate()
		_, err := suite.chainA.SendMsgs(msg)

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
		suite.coordinator.Setup(suite.path) // setup channel

		ctxA := suite.chainA.GetContext()

		refundAcc := suite.chainA.SenderAccount.GetAddress()

		// build packetId
		channelID := suite.path.EndpointA.ChannelID
		fee := types.Fee{
			RecvFee:    defaultReceiveFee,
			AckFee:     defaultAckFee,
			TimeoutFee: defaultTimeoutFee,
		}
		seq, _ := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceSend(ctxA, suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)

		// build fee
		packetId := channeltypes.NewPacketId(channelID, suite.path.EndpointA.ChannelConfig.PortID, seq)
		identifiedPacketFee := types.IdentifiedPacketFee{PacketId: packetId, Fee: fee, RefundAddress: refundAcc.String(), Relayers: []string{}}

		tc.malleate()

		msg := types.NewMsgPayPacketFeeAsync(identifiedPacketFee)
		_, err := suite.chainA.SendMsgs(msg)

		if tc.expPass {
			suite.Require().NoError(err) // message committed
		} else {
			suite.Require().Error(err)
		}
	}
}
