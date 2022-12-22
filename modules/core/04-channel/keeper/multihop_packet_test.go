package keeper_test

import (
	"errors"
	"fmt"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
	ibcmock "github.com/cosmos/ibc-go/v6/testing/mock"
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
		packet     *types.Packet
		channelCap *capabilitytypes.Capability
		err        error
		// expError   *sdkerrors.Error
	)

	testCases := []channelTestCase{
		{"success: ORDERED channel", true, func() {
			suite.chanPath.SetChannelOrdered()
			packet, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, true},
		{"success: UNORDERED channel", true, func() {
			packet, err = suite.A().
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
			suite.coord.Setup(suite.chanPath) // setup multihop channels

			tc.malleate()

			proof := suite.A().QueryPacketProof(packet)
			err := suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.RecvPacket(
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
func (suite *MultihopTestSuite) TestAcknowledgePacketMultihop() {
	var (
		packet types.Packet
		ack    = ibcmock.MockAcknowledgement

		channelCap *capabilitytypes.Capability
		expError   *sdkerrors.Error

		endpointA *ibctesting.Endpoint
		endpointZ *ibctesting.Endpoint
	)

	testCases := []testCase{
		{"success on ordered channel", func() {
			ibctesting.SetupChannel(suite.paths)
			// create packet commitment
			sequence, err := endpointA.SendPacket(
				defaultTimeoutHeight,
				disabledTimeoutTimestamp,
				ibctesting.MockPacketData,
			)
			suite.Require().NoError(err)

			suite.paths.UpdateClients()
			// create packet receipt and acknowledgement
			packet = types.NewPacket(
				ibctesting.MockPacketData,
				sequence,
				endpointA.ChannelConfig.PortID,
				endpointA.ChannelID,
				endpointZ.ChannelConfig.PortID,
				endpointZ.ChannelID,
				defaultTimeoutHeight,
				disabledTimeoutTimestamp,
			)
			_, err = ibctesting.RecvPacket(suite.paths, packet)
			suite.Require().NoError(err)
			suite.paths.Reverse().UpdateClients()
			channelCap = endpointA.Chain.GetChannelCapability(endpointA.ChannelConfig.PortID, endpointA.ChannelID)
		}, true},
	}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			suite.SetupTest()
			endpointA = suite.paths.A()
			endpointZ = suite.paths.Z()
			expError = nil // must explcitly set error for failed cases

			tc.malleate()

			packetKey := host.PacketAcknowledgementKey(
				packet.GetDestPort(),
				packet.GetDestChannel(),
				packet.GetSequence(),
			)
			expectedVal := types.CommitAcknowledgement(ack.Acknowledgement()) //resp.Acknowledgement

			proofs, err := ibctesting.GenerateMultiHopProof(suite.paths.Reverse(), packetKey, expectedVal, false)
			suite.Require().NoError(err)
			proofHeight := endpointA.GetClientState().GetLatestHeight()
			proof, err := proofs.Marshal()
			suite.Require().NoError(err)

			err = endpointA.Chain.App.GetIBCKeeper().ChannelKeeper.AcknowledgePacket(
				endpointA.Chain.GetContext(),
				channelCap,
				packet,
				ack.Acknowledgement(),
				proof,
				proofHeight,
			)
			suite.Require().NoError(err)
			pc := endpointA.Chain.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(
				endpointA.Chain.GetContext(),
				packet.GetSourcePort(),
				packet.GetSourceChannel(),
				packet.GetSequence(),
			)

			channelA, _ := endpointA.Chain.App.GetIBCKeeper().ChannelKeeper.GetChannel(
				endpointA.Chain.GetContext(),
				packet.GetSourcePort(),
				packet.GetSourceChannel(),
			)
			sequenceAck, _ := endpointA.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceAck(
				endpointA.Chain.GetContext(),
				packet.GetSourcePort(),
				packet.GetSourceChannel(),
			)

			if tc.expPass {
				suite.NoError(err)
				suite.Nil(pc)

				if channelA.Ordering == types.ORDERED {
					suite.Require().
						Equal(packet.GetSequence()+1, sequenceAck, "sequence not incremented in ordered channel")
				} else {
					suite.Require().Equal(uint64(1), sequenceAck, "sequence incremented for UNORDERED channel")
				}
			} else {
				suite.Error(err)
				// only check if expError is set, since not all error codes can be known
				if expError != nil {
					suite.Require().True(errors.Is(err, expError))
				}
			}
		})
	}
}
