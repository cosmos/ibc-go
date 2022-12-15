package keeper_test

import (
	"errors"
	"fmt"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
	ibcmock "github.com/cosmos/ibc-go/v6/testing/mock"
)

// TestRecvPacketMultihop test RecvPacket on chainB. Since packet commitment verification will always
// occur last (resource instensive), only tests expected to succeed and packet commitment
// verification tests need to simulate sending a packet from chainA to chainB.
func (suite *MultihopTestSuite) TestRecvPacketMultihop() {
	var (
		paths      ibctesting.LinkedPaths
		packet     exported.PacketI
		channelCap *capabilitytypes.Capability
		numChains  int
		endpointA  *ibctesting.Endpoint
		endpointZ  *ibctesting.Endpoint
		expError   *sdkerrors.Error
	)

	testCases := []testCase{
		{"success: ORDERED channel", func() {
			ibctesting.SetupChannel(paths)

			sequence, err := endpointA.SendPacket(
				defaultTimeoutHeight,
				disabledTimeoutTimestamp,
				ibctesting.MockPacketData,
			)
			suite.Require().NoError(err)
			paths.UpdateClients()
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
			channelCap = endpointZ.Chain.GetChannelCapability(endpointZ.ChannelConfig.PortID, endpointZ.ChannelID)
		}, true},
	}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			numChains = 5
			_, paths = ibctesting.CreateLinkedChains(&suite.Suite, numChains)
			endpointA = paths[0].EndpointA
			endpointZ = paths[len(paths)-1].EndpointB

			expError = nil // must explicitly set for failed cases

			tc.malleate()

			// get proof of packet commitment from chainA
			packetKey := host.PacketCommitmentKey(
				packet.GetSourcePort(),
				packet.GetSourceChannel(),
				packet.GetSequence(),
			)
			expectedVal := types.CommitPacket(endpointA.Chain.Codec, packet)

			// generate multihop proof given keypath and value
			proofs, err := ibctesting.GenerateMultiHopProof(paths, packetKey, expectedVal)
			suite.Require().NoError(err)
			proofHeight := endpointZ.GetClientState().GetLatestHeight()
			proof, err := proofs.Marshal()
			suite.Require().NoError(err)

			err = endpointZ.Chain.App.GetIBCKeeper().ChannelKeeper.RecvPacket(
				endpointZ.Chain.GetContext(),
				channelCap,
				packet,
				proof,
				proofHeight,
			)

			if tc.expPass {
				suite.Require().NoError(err)

				channelB, _ := endpointZ.Chain.App.GetIBCKeeper().ChannelKeeper.GetChannel(
					endpointZ.Chain.GetContext(),
					packet.GetDestPort(),
					packet.GetDestChannel(),
				)
				nextSeqRecv, found := endpointZ.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(
					endpointZ.Chain.GetContext(),
					packet.GetDestPort(),
					packet.GetDestChannel(),
				)
				suite.Require().True(found)
				receipt, receiptStored := endpointZ.Chain.App.GetIBCKeeper().ChannelKeeper.GetPacketReceipt(
					endpointZ.Chain.GetContext(),
					packet.GetDestPort(),
					packet.GetDestChannel(),
					packet.GetSequence(),
				)

				if channelB.Ordering == types.ORDERED {
					suite.Require().
						Equal(packet.GetSequence()+1, nextSeqRecv, "sequence not incremented in ordered channel")
					suite.Require().False(receiptStored, "packet receipt stored on ORDERED channel")
				} else {
					suite.Require().Equal(uint64(1), nextSeqRecv, "sequence incremented for UNORDERED channel")
					suite.Require().True(receiptStored, "packet receipt not stored after RecvPacket in UNORDERED channel")
					suite.Require().Equal(string([]byte{byte(1)}), receipt, "packet receipt is not empty string")
				}
			} else {
				suite.Require().Error(err)

				// only check if expError is set, since not all error codes can be known
				if expError != nil {
					suite.Require().True(errors.Is(err, expError))
				}
			}
		})
	}
}

// TestAcknowledgePacketMultihop tests the call AcknowledgePacket on chainA.
func (suite *KeeperTestSuite) TestAcknowledgePacketMultihop() {
	var (
		paths  ibctesting.LinkedPaths
		packet types.Packet
		ack    = ibcmock.MockAcknowledgement

		channelCap *capabilitytypes.Capability
		expError   *sdkerrors.Error

		numChains int
		endpointA *ibctesting.Endpoint
		endpointZ *ibctesting.Endpoint
	)

	testCases := []testCase{
		{"success on ordered channel", func() {
			ibctesting.SetupChannel(paths)
			// create packet commitment
			sequence, err := endpointA.SendPacket(
				defaultTimeoutHeight,
				disabledTimeoutTimestamp,
				ibctesting.MockPacketData,
			)
			suite.Require().NoError(err)

			paths.UpdateClients()
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
			_, err = ibctesting.RecvPacket(paths, packet)
			suite.Require().NoError(err)
			paths.Reverse().UpdateClients()
			channelCap = endpointA.Chain.GetChannelCapability(endpointA.ChannelConfig.PortID, endpointA.ChannelID)
		}, true},
	}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			numChains = 5
			_, paths = ibctesting.CreateLinkedChains(&suite.Suite, numChains)
			endpointA = paths[0].EndpointA
			endpointZ = paths[len(paths)-1].EndpointB
			expError = nil // must explcitly set error for failed cases

			tc.malleate()

			packetKey := host.PacketAcknowledgementKey(
				packet.GetDestPort(),
				packet.GetDestChannel(),
				packet.GetSequence(),
			)
			expectedVal := types.CommitAcknowledgement(ack.Acknowledgement()) //resp.Acknowledgement

			proofs, err := ibctesting.GenerateMultiHopProof(paths.Reverse(), packetKey, expectedVal)
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
