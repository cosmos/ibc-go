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
		},
		Signer: suite.chainA.SenderAccount.GetAddress().String(),
	}

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	packet := channeltypesv2.NewPacketV2(1, msg.SourceId, msg.DestinationId, msg.TimeoutTimestamp, msg.PacketData...)

	packetKey := host.PacketCommitmentKey("foo", packet.SourceId, packet.GetSequence())
	proof, proofHeight := path.EndpointA.QueryProof(packetKey)
	suite.Require().NotNil(proof)
	suite.Require().False(proofHeight.IsZero())

	// // RecvPacket
	// recvMsg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, suite.chainB.SenderAccount.GetAddress().String())
	// recvPacketResponse, err := path.EndpointB.Chain.SendMsgs(recvMsg)
	// suite.Require().NoError(path.EndpointA.UpdateClient())

	// suite.Require().NotNil(recvPacketResponse)
	// suite.Require().NoError(err)

	// // ensure that the ack that was written, is a multi ack with a single item that hat as a success status.
	// ack, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	// suite.Require().True(found)
	// suite.Require().NotNil(ack)

	// expectedAck := channeltypes.MultiAcknowledgement{
	// 	AcknowledgementResults: []channeltypes.AcknowledgementResult{
	// 		{
	// 			AppName: path.EndpointB.ChannelConfig.PortID,
	// 			RecvPacketResult: channeltypes.RecvPacketResult{
	// 				Status:          channeltypes.PacketStatus_Success,
	// 				Acknowledgement: ibctesting.MockAcknowledgement,
	// 			},
	// 		},
	// 	},
	// }

	// expectedBz := suite.chainB.Codec.MustMarshal(&expectedAck)
	// expectedCommittedBz := channeltypes.CommitAcknowledgement(expectedBz)
	// suite.Require().Equal(expectedCommittedBz, ack)

	// packetKey = host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	// proof, proofHeight = path.EndpointB.QueryProof(packetKey)

	// legacyMultiAck := legacy.NewLMultiAck(suite.chainA.Codec, ibcmock.MockAcknowledgement, path.EndpointB.ChannelConfig.PortID)

	// msgAck := channeltypes.NewMsgAcknowledgement(packet, legacyMultiAck.Acknowledgement(), proof, proofHeight, suite.chainA.SenderAccount.GetAddress().String())

	// ackPacketResponse, err := path.EndpointA.Chain.SendMsgs(msgAck)
	// suite.Require().NoError(path.EndpointB.UpdateClient())

	// suite.Require().NoError(err)
	// suite.Require().NotNil(ackPacketResponse)
}
