package keeper_test

import (
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

func (suite *KeeperTestSuite) TestWriteAcknowledgementAsync() {
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
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			// open incentivized channel
			suite.coordinator.Setup(suite.path)

			// build packet
			timeoutTimestamp := ^uint64(0)
			packet := channeltypes.NewPacket(
				[]byte("packetData"),
				1,
				suite.path.EndpointA.ChannelConfig.PortID,
				suite.path.EndpointA.ChannelID,
				suite.path.EndpointB.ChannelConfig.PortID,
				suite.path.EndpointB.ChannelID,
				clienttypes.ZeroHeight(),
				timeoutTimestamp,
			)

			ack := []byte("ack")
			chanCap := suite.chainB.GetChannelCapability(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)

			// malleate test case
			tc.malleate()

			err := suite.chainB.GetSimApp().IBCFeeKeeper.WriteAcknowledgement(suite.chainB.GetContext(), chanCap, packet, ack)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
