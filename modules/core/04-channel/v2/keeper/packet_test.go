package keeper_test

import (
	"fmt"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

var unusedChannel = "channel-5"

func (s *KeeperTestSuite) TestSendPacket() {
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
				_, _, err := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(s.chainA.GetContext(), packet.SourceClient, packet.TimeoutTimestamp, packet.Payloads)
				s.Require().NoError(err)
				expSequence = 2
			},
			nil,
		},
		{
			"client not found",
			func() {
				packet.SourceClient = ibctesting.InvalidID
			},
			clientv2types.ErrCounterpartyNotFound,
		},
		{
			"packet failed basic validation",
			func() {
				// invalid data
				packet.Payloads[0].SourcePort = ""
			},
			types.ErrInvalidPacket,
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
				s.Require().True(ok)

				// force a consensus state into the store at height zero to allow client status check to pass.
				consensusState := path.EndpointA.GetConsensusState(cs.LatestHeight)
				path.EndpointA.SetConsensusState(consensusState, clienttypes.ZeroHeight())

				cs.LatestHeight = clienttypes.ZeroHeight()
				s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), path.EndpointA.ClientID, cs)
			},
			clienttypes.ErrInvalidHeight,
		},
		{
			"timeout equal to sending chain blocktime", func() {
				packet.TimeoutTimestamp = uint64(s.chainA.GetContext().BlockTime().Unix())
			},
			types.ErrTimeoutElapsed,
		},
		{
			"timeout elapsed", func() {
				packet.TimeoutTimestamp = 1
			},
			types.ErrTimeoutElapsed,
		},
	}

	for i, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.name, i, len(testCases)), func() {
			s.SetupTest() // reset

			// create clients and set counterparties on both chains
			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// create standard packet that can be malleated
			packet = types.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID,
				timeoutTimestamp, payload)
			expSequence = 1

			// malleate the test case
			tc.malleate()

			// send packet
			seq, destClient, err := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(s.chainA.GetContext(), packet.SourceClient, packet.TimeoutTimestamp, packet.Payloads)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
				// verify send packet method instantiated packet with correct sequence and destination channel
				s.Require().Equal(expSequence, seq)
				s.Require().Equal(path.EndpointB.ClientID, destClient)
				// verify send packet stored the packet commitment correctly
				expCommitment := types.CommitPacket(packet)
				s.Require().Equal(expCommitment, s.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(s.chainA.GetContext(), packet.SourceClient, seq))
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expError)
				s.Require().Equal(uint64(0), seq)
				s.Require().Nil(s.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(s.chainA.GetContext(), packet.SourceClient, seq))
			}
		})
	}
}

