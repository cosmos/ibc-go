package keeper_test

import (
	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	ibcmock "github.com/cosmos/ibc-go/v9/testing/mock"
)

func (suite *KeeperTestSuite) TestWriteAcknowledgementAsync() {
	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				suite.chainB.GetSimApp().IBCFeeKeeper.SetRelayerAddressForAsyncAck(suite.chainB.GetContext(), channeltypes.NewPacketID(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID, 1), suite.chainA.SenderAccount.GetAddress().String())
				suite.chainB.GetSimApp().IBCFeeKeeper.SetCounterpartyPayeeAddress(suite.chainB.GetContext(), suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), suite.path.EndpointB.ChannelID)
			},
			nil,
		},
		{
			"relayer address not set for async WriteAcknowledgement",
			func() {},
			types.ErrRelayerNotFoundForAsyncAck,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			// open incentivized channels
			// setup pathAToC (chainA -> chainC) first in order to have different channel IDs for chainA & chainB
			suite.pathAToC.Setup()
			// setup path for chainA -> chainB
			suite.path.Setup()

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

			ack := channeltypes.NewResultAcknowledgement([]byte("success"))

			// malleate test case
			tc.malleate()

			err := suite.chainB.GetSimApp().IBCFeeKeeper.WriteAcknowledgement(suite.chainB.GetContext(), packet, ack)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				_, found := suite.chainB.GetSimApp().IBCFeeKeeper.GetRelayerAddressForAsyncAck(suite.chainB.GetContext(), channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1))
				suite.Require().False(found)

				expectedAck := types.NewIncentivizedAcknowledgement(suite.chainB.SenderAccount.GetAddress().String(), ack.Acknowledgement(), ack.Success())
				committedAck, _ := suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, 1)
				suite.Require().Equal(committedAck, channeltypes.CommitAcknowledgement(expectedAck.Acknowledgement()))
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestWriteAcknowledgementAsyncFeeDisabled() {
	// open incentivized channel
	suite.path.Setup()
	suite.chainB.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(suite.chainB.GetContext(), suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)

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

	ack := channeltypes.NewResultAcknowledgement([]byte("success"))

	err := suite.chainB.GetSimApp().IBCFeeKeeper.WriteAcknowledgement(suite.chainB.GetContext(), packet, ack)
	suite.Require().NoError(err)

	packetAck, _ := suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, 1)
	suite.Require().Equal(packetAck, channeltypes.CommitAcknowledgement(ack.Acknowledgement()))
}

func (suite *KeeperTestSuite) TestGetAppVersion() {
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
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.EndpointA.ChannelConfig.PortID = ibctesting.MockFeePort
				path.EndpointB.ChannelConfig.PortID = ibctesting.MockFeePort
				// by default a new path uses a non fee channel
				path.Setup()
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
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.path.Setup()

			portID = suite.path.EndpointA.ChannelConfig.PortID
			channelID = suite.path.EndpointA.ChannelID

			// malleate test case
			tc.malleate()

			appVersion, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetAppVersion(suite.chainA.GetContext(), portID, channelID)

			if tc.expFound {
				suite.Require().True(found)
				suite.Require().Equal(expAppVersion, appVersion)
			} else {
				suite.Require().False(found)
				suite.Require().Empty(appVersion)
			}
		})
	}
}
