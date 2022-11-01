package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
)

func checkEvents(
	suite *KeeperTestSuite,
	actualEvents sdk.Events,
	expEvents map[string]map[string]string,
) {
	hasEvents := make(map[string]bool)
	for eventType := range expEvents {
		hasEvents[eventType] = false
	}

	for _, event := range actualEvents {
		expEvent, eventFound := expEvents[event.Type]
		if eventFound {
			hasEvents[event.Type] = true
			suite.Require().Len(event.Attributes, len(expEvent))
			for _, attr := range event.Attributes {
				expValue, found := expEvent[string(attr.Key)]
				suite.Require().True(found)
				suite.Require().Equal(expValue, string(attr.Value))
			}
		}
	}

	for eventName, hasEvent := range hasEvents {
		suite.Require().True(hasEvent, "event: %s was not found in events", eventName)
	}
}

func (suite *KeeperTestSuite) TestMsgTransfer() {
	var msg *types.MsgTransfer

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
			"send transfers disabled",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetParams(suite.chainA.GetContext(),
					types.Params{
						SendEnabled: false,
					},
				)
			},
			false,
		},
		{
			"invalid sender",
			func() {
				msg.Sender = "address"
			},
			false,
		},
		{
			"sender is a blocked address",
			func() {
				msg.Sender = suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName).String()
			},
			false,
		},
		{
			"channel does not exist",
			func() {
				msg.SourceChannel = "channel-100"
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path := NewTransferPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			coin := sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))
			msg = types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				coin, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(),
				suite.chainB.GetTimeoutHeight(), 0, // only use timeout height
				"memo",
			)

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().TransferKeeper.Transfer(sdk.WrapSDKContext(ctx), msg)

			expEvents := map[string]map[string]string{
				"ibc_transfer": {
					"sender":            suite.chainA.SenderAccount.GetAddress().String(),
					"receiver":          suite.chainB.SenderAccount.GetAddress().String(),
					"amount":            coin.Amount.String(),
					"denom":             coin.Denom,
					"src_port":          path.EndpointA.ChannelConfig.PortID,
					"src_channel":       path.EndpointA.ChannelID,
					"dst_port":          path.EndpointB.ChannelConfig.PortID,
					"dst_channel":       path.EndpointB.ChannelID,
					"timeout_height":    suite.chainB.GetTimeoutHeight().String(),
					"timeout_timestamp": "0",
					"memo":              "memo",
				},
			}

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().NotEqual(res.Sequence, uint64(0))

				events := ctx.EventManager().Events()
				checkEvents(suite, events, expEvents)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(res)

				events := ctx.EventManager().Events()
				suite.Require().Len(events, 0)
			}
		})
	}
}
