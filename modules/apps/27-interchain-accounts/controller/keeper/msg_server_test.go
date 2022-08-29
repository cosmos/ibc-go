package keeper_test

import (
	sdktypes "github.com/cosmos/cosmos-sdk/types"

	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
)

func (suite *KeeperTestSuite) TestRegisterAccount() {
	var (
		msg               *icatypes.MsgRegisterAccount
		expectedChannelID = "channel-0"
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
			"invalid connection id",
			false,
			func() {
				msg.ConnectionId = "connection-100"
			},
		},
		{
			"non-empty owner address is valid",
			true,
			func() {
				msg.Owner = "<invalid-owner>"
			},
		},
		{
			"empty address invalid",
			false,
			func() {
				msg.Owner = ""
			},
		},
		{
			"port is already bound for owner but capability is claimed by another module",
			false,
			func() {
				capability := suite.chainA.GetSimApp().IBCKeeper.PortKeeper.BindPort(suite.chainA.GetContext(), TestPortID)
				err := suite.chainA.GetSimApp().TransferKeeper.ClaimCapability(suite.chainA.GetContext(), capability, host.PortPath(TestPortID))
				suite.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		suite.SetupTest()

		path := NewICAPath(suite.chainA, suite.chainB)
		suite.coordinator.SetupConnections(path)

		msg = icatypes.NewMsgRegisterAccount(ibctesting.FirstConnectionID, ibctesting.TestAccAddress, "")

		tc.malleate()

		ctx := suite.chainA.GetContext()
		res, err := suite.chainA.GetSimApp().ICAControllerKeeper.RegisterAccount(ctx, msg)

		if tc.expPass {
			suite.Require().NoError(err)
			suite.Require().NotNil(res)
			suite.Require().Equal(expectedChannelID, res.ChannelId)

			events := ctx.EventManager().Events()
			suite.Require().Len(events, 2)
			suite.Require().Equal(events[0].Type, channeltypes.EventTypeChannelOpenInit)
			suite.Require().Equal(events[1].Type, sdktypes.EventTypeMessage)
		} else {
			suite.Require().Error(err)
			suite.Require().Nil(res)
		}
	}
}
