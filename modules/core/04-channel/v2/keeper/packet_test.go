package keeper_test

import (
	"fmt"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	mockv2 "github.com/cosmos/ibc-go/v9/testing/mock/v2"
)

var unusedChannel = "channel-5"

func (suite *KeeperTestSuite) TestSendPacket() {
	var (
		path        *ibctesting.Path
		packet      types.Packet
		expSequence uint64
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
			"success with later packet",
			func() {
				// send the same packet earlier so next packet send should be sequence 2
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceChannel, packet.TimeoutTimestamp, packet.Data)
				suite.Require().NoError(err)
				expSequence = 2
			},
			nil,
		},
		{
			"channel not found",
			func() {
				packet.SourceChannel = ibctesting.InvalidID
			},
			types.ErrChannelNotFound,
		},
		{
			"packet failed basic validation",
			func() {
				// invalid data
				packet.Data = nil
			},
			channeltypes.ErrInvalidPacket,
		},
		{
			"client status invalid",
			func() {
				path.EndpointA.FreezeClient()
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"client state zero height", func() {
				clientState := path.EndpointA.GetClientState()
				cs, ok := clientState.(*ibctm.ClientState)
				suite.Require().True(ok)

				// force a consensus state into the store at height zero to allow client status check to pass.
				consensusState := path.EndpointA.GetConsensusState(cs.LatestHeight)
				path.EndpointA.SetConsensusState(consensusState, clienttypes.ZeroHeight())

				cs.LatestHeight = clienttypes.ZeroHeight()
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID, cs)
			},
			clienttypes.ErrInvalidHeight,
		},
		{
			"timeout elapsed", func() {
				packet.TimeoutTimestamp = 1
			},
			channeltypes.ErrTimeoutElapsed,
		},
	}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.name, i, len(testCases)), func() {
			suite.SetupTest() // reset

			// create clients and set counterparties on both chains
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			packetData := mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// create standard packet that can be malleated
			packet = types.NewPacket(1, path.EndpointA.ChannelID, path.EndpointB.ChannelID,
				timeoutTimestamp, packetData)
			expSequence = 1

			// malleate the test case
			tc.malleate()

			// send packet
			seq, destChannel, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceChannel, packet.TimeoutTimestamp, packet.Data)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				// verify send packet method instantiated packet with correct sequence and destination channel
				suite.Require().Equal(expSequence, seq)
				suite.Require().Equal(path.EndpointB.ChannelID, destChannel)
				// verify send packet stored the packet commitment correctly
				expCommitment := types.CommitPacket(packet)
				suite.Require().Equal(expCommitment, suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(suite.chainA.GetContext(), packet.SourceChannel, seq))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Equal(uint64(0), seq)
				suite.Require().Nil(suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(suite.chainA.GetContext(), packet.SourceChannel, seq))

			}
		})
	}
}

