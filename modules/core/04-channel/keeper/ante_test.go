package keeper_test

import (
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (suite *KeeperTestSuite) TestRecvPacketReCheckTx() {
	var (
		path   *ibctesting.Path
		packet types.Packet
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"channel not found",
			func() {
				packet.DestinationPort = "invalid-port" //nolint:goconst
			},
			types.ErrChannelNotFound,
		},
		{
			"redundant relay",
			func() {
				err := suite.chainB.App.GetIBCKeeper().ChannelKeeper.RecvPacketReCheckTx(suite.chainB.GetContext(), packet)
				suite.Require().NoError(err)
			},
			types.ErrNoOpMsg,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			tc.malleate()

			err = suite.chainB.App.GetIBCKeeper().ChannelKeeper.RecvPacketReCheckTx(suite.chainB.GetContext(), packet)

			if tc.expError == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
