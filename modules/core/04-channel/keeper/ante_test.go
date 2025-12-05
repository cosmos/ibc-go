package keeper_test

import (
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestRecvPacketReCheckTx() {
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
				err := s.chainB.App.GetIBCKeeper().ChannelKeeper.RecvPacketReCheckTx(s.chainB.GetContext(), packet)
				s.Require().NoError(err)
			},
			types.ErrNoOpMsg,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			tc.malleate()

			err = s.chainB.App.GetIBCKeeper().ChannelKeeper.RecvPacketReCheckTx(s.chainB.GetContext(), packet)

			if tc.expError == nil {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
