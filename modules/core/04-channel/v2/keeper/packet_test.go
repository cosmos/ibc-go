package keeper_test

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	v11 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v11"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
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
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceClient, packet.TimeoutTimestamp, packet.Payloads)
				suite.Require().NoError(err)
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
			"timeout equal to sending chain blocktime", func() {
				packet.TimeoutTimestamp = uint64(suite.chainA.GetContext().BlockTime().Unix())
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
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.name, i, len(testCases)), func() {
			suite.SetupTest() // reset

			// create clients and set counterparties on both chains
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// create standard packet that can be malleated
			packet = types.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID,
				timeoutTimestamp, payload)
			expSequence = 1

			// malleate the test case
			tc.malleate()

			// send packet
			seq, destClient, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceClient, packet.TimeoutTimestamp, packet.Payloads)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				// verify send packet method instantiated packet with correct sequence and destination channel
				suite.Require().Equal(expSequence, seq)
				suite.Require().Equal(path.EndpointB.ClientID, destClient)
				// verify send packet stored the packet commitment correctly
				expCommitment := types.CommitPacket(packet)
				suite.Require().Equal(expCommitment, suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(suite.chainA.GetContext(), packet.SourceClient, seq))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Equal(uint64(0), seq)
				suite.Require().Nil(suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(suite.chainA.GetContext(), packet.SourceClient, seq))

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
				suite.coordinator.IncrementTimeBy(time.Hour * 20)
			},
			types.ErrTimeoutElapsed,
		},
		{
			"failure: packet already received",
			func() {
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
			},
			types.ErrNoOpMsg,
		},
		{
			"failure: verify membership failed",
			func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SetPacketCommitment(suite.chainA.GetContext(), packet.SourceClient, packet.Sequence, []byte(""))
				suite.coordinator.CommitBlock(path.EndpointA.Chain)
				suite.Require().NoError(path.EndpointB.UpdateClient())
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// send packet
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
			suite.Require().NoError(err)

			tc.malleate()

			// get proof of v2 packet commitment from chainA
			packetKey := hostv2.PacketCommitmentKey(packet.GetSourceClient(), packet.GetSequence())
			proof, proofHeight := path.EndpointA.QueryProof(packetKey)

			err = suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.RecvPacketTest(suite.chainB.GetContext(), packet, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				_, found := suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.GetPacketReceipt(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
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
			"failure: client not found",
			func() {
				packet.DestinationClient = ibctesting.InvalidID
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetAsyncPacket(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence, packet)
			},
			clientv2types.ErrCounterpartyNotFound,
		},
		{
			"failure: counterparty client identifier different than source client",
			func() {
				packet.SourceClient = unusedChannel
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetAsyncPacket(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence, packet)
			},
			clientv2types.ErrInvalidCounterparty,
		},
		{
			"failure: ack already exists",
			func() {
				ackBz := types.CommitAcknowledgement(ack)
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketAcknowledgement(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence, ackBz)
			},
			types.ErrAcknowledgementExists,
		},
		{
			"failure: receipt not found for packet",
			func() {
				packet.Sequence = 2
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetAsyncPacket(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence, packet)
			},
			types.ErrInvalidPacket,
		},
		{
			"failure: async packet not found",
			func() {
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.DeleteAsyncPacket(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
			},
			types.ErrInvalidAcknowledgement,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// create standard packet that can be malleated
			packet = types.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID,
				timeoutTimestamp, payload)

			// create standard ack that can be malleated
			ack = types.Acknowledgement{
				AppAcknowledgements: [][]byte{mockv2.MockRecvPacketResult.Acknowledgement},
			}

			// mock receive with async acknowledgement
			suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
			suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetAsyncPacket(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence, packet)

			tc.malleate()

			err := suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.WriteAcknowledgement(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence, ack)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				ackCommitment := suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
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
				suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.DeletePacketCommitment(suite.chainA.GetContext(), packet.SourceClient, packet.Sequence)
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
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			freezeClient = false

			payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

			// send packet
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
			suite.Require().NoError(err)

			err = path.EndpointB.MsgRecvPacket(packet)
			suite.Require().NoError(err)

			tc.malleate()

			packetKey := hostv2.PacketAcknowledgementKey(packet.DestinationClient, packet.Sequence)
			proof, proofHeight := path.EndpointB.QueryProof(packetKey)

			if freezeClient {
				path.EndpointA.FreezeClient()
			}

			err = suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.AcknowledgePacketTest(suite.chainA.GetContext(), packet, ack, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				commitment := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(suite.chainA.GetContext(), packet.SourceClient, packet.Sequence)
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
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				suite.Require().NoError(err, "send packet failed")
			},
			nil,
		},
		{
			"failure: client not found",
			func() {
				// send packet
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				suite.Require().NoError(err, "send packet failed")

				packet.SourceClient = ibctesting.InvalidID
			},
			clientv2types.ErrCounterpartyNotFound,
		},
		{
			"failure: counterparty client identifier different than destination client",
			func() {
				// send packet
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				suite.Require().NoError(err, "send packet failed")

				packet.DestinationClient = unusedChannel
			},
			clientv2types.ErrInvalidCounterparty,
		},
		{
			"failure: packet has not timed out yet",
			func() {
				packet.TimeoutTimestamp = uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).Unix())

				// send packet
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				suite.Require().NoError(err, "send packet failed")
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
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				suite.Require().NoError(err, "send packet failed")

				// try to timeout packet with different data
				packet.Payloads[0].Value = []byte("different value")
			},
			types.ErrInvalidPacket,
		},
		{
			"failure: client status invalid",
			func() {
				// send packet
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				suite.Require().NoError(err, "send packet failed")

				freezeClient = true
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"failure: verify non-membership failed",
			func() {
				// send packet
				_, _, err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.SendPacketTest(suite.chainA.GetContext(), packet.SourceClient,
					packet.TimeoutTimestamp, packet.Payloads)
				suite.Require().NoError(err, "send packet failed")

				// set packet receipt to mock a valid past receive
				suite.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationClient, packet.Sequence)
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			// initialize freezeClient to false
			freezeClient = false

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			// create default packet with a timed out timestamp
			payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

			// make timeoutTimestamp 1 second more than sending chain time to ensure it passes SendPacket
			// and times out successfully after update
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Add(time.Second).Unix())

			// test cases may mutate timeout values
			packet = types.NewPacket(1, path.EndpointA.ClientID, path.EndpointB.ClientID,
				timeoutTimestamp, payload)

			tc.malleate()

			// need to update chainA's client representing chainB to prove missing ack
			// commit the changes and update the clients
			suite.coordinator.CommitBlock(path.EndpointA.Chain)
			suite.Require().NoError(path.EndpointB.UpdateClient())
			suite.Require().NoError(path.EndpointA.UpdateClient())

			// get proof of packet receipt absence from chainB
			receiptKey := hostv2.PacketReceiptKey(packet.DestinationClient, packet.Sequence)
			proof, proofHeight := path.EndpointB.QueryProof(receiptKey)

			if freezeClient {
				path.EndpointA.FreezeClient()
			}

			err := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.TimeoutPacketTest(suite.chainA.GetContext(), packet, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				commitment := suite.chainA.App.GetIBCKeeper().ChannelKeeperV2.GetPacketCommitment(suite.chainA.GetContext(), packet.DestinationClient, packet.Sequence)
				suite.Require().Nil(commitment, "packet commitment not deleted")
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestAliasedChannel() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.Setup()

	// mock v1 format for both sides of the channel
	mockV1Format(path.EndpointA)
	mockV1Format(path.EndpointB)

	// migrate the store for both chains
	err := v11.MigrateStore(suite.chainA.GetContext(), runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey)), suite.chainA.App.AppCodec(), suite.chainA.App.GetIBCKeeper())
	suite.Require().NoError(err, "migrate store failed for chain A")
	err = v11.MigrateStore(suite.chainB.GetContext(), runtime.NewKVStoreService(suite.chainB.GetSimApp().GetKey(ibcexported.StoreKey)), suite.chainB.App.AppCodec(), suite.chainB.App.GetIBCKeeper())
	suite.Require().NoError(err, "migrate store failed for chain B")

	// create v2 path from the original client ids
	// the path config is only used for updating
	// the packet client ids will be the original channel identifiers
	// but they are not validated against the client ids in the path in the tests
	pathv2 := ibctesting.NewPath(suite.chainA, suite.chainB)
	pathv2.EndpointA.ClientID = path.EndpointA.ClientID
	pathv2.EndpointB.ClientID = path.EndpointB.ClientID

	// send a v1 packet on the channel id
	// create default packet with a timed out timestamp
	payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

	// create a timeout timestamp that is 1 hour in the future
	timeoutTimestamp := uint64(suite.chainA.GetContext().BlockTime().Add(time.Hour).Unix())
	timeoutTimestampNano := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).UnixNano())

	// send v1 packet
	sequence, err := path.EndpointA.SendPacket(clienttypes.Height{}, timeoutTimestampNano, ibctesting.MockPacketData)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(1), sequence, "sequence should be 1 for first packet on channel")
	packetv1 := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.Height{}, timeoutTimestampNano)

	// relay v1 packet
	err = path.RelayPacket(packetv1)
	suite.Require().NoError(err, "relay v1 packet failed")

	// send v2 packet
	msgSendPacket := types.NewMsgSendPacket(
		path.EndpointA.ChannelID,
		timeoutTimestamp,
		path.EndpointA.Chain.SenderAccount.GetAddress().String(),
		payload,
	)
	res, err := path.EndpointA.Chain.SendMsgs(msgSendPacket)
	suite.Require().NoError(err, "send v2 packet failed")

	packetv2, err := ibctesting.ParseV2PacketFromEvents(res.Events)
	suite.Require().NoError(err, "parse v2 packet from events failed")
	suite.Require().Equal(uint64(2), packetv2.Sequence, "sequence should be incremented across protocol versions")

	err = path.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// relay v2 packet
	err = pathv2.EndpointA.RelayPacket(packetv2)
	suite.Require().NoError(err)

	// send v1 packet again
	sequence, err = path.EndpointA.SendPacket(clienttypes.Height{}, timeoutTimestampNano, ibctesting.MockPacketData)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(3), sequence, "sequence should be 3 for first packet on channel")
	packetv1 = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.Height{}, timeoutTimestampNano)

	// relay v1 packet again
	err = path.RelayPacket(packetv1)
	suite.Require().NoError(err, "relay v1 packet failed")
}