func (s *KeeperTestSuite) TestRecvPacket() {
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
			"failure: client not found",
			func() {
				packet.DestinationClient = ibctesting.InvalidID
			},
			clientv2types.ErrCounterpartyNotFound,
		},
		{
			"failure: client is not active",
			func() {
				path.EndpointB.FreezeClient()
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"failure: counterparty client identifier different than source client",
			func() {
				packet.SourceClient = unusedChannel
			},
			clientv2types.ErrInvalidCounterparty,
		},
		{
			"failure: packet has timed out",
			func() {
				s.coordinator.IncrementTimeBy(time.Hour * 20)
			},
			types.ErrTimeoutElapsed,
		},
		{
			"failure: packet already received",
			func() {
				s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
			},
			types.ErrNoOpMsg,
		},
		{
			"failure: verify membership failed",
			func() {
				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(s.chainA.GetContext(), packet.SourceClient, packet.Sequence, []byte(""))
				s.coordinator.CommitBlock(path.EndpointA.Chain)
				s.Require().NoError(path.EndpointB.UpdateClient())
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// send packet
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
			s.Require().NoError(err)

			tc.malleate()

			// get proof of v2 packet commitment from chainA
			packetKey := hostv2.PacketCommitmentKey(packet.GetSourceClient(), packet.GetSequence())
			proof, proofHeight := path.EndpointA.QueryProof(packetKey)

			err = s.chainB.App.GetIBCKeeper().ChannelKeeperV2.RecvPacketTest(s.chainB.GetContext(), packet, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)

				_, found := s.chainB.App.GetIBCKeeper().ChannelKeeperV2.GetPacketReceipt(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
				s.Require().True(found)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestWriteAcknowledgement() {
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
			"failure: client not found",
			func() {
				packet.DestinationClient = ibctesting.InvalidID
				s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
				s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetAsyncPacket(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence, packet)
			},
			clientv2types.ErrCounterpartyNotFound,
		},
		{
			"failure: counterparty client identifier different than source client",
			func() {
				packet.SourceClient = unusedChannel
				s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
				s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetAsyncPacket(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence, packet)
			},
			clientv2types.ErrInvalidCounterparty,
		},
		{
			"failure: ack already exists",
			func() {
				ackBz := types.CommitAcknowledgement(ack)
				s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketAcknowledgement(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence, ackBz)
			},
			types.ErrAcknowledgementExists,
		},
		{
			"failure: receipt not found for packet",
			func() {
				packet.Sequence = 2
				s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetAsyncPacket(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence, packet)
			},
			types.ErrInvalidPacket,
		},
		{
			"failure: async packet not found",
			func() {
				s.chainB.App.GetIBCKeeper().ChannelKeeperV2.DeleteAsyncPacket(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
			},
			types.ErrInvalidAcknowledgement,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path := ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// create standard packet that can be malleated
			packet = types.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID,
				timeoutTimestamp, payload)

			// create standard ack that can be malleated
			ack = types.Acknowledgement{
				AppAcknowledgements: [][]byte{mockv2.MockRecvPacketResult.Acknowledgement},
			}

			// mock receive with async acknowledgement
			s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
			s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetAsyncPacket(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence, packet)

			tc.malleate()

			err := s.chainB.App.GetIBCKeeper().ChannelKeeperV2.WriteAcknowledgement(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence, ack)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)

				ackCommitment := s.chainB.App.GetIBCKeeper().ChannelKeeperV2.GetPacketAcknowledgement(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
				s.Require().Equal(types.CommitAcknowledgement(ack), ackCommitment)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestAcknowledgePacket() {
	var (
		packet types.Packet
		err    error
		ack    = types.Acknowledgement{
			AppAcknowledgements: [][]byte{mockv2.MockRecvPacketResult.Acknowledgement},
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
			"failure: client not found",
			func() {
				packet.SourceClient = ibctesting.InvalidID
			},
			clientv2types.ErrCounterpartyNotFound,
		},
		{
			"failure: counterparty client identifier different than destination client",
			func() {
				packet.DestinationClient = unusedChannel
			},
			clientv2types.ErrInvalidCounterparty,
		},
		{
			"failure: packet commitment doesn't exist.",
			func() {
				s.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeletePacketCommitment(s.chainA.GetContext(), packet.SourceClient, packet.Sequence)
			},
			types.ErrNoOpMsg,
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
				// change payload after send to acknowledge different packet
				packet.Payloads[0].Value = []byte("different value")
			},
			types.ErrInvalidPacket,
		},
		{
			"failure: verify membership fails",
			func() {
				ack.AppAcknowledgements[0] = types.ErrorAcknowledgement[:]
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path := ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			freezeClient = false

			payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// send packet
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
			s.Require().NoError(err)

			err = path.EndpointB.MsgRecvPacket(packet)
			s.Require().NoError(err)

			tc.malleate()

			packetKey := hostv2.PacketAcknowledgementKey(packet.DestinationClient, packet.Sequence)
			proof, proofHeight := path.EndpointB.QueryProof(packetKey)

			if freezeClient {
				path.EndpointA.FreezeClient()
			}

			err = s.chainA.App.GetIBCKeeper().ChannelKeeperV2.AcknowledgePacketTest(s.chainA.GetContext(), packet, ack, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)

				commitment := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(s.chainA.GetContext(), packet.SourceClient, packet.Sequence)
				s.Require().Empty(commitment)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestTimeoutPacket() {
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
				_, _, err := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(s.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				s.Require().NoError(err, "send packet failed")
			},
			nil,
		},
		{
			"failure: client not found",
			func() {
				// send packet
				_, _, err := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(s.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				s.Require().NoError(err, "send packet failed")

				packet.SourceClient = ibctesting.InvalidID
			},
			clientv2types.ErrCounterpartyNotFound,
		},
		{
			"failure: counterparty client identifier different than destination client",
			func() {
				// send packet
				_, _, err := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(s.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				s.Require().NoError(err, "send packet failed")

				packet.DestinationClient = unusedChannel
			},
			clientv2types.ErrInvalidCounterparty,
		},
		{
			"failure: packet has not timed out yet",
			func() {
				packet.TimeoutTimestamp = uint64(s.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

				// send packet
				_, _, err := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(s.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				s.Require().NoError(err, "send packet failed")
			},
			types.ErrTimeoutNotReached,
		},
		{
			"failure: packet already timed out",
			func() {}, // equivalent to not sending packet at all
			types.ErrNoOpMsg,
		},
		{
			"failure: packet does not match commitment",
			func() {
				_, _, err := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(s.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				s.Require().NoError(err, "send packet failed")

				// try to timeout packet with different data
				packet.Payloads[0].Value = []byte("different value")
			},
			types.ErrInvalidPacket,
		},
		{
			"failure: client status invalid",
			func() {
				// send packet
				_, _, err := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(s.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				s.Require().NoError(err, "send packet failed")

				freezeClient = true
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"failure: verify non-membership failed",
			func() {
				// send packet
				_, _, err := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(s.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				s.Require().NoError(err, "send packet failed")

				// set packet receipt to mock a valid past receive
				s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(s.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			// initialize freezeClient to false
			freezeClient = false

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupV2()

			// create default packet with a timed out timestamp
			payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			// make timeoutTimestamp 1 second more than sending chain time to ensure it passes SendPacket
			// and times out successfully after update
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Second).Unix())

			// test cases may mutate timeout values
			packet = types.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID,
				timeoutTimestamp, payload)

			tc.malleate()

			// need to update chainA's client representing chainB to prove missing ack
			// commit the changes and update the clients
			s.coordinator.CommitBlock(path.EndpointA.Chain)
			s.Require().NoError(path.EndpointB.UpdateClient())
			s.Require().NoError(path.EndpointA.UpdateClient())

			// get proof of packet receipt absence from chainB
			receiptKey := hostv2.PacketReceiptKey(packet.DestinationClient, packet.Sequence)
			proof, proofHeight := path.EndpointB.QueryProof(receiptKey)

			if freezeClient {
				path.EndpointA.FreezeClient()
			}

			err := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.TimeoutPacketTest(s.chainA.GetContext(), packet, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)

				commitment := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(s.chainA.GetContext(), packet.DestinationClient, packet.Sequence)
				s.Require().Nil(commitment, "packet commitment not deleted")
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
