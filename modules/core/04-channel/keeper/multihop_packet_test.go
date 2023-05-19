package keeper_test

import (
	"fmt"

	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

type channelTestCase = struct {
	msg            string
	orderedChannel bool
	malleate       func()
	expPass        bool
}

// TestRecvPacketMultihop test RecvPacket on chainB. Since packet commitment verification will always
// occur last (resource instensive), only tests expected to succeed and packet commitment
// verification tests need to simulate sending a packet from chainA to chainB.
func (suite *MultihopTestSuite) TestRecvPacket() {
	var (
		packet       *types.Packet
		packetHeight exported.Height
		channelCap   *capabilitytypes.Capability
		err          error
	)

	testCases := []channelTestCase{
		{"success: ORDERED channel", true, func() {
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, true},
		{"success: UNORDERED channel", true, func() {
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, true},
		// {"success with out order packet: UNORDERED channel", false,  func() {

		// }, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			if tc.orderedChannel {
				suite.chanPath.SetChannelOrdered()
			}
			suite.SetupChannels() // setup multihop channels

			tc.malleate()

			proof, err := suite.A().QueryPacketProof(packet, packetHeight)
			suite.Require().NoError(err)

			err = suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.RecvPacket(
				suite.Z().Chain.GetContext(),
				channelCap,
				packet,
				proof,
				suite.Z().ProofHeight(),
			)

			// assert no error
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(packet)

				channelZ, found := suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.GetChannel(
					suite.Z().Chain.GetContext(),
					packet.GetDestPort(),
					packet.GetDestChannel(),
				)
				suite.Require().True(found)
				nextSeqRecv, found := suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(
					suite.Z().Chain.GetContext(),
					packet.GetDestPort(),
					packet.GetDestChannel(),
				)
				suite.Require().True(found)
				receipt, receiptStored := suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.GetPacketReceipt(
					suite.Z().Chain.GetContext(),
					packet.GetDestPort(),
					packet.GetDestChannel(),
					packet.GetSequence(),
				)

				if tc.orderedChannel {
					suite.Require().True(channelZ.Ordering == types.ORDERED)
					suite.Require().
						Equal(nextSeqRecv, packet.GetSequence()+1, "sequence not incremented in ORDERED channel")
					suite.Require().False(receiptStored, "packet receipt stored in ORDERED channel")
				} else {
					suite.Require().True(channelZ.Ordering == types.UNORDERED)
					suite.Require().Equal(nextSeqRecv, packet.GetSequence(), "sequence incremented in UNORDERED channel")
					suite.Require().Equal(nextSeqRecv, uint64(2), "sequence incremented in UNORDERED channel")
					suite.Require().True(receiptStored, "packet receipt not stored in UNORDERED channel")
					suite.Require().Equal(string([]byte{byte(1)}), receipt, "packet receipt not stored in UNORDERED channel")
				}
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestAcknowledgePacketMultihop tests the call AcknowledgePacket on chainA.
func (suite *MultihopTestSuite) TestAcknowledgePacket() {
	var (
		packet       *types.Packet
		packetHeight exported.Height
		ack          = ibcmock.MockAcknowledgement
		channelCap   *capabilitytypes.Capability
		err          error
	)

	testCases := []channelTestCase{
		{"success: ORDERED channel", true, func() {
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			suite.Require().NoError(suite.Z().RecvPacket(packet, packetHeight))
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
		{"success: UNORDERED channel", false, func() {
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			suite.Require().NoError(suite.Z().RecvPacket(packet, packetHeight))
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			if tc.orderedChannel {
				suite.chanPath.SetChannelOrdered()
			}
			suite.SetupChannels() // setup multihop channels

			tc.malleate()

			proof, err := suite.Z().QueryPacketAcknowledgementProof(packet, packetHeight)
			suite.Require().NoError(err)

			err = suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.AcknowledgePacket(
				suite.A().Chain.GetContext(),
				channelCap,
				packet,
				ack.Acknowledgement(),
				proof,
				suite.A().ProofHeight(),
			)

			if tc.expPass {
				suite.Require().NoError(err)
				pc := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(
					suite.A().Chain.GetContext(),
					packet.GetSourcePort(),
					packet.GetSourceChannel(),
					packet.GetSequence(),
				)
				channelA, found := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.GetChannel(
					suite.A().Chain.GetContext(),
					packet.GetSourcePort(),
					packet.GetSourceChannel(),
				)
				suite.Require().True(found)
				seqAck, found := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceAck(
					suite.A().Chain.GetContext(),
					packet.GetSourcePort(),
					packet.GetSourceChannel(),
				)
				suite.Require().True(found)

				suite.Require().NoError(err)
				suite.Require().Nil(pc)
				suite.Require().NotNil(packet)

				if tc.orderedChannel {
					suite.Require().True(channelA.Ordering == types.ORDERED)
					suite.Require().
						Equal(packet.GetSequence()+1, seqAck, "sequence not incremented in ORDERED channel")
				} else {
					suite.Require().True(channelA.Ordering == types.UNORDERED)
					suite.Require().Equal(packet.GetSequence(), uint64(1), "sequence incremented in UNORDERED channel")
				}
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