func (suite *KeeperTestSuite) TestPostMigrationAliasing() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.Setup()

	// ensure we can send a v2 packet on the channel automatically
	// after v1 channel handshake completes
	// create v2 path from the original client ids
	// the path config is only used for updating
	// the packet client ids will be the original channel identifiers
	// but they are not validated against the client ids in the path in the tests
	pathv2 := ibctesting.NewPath(suite.chainA, suite.chainB)
	pathv2.EndpointA.ClientID = path.EndpointA.ClientID
	pathv2.EndpointB.ClientID = path.EndpointB.ClientID

	// send a v1 packet on the channel id
	// create default packet with a timed out timestamp
	payload := mockv2.NewMockPayload(mockv2.ModuleNameA, mockv2.ModuleNameB)

	// create a timeout timestamp that is 1 hour in the future
	timeoutTimestamp := uint64(suite.chainA.GetContext().BlockTime().Add(time.Hour).Unix())
	timeoutTimestampNano := uint64(suite.chainB.GetContext().BlockTime().Add(time.Hour).UnixNano())

	// send v1 packet
	sequence, err := path.EndpointA.SendPacket(clienttypes.Height{}, timeoutTimestampNano, ibctesting.MockPacketData)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(1), sequence, "sequence should be 1 for first packet on channel")
	packetv1 := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.Height{}, timeoutTimestampNano)

	// relay v1 packet
	err = path.RelayPacket(packetv1)
	suite.Require().NoError(err, "relay v1 packet failed")

	// send v2 packet
	msgSendPacket := types.NewMsgSendPacket(
		path.EndpointA.ChannelID,
		timeoutTimestamp,
		path.EndpointA.Chain.SenderAccount.GetAddress().String(),
		payload,
	)
	res, err := path.EndpointA.Chain.SendMsgs(msgSendPacket)
	suite.Require().NoError(err, "send v2 packet failed")

	packetv2, err := ibctesting.ParseV2PacketFromEvents(res.Events)
	suite.Require().NoError(err, "parse v2 packet from events failed")
	suite.Require().Equal(uint64(2), packetv2.Sequence, "sequence should be incremented across protocol versions")

	err = path.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// relay v2 packet
	err = pathv2.EndpointA.RelayPacket(packetv2)
	suite.Require().NoError(err)

}

func mockV1Format(endpoint *ibctesting.Endpoint) {
	// mock v1 format by setting the sequence in the old key
	seq, ok := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceSend(endpoint.Chain.GetContext(), endpoint.ChannelConfig.PortID, endpoint.ChannelID)
	if !ok {
		panic("sequence not found")
	}

	// move the next sequence send back to the old v1 format key
	// so we can migrate it in our tests
	storeService := runtime.NewKVStoreService(endpoint.Chain.GetSimApp().GetKey(ibcexported.StoreKey))
	store := storeService.OpenKVStore(endpoint.Chain.GetContext())
	store.Set(v11.NextSequenceSendKey(endpoint.ChannelConfig.PortID, endpoint.ChannelID), sdk.Uint64ToBigEndian(seq))
	store.Delete(hostv2.NextSequenceSendKey(endpoint.ChannelID))

	// Remove counterparty to mock pre migration channels
	clientStore := endpoint.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(endpoint.Chain.GetContext(), endpoint.ChannelID)
	clientStore.Delete(clientv2types.CounterpartyKey())
}
