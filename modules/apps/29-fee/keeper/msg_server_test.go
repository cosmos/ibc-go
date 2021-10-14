package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

func (suite *KeeperTestSuite) TestRegisterCounterpartyAddress() {
	var (
		addr  string
		addr2 string
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
	}

	for _, tc := range testCases {
		suite.SetupTest()
		ctx := suite.chainA.GetContext()

		addr = suite.chainA.SenderAccount.GetAddress().String()
		addr2 = suite.chainB.SenderAccount.GetAddress().String()
		msg := types.NewMsgRegisterCounterpartyAddress(addr, addr2)
		tc.malleate()

		_, err := suite.chainA.SendMsgs(msg)

		if tc.expPass {
			suite.Require().NoError(err) // message committed

			counterpartyAddress, _ := suite.chainA.GetSimApp().IBCFeeKeeper.GetCounterpartyAddress(ctx, suite.chainA.SenderAccount.GetAddress().String())
			suite.Require().Equal(addr2, counterpartyAddress)
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
		SetupFeePath(suite.path)
		refundAcc := suite.chainA.SenderAccount.GetAddress()
		validCoin := &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}
		channelID := suite.path.EndpointA.ChannelID
		fee := &types.Fee{validCoin, validCoin, validCoin}
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
		suite.coordinator.SetupConnections(suite.path)
		SetupFeePath(suite.path)
		refundAcc := suite.chainA.SenderAccount.GetAddress()
		validCoin := &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}
		channelID := suite.path.EndpointA.ChannelID
		fee := &types.Fee{validCoin, validCoin, validCoin}
		ctxA := suite.chainA.GetContext()
		seq, _ := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceSend(ctxA, suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
		packetId := &channeltypes.PacketId{ChannelId: channelID, PortId: suite.path.EndpointA.ChannelConfig.PortID, Sequence: seq}
		identifiedPacketFee := &types.IdentifiedPacketFee{PacketId: packetId, Fee: fee, Relayers: []string{}}
		msg := types.NewMsgPayPacketFeeAsync(identifiedPacketFee, refundAcc.String())

		tc.malleate()
		_, err := suite.chainA.SendMsgs(msg)

		if tc.expPass {
			suite.Require().NoError(err) // message committed
		} else {
			suite.Require().Error(err)
		}
	}
}