func (suite *KeeperTestSuite) TestRecvPacket() {
	var (
		path   *ibctesting.Path
		err    error
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
			"failure: channel not found",
			func() {
				packet.DestinationChannel = ibctesting.InvalidID
			},
			types.ErrChannelNotFound,
		},
		{
			"failure: client is not active",
			func() {
				path.EndpointB.FreezeClient()
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"failure: counterparty channel identifier different than source channel",
			func() {
				packet.SourceChannel = unusedChannel
			},
			channeltypes.ErrInvalidChannelIdentifier,
		},
		{
			"failure: packet has timed out",
			func() {
				suite.coordinator.IncrementTimeBy(time.Hour * 20)
			},
			channeltypes.ErrTimeoutElapsed,
		},
		{
			"failure: packet already received",
			func() {
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationChannel, packet.Sequence)
			},
			channeltypes.ErrNoOpMsg,
		},
		{
			"failure: verify membership failed",
			func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), packet.SourceChannel, packet.Sequence, []byte(""))
				suite.coordinator.CommitBlock(path.EndpointA.Chain)
				suite.Require().NoError(path.EndpointB.UpdateClient())
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			packetData := mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// send packet
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, packetData)
			suite.Require().NoError(err)

			tc.malleate()

			// get proof of v2 packet commitment from chainA
			packetKey := hostv2.PacketCommitmentKey(packet.GetSourceChannel(), packet.GetSequence())
			proof, proofHeight := path.EndpointA.QueryProof(packetKey)

			err = suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.RecvPacketTest(suite.chainB.GetContext(), packet, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				_, found := suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.GetPacketReceipt(suite.chainB.GetContext(), packet.DestinationChannel, packet.Sequence)
				suite.Require().True(found)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestWriteAcknowledgement() {
	var (
		packet types.Packet
		ack    types.Acknowledgement
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
			"failure: channel not found",
			func() {
				packet.DestinationChannel = ibctesting.InvalidID
			},
			types.ErrChannelNotFound,
		},
		{
			"failure: counterparty channel identifier different than source channel",
			func() {
				packet.SourceChannel = unusedChannel
			},
			channeltypes.ErrInvalidChannelIdentifier,
		},
		{
			"failure: ack already exists",
			func() {
				ackBz := types.CommitAcknowledgement(ack)
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketAcknowledgement(suite.chainB.GetContext(), packet.DestinationChannel, packet.Sequence, ackBz)
			},
			channeltypes.ErrAcknowledgementExists,
		},
		{
			"failure: empty ack",
			func() {
				ack = types.Acknowledgement{
					AcknowledgementResults: []types.AcknowledgementResult{},
				}
			},
			types.ErrInvalidAcknowledgement,
		},
		{
			"failure: receipt not found for packet",
			func() {
				packet.Sequence = 2
			},
			channeltypes.ErrInvalidPacket,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			packetData := mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// create standard packet that can be malleated
			packet = types.NewPacket(1, path.EndpointA.ChannelID, path.EndpointB.ChannelID,
				timeoutTimestamp, packetData)

			// create standard ack that can be malleated
			ack = types.Acknowledgement{
				AcknowledgementResults: []types.AcknowledgementResult{
					{
						AppName:          mockv2.ModuleNameB,
						RecvPacketResult: mockv2.MockRecvPacketResult,
					},
				},
			}

			suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationChannel, packet.Sequence)

			tc.malleate()

			err := suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.WriteAcknowledgement(suite.chainB.GetContext(), packet, ack)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				ackCommitment := suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.DestinationChannel, packet.Sequence)
				suite.Require().Equal(types.CommitAcknowledgement(ack), ackCommitment)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestAcknowledgePacket() {
	var (
		packet types.Packet
		err    error
		ack    = types.Acknowledgement{
			AcknowledgementResults: []types.AcknowledgementResult{{
				AppName:          mockv2.ModuleNameB,
				RecvPacketResult: mockv2.MockRecvPacketResult,
			}},
		}
		freezeClient bool
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
			"failure: channel not found",
			func() {
				packet.SourceChannel = ibctesting.InvalidID
			},
			types.ErrChannelNotFound,
		},
		{
			"failure: counterparty channel identifier different than source channel",
			func() {
				packet.DestinationChannel = unusedChannel
			},
			channeltypes.ErrInvalidChannelIdentifier,
		},
		{
			"failure: packet commitment doesn't exist.",
			func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeletePacketCommitment(suite.chainA.GetContext(), packet.SourceChannel, packet.Sequence)
			},
			channeltypes.ErrNoOpMsg,
		},
		{
			"failure: client status invalid",
			func() {
				freezeClient = true
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"failure: packet commitment bytes differ",
			func() {
				// change packet data after send to acknowledge different packet
				packet.Data[0].Payload.Value = []byte("different value")
			},
			channeltypes.ErrInvalidPacket,
		},
		{
			"failure: verify membership fails",
			func() {
				ack.AcknowledgementResults[0].RecvPacketResult = mockv2.MockFailRecvPacketResult
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			freezeClient = false

			packetData := mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// send packet
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, packetData)
			suite.Require().NoError(err)

			err = path.EndpointB.MsgRecvPacket(packet)
			suite.Require().NoError(err)

			tc.malleate()

			packetKey := hostv2.PacketAcknowledgementKey(packet.DestinationChannel, packet.Sequence)
			proof, proofHeight := path.EndpointB.QueryProof(packetKey)

			if freezeClient {
				path.EndpointA.FreezeClient()
			}

			err = suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.AcknowledgePacketTest(suite.chainA.GetContext(), packet, ack, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				commitment := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(suite.chainA.GetContext(), packet.SourceChannel, packet.Sequence)
				suite.Require().Empty(commitment)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestTimeoutPacket() {
	var (
		path         *ibctesting.Path
		packet       types.Packet
		freezeClient bool
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				// send packet
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceChannel,
					packet.TimeoutTimestamp, packet.Data)
				suite.Require().NoError(err, "send packet failed")
			},
			nil,
		},
		{
			"failure: channel not found",
			func() {
				// send packet
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceChannel,
					packet.TimeoutTimestamp, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				packet.SourceChannel = ibctesting.InvalidID
			},
			types.ErrChannelNotFound,
		},
		{
			"failure: counterparty channel identifier different than source channel",
			func() {
				// send packet
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceChannel,
					packet.TimeoutTimestamp, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				packet.DestinationChannel = unusedChannel
			},
			channeltypes.ErrInvalidChannelIdentifier,
		},
		{
			"failure: packet has not timed out yet",
			func() {
				packet.TimeoutTimestamp = uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

				// send packet
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceChannel,
					packet.TimeoutTimestamp, packet.Data)
				suite.Require().NoError(err, "send packet failed")
			},
			channeltypes.ErrTimeoutNotReached,
		},
		{
			"failure: packet already timed out",
			func() {}, // equivalent to not sending packet at all
			channeltypes.ErrNoOpMsg,
		},
		{
			"failure: packet does not match commitment",
			func() {
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceChannel,
					packet.TimeoutTimestamp, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				// try to timeout packet with different data
				packet.Data[0].Payload.Value = []byte("different value")
			},
			channeltypes.ErrInvalidPacket,
		},
		{
			"failure: client status invalid",
			func() {
				// send packet
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceChannel,
					packet.TimeoutTimestamp, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				freezeClient = true
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"failure: verify non-membership failed",
			func() {
				// send packet
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceChannel,
					packet.TimeoutTimestamp, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				// set packet receipt to mock a valid past receive
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationChannel, packet.Sequence)
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			// initialize freezeClient to false
			freezeClient = false

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			// create default packet with a timed out timestamp
			packetData := mockv2.NewMockPacketData(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Unix())

			// test cases may mutate timeout values
			packet = types.NewPacket(1, path.EndpointA.ChannelID, path.EndpointB.ChannelID,
				timeoutTimestamp, packetData)

			tc.malleate()

			// need to update chainA's client representing chainB to prove missing ack
			// commit the changes and update the clients
			suite.coordinator.CommitBlock(path.EndpointA.Chain)
			suite.Require().NoError(path.EndpointB.UpdateClient())
			suite.Require().NoError(path.EndpointA.UpdateClient())

			// get proof of packet receipt absence from chainB
			receiptKey := hostv2.PacketReceiptKey(packet.DestinationChannel, packet.Sequence)
			proof, proofHeight := path.EndpointB.QueryProof(receiptKey)

			if freezeClient {
				path.EndpointA.FreezeClient()
			}

			err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.TimeoutPacketTest(suite.chainA.GetContext(), packet, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				commitment := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(suite.chainA.GetContext(), packet.DestinationChannel, packet.Sequence)
				suite.Require().Nil(commitment, "packet commitment not deleted")
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
