package keeper_test

import (
	"github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

func (s *KeeperTestSuite) TestWriteAcknowledgementAsync() {
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				s.chainB.GetSimApp().IBCFeeKeeper.SetRelayerAddressForAsyncAck(s.chainB.GetContext(), channeltypes.NewPacketID(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID, 1), s.chainA.SenderAccount.GetAddress().String())
				s.chainB.GetSimApp().IBCFeeKeeper.SetCounterpartyPayeeAddress(s.chainB.GetContext(), s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), s.path.EndpointB.ChannelID)
			},
			true,
		},
		{
			"relayer address not set for async WriteAcknowledgement",
			func() {},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()

			// open incentivized channels
			// setup pathAToC (chainA -> chainC) first in order to have different channel IDs for chainA & chainB
			s.coordinator.Setup(s.pathAToC)
			// setup path for chainA -> chainB
			s.coordinator.Setup(s.path)

			// build packet
			timeoutTimestamp := ^uint64(0)
			packet := channeltypes.NewPacket(
				[]byte("packetData"),
				1,
				s.path.EndpointA.ChannelConfig.PortID,
				s.path.EndpointA.ChannelID,
				s.path.EndpointB.ChannelConfig.PortID,
				s.path.EndpointB.ChannelID,
				clienttypes.ZeroHeight(),
				timeoutTimestamp,
			)

			ack := channeltypes.NewResultAcknowledgement([]byte("success"))
			chanCap := s.chainB.GetChannelCapability(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID)

			// malleate test case
			tc.malleate()

			err := s.chainB.GetSimApp().IBCFeeKeeper.WriteAcknowledgement(s.chainB.GetContext(), chanCap, packet, ack)

			if tc.expPass {
				s.Require().NoError(err)
				_, found := s.chainB.GetSimApp().IBCFeeKeeper.GetRelayerAddressForAsyncAck(s.chainB.GetContext(), channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1))
				s.Require().False(found)

				expectedAck := types.NewIncentivizedAcknowledgement(s.chainB.SenderAccount.GetAddress().String(), ack.Acknowledgement(), ack.Success())
				commitedAck, _ := s.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(s.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, 1)
				s.Require().Equal(commitedAck, channeltypes.CommitAcknowledgement(expectedAck.Acknowledgement()))
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestWriteAcknowledgementAsyncFeeDisabled() {
	// open incentivized channel
	s.coordinator.Setup(s.path)
	s.chainB.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(s.chainB.GetContext(), s.path.EndpointB.ChannelConfig.PortID, "channel-0")

	// build packet
	timeoutTimestamp := ^uint64(0)
	packet := channeltypes.NewPacket(
		[]byte("packetData"),
		1,
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		s.path.EndpointB.ChannelConfig.PortID,
		s.path.EndpointB.ChannelID,
		clienttypes.ZeroHeight(),
		timeoutTimestamp,
	)

	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	chanCap := s.chainB.GetChannelCapability(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID)

	err := s.chainB.GetSimApp().IBCFeeKeeper.WriteAcknowledgement(s.chainB.GetContext(), chanCap, packet, ack)
	s.Require().NoError(err)

	packetAck, _ := s.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(s.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, 1)
	s.Require().Equal(packetAck, channeltypes.CommitAcknowledgement(ack.Acknowledgement()))
}

func (s *KeeperTestSuite) TestGetAppVersion() {
	var (
		portID        string
		channelID     string
		expAppVersion string
	)
	testCases := []struct {
		name     string
		malleate func()
		expFound bool
	}{
		{
			"success for fee enabled channel",
			func() {
				expAppVersion = ibcmock.Version
			},
			true,
		},
		{
			"success for non fee enabled channel",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.EndpointA.ChannelConfig.PortID = ibctesting.MockFeePort
				path.EndpointB.ChannelConfig.PortID = ibctesting.MockFeePort
				// by default a new path uses a non fee channel
				s.coordinator.Setup(path)
				portID = path.EndpointA.ChannelConfig.PortID
				channelID = path.EndpointA.ChannelID

				expAppVersion = ibcmock.Version
			},
			true,
		},
		{
			"channel does not exist",
			func() {
				channelID = "does not exist"
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()
			s.coordinator.Setup(s.path)

			portID = s.path.EndpointA.ChannelConfig.PortID
			channelID = s.path.EndpointA.ChannelID

			// malleate test case
			tc.malleate()

			appVersion, found := s.chainA.GetSimApp().IBCFeeKeeper.GetAppVersion(s.chainA.GetContext(), portID, channelID)

			if tc.expFound {
				s.Require().True(found)
				s.Require().Equal(expAppVersion, appVersion)
			} else {
				s.Require().False(found)
				s.Require().Empty(appVersion)
			}
		})
	}
}
