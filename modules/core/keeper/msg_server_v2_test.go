package keeper_test

import (
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

// TestV2PacketFlow sets up clients and counterparties and sends a V2 packet.
// It ensures that a V2 ack structure is used and that the ack is correctly written to state.
func (suite *KeeperTestSuite) TestMsgServerV2PacketFlow() {
	suite.SetupTest()

	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.SetupV2()

	timeoutTimestamp := suite.chainA.GetTimeoutTimestamp()

	msg := &channeltypesv2.MsgSendPacket{
		SourceId:         path.EndpointA.ClientID,
		TimeoutTimestamp: timeoutTimestamp,
		PacketData: []channeltypes.PacketData{
			{
				SourcePort:      mock.ModuleNameV2A,
				DestinationPort: mock.ModuleNameV2A,
				Payload: channeltypes.Payload{
					Version:  mock.Version,
					Encoding: "json",
					Value:    ibctesting.MockPacketData,
				},
			},
			{
				SourcePort:      mock.ModuleNameV2B,
				DestinationPort: mock.ModuleNameV2B,
				Payload: channeltypes.Payload{
					Version:  mock.Version,
					Encoding: "json",
					Value:    ibctesting.MockPacketData,
				},
			},
		},
		Signer: suite.chainA.SenderAccount.GetAddress().String(),
	}

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	suite.Require().NoError(path.EndpointB.UpdateClient())

	packet := channeltypesv2.NewPacketV2(1, msg.SourceId, path.EndpointB.ClientID, msg.TimeoutTimestamp, msg.PacketData...)

	packetCommitment := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetPacketCommitment(suite.chainA.GetContext(), host.SentinelV2PortID, packet.SourceId, packet.GetSequence())
	suite.Require().NotNil(packetCommitment)

	packetKey := host.PacketCommitmentKey(host.SentinelV2PortID, packet.SourceId, packet.GetSequence())
	proof, proofHeight := path.EndpointA.QueryProof(packetKey)
	suite.Require().NotNil(proof)
	suite.Require().False(proofHeight.IsZero())

	// RecvPacket
	recvMsg := &channeltypesv2.MsgRecvPacket{
		Packet:          packet,
		ProofCommitment: proof,
		ProofHeight:     proofHeight,
		Signer:          suite.chainB.SenderAccount.GetAddress().String(),
	}

	recvPacketResponse, err := path.EndpointB.Chain.SendMsgs(recvMsg)

	suite.Require().NotNil(recvPacketResponse)
	suite.Require().NoError(err)
	suite.Require().NoError(path.EndpointA.UpdateClient())

	// ensure that the ack that was written, is a multi ack with a single item that hat as a success status.
	ack, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), host.SentinelV2PortID, packet.DestinationId, packet.GetSequence())
	suite.Require().True(found)
	suite.Require().NotNil(ack)

	// the expected multi ack should be a successful ack for the underlying mock applications.
	expectedAck := channeltypes.MultiAcknowledgement{
		AcknowledgementResults: []channeltypes.AcknowledgementResult{
			{
				AppName: mock.ModuleNameV2A,
				RecvPacketResult: channeltypes.RecvPacketResult{
					Status:          channeltypes.PacketStatus_Success,
					Acknowledgement: ibctesting.MockAcknowledgement,
				},
			},
			{
				AppName: mock.ModuleNameV2B,
				RecvPacketResult: channeltypes.RecvPacketResult{
					Status:          channeltypes.PacketStatus_Success,
					Acknowledgement: ibctesting.MockAcknowledgement,
				},
			},
		},
	}

	expectedBz := suite.chainB.Codec.MustMarshal(&expectedAck)
	expectedCommittedBz := channeltypes.CommitAcknowledgement(expectedBz)
	suite.Require().Equal(expectedCommittedBz, ack)

	packetKey = host.PacketAcknowledgementKey(host.SentinelV2PortID, packet.DestinationId, packet.GetSequence())
	proof, proofHeight = path.EndpointB.QueryProof(packetKey)

	msgAck := &channeltypesv2.MsgAcknowledgement{
		Packet:               packet,
		MultiAcknowledgement: expectedAck,
		ProofAcked:           proof,
		ProofHeight:          proofHeight,
		Signer:               suite.chainA.SenderAccount.GetAddress().String(),
	}

	ackPacketResponse, err := path.EndpointA.Chain.SendMsgs(msgAck)
	suite.Require().NoError(path.EndpointB.UpdateClient())

	suite.Require().NoError(err)
	suite.Require().NotNil(ackPacketResponse)
}
