package keeper_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	abci "github.com/cometbft/cometbft/abci/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

var (
	disabledTimeoutTimestamp = uint64(0)
	disabledTimeoutHeight    = clienttypes.ZeroHeight()
	defaultTimeoutHeight     = clienttypes.NewHeight(1, 100)

	// for when the testing package cannot be used
	connIDA = "connA"
	connIDB = "connB"
)

// TestSendPacket tests SendPacket from chainA to chainB
func (s *KeeperTestSuite) TestSendPacket() {
	var (
		path             *ibctesting.Path
		sourcePort       string
		sourceChannel    string
		packetData       []byte
		timeoutHeight    clienttypes.Height
		timeoutTimestamp uint64
	)

	testCases := []testCase{
		{"success: UNORDERED channel", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID
		}, nil},
		{"success: ORDERED channel", func() {
			path.SetChannelOrdered()
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID
		}, nil},
		{"success with solomachine: UNORDERED channel", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			// swap client with solo machine
			solomachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachinesingle", "testing", 1)
			path.EndpointA.ClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 10)
			path.EndpointA.SetClientState(solomachine.ClientState())
			path.EndpointA.UpdateConnection(func(c *connectiontypes.ConnectionEnd) { c.ClientId = path.EndpointA.ClientID })
		}, nil},
		{"success with solomachine: ORDERED channel", func() {
			path.SetChannelOrdered()
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			// swap client with solomachine
			solomachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachinesingle", "testing", 1)
			path.EndpointA.ClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 10)
			path.EndpointA.SetClientState(solomachine.ClientState())

			path.EndpointA.UpdateConnection(func(c *connectiontypes.ConnectionEnd) { c.ClientId = path.EndpointA.ClientID })
		}, nil},
		{"packet basic validation failed, empty packet data", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			packetData = []byte{}
		}, types.ErrInvalidPacket},
		{"channel not found", func() {
			// use wrong channel naming
			path.Setup()
			sourceChannel = ibctesting.InvalidID
		}, types.ErrChannelNotFound},
		{"channel is in CLOSED state", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
		}, types.ErrInvalidChannelState},
		{"channel is in INIT state", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.INIT })
		}, types.ErrInvalidChannelState},
		{"channel is in TRYOPEN stage", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.TRYOPEN })
		}, types.ErrInvalidChannelState},
		{"connection not found", func() {
			// pass channel check
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.ConnectionHops[0] = "invalid-connection" })
		}, errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, "")},
		{"client state not found", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			// change connection client ID
			path.EndpointA.UpdateConnection(func(c *connectiontypes.ConnectionEnd) { c.ClientId = ibctesting.InvalidID })
		}, clienttypes.ErrClientNotActive},
		{"client state is frozen", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			connection := path.EndpointA.GetConnection()
			clientState := path.EndpointA.GetClientState()
			cs, ok := clientState.(*ibctm.ClientState)
			s.Require().True(ok)

			// freeze client
			cs.FrozenHeight = clienttypes.NewHeight(0, 1)
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), connection.ClientId, cs)
		}, clienttypes.ErrClientNotActive},
		{"client state zero height", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			connection := path.EndpointA.GetConnection()
			clientState := path.EndpointA.GetClientState()
			cs, ok := clientState.(*ibctm.ClientState)
			s.Require().True(ok)

			// force a consensus state into the store at height zero to allow client status check to pass.
			consensusState := path.EndpointA.GetConsensusState(cs.LatestHeight)
			path.EndpointA.SetConsensusState(consensusState, clienttypes.ZeroHeight())

			cs.LatestHeight = clienttypes.ZeroHeight()
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), connection.ClientId, cs)
		}, clienttypes.ErrInvalidHeight},
		{"timeout height passed", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			var ok bool
			timeoutHeight, ok = path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			s.Require().True(ok)
		}, types.ErrTimeoutElapsed},
		{"timeout timestamp passed", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			connection := path.EndpointA.GetConnection()
			timestamp, err := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientTimestampAtHeight(s.chainA.GetContext(), connection.ClientId, path.EndpointA.GetClientLatestHeight())
			s.Require().NoError(err)

			timeoutHeight = disabledTimeoutHeight
			timeoutTimestamp = timestamp
		}, types.ErrTimeoutElapsed},
		{"timeout timestamp passed with solomachine", func() {
			path.Setup()
			// swap client with solomachine
			solomachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachinesingle", "testing", 1)
			path.EndpointA.ClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 10)
			path.EndpointA.SetClientState(solomachine.ClientState())

			path.EndpointA.UpdateConnection(func(c *connectiontypes.ConnectionEnd) { c.ClientId = path.EndpointA.ClientID })

			connection := path.EndpointA.GetConnection()
			timestamp, err := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientTimestampAtHeight(s.chainA.GetContext(), connection.ClientId, path.EndpointA.GetClientLatestHeight())
			s.Require().NoError(err)

			sourceChannel = path.EndpointA.ChannelID
			timeoutHeight = disabledTimeoutHeight
			timeoutTimestamp = timestamp
		}, types.ErrTimeoutElapsed},
		{"next sequence send not found", func() {
			path := ibctesting.NewPath(s.chainA, s.chainB)
			sourceChannel = path.EndpointA.ChannelID

			path.SetupConnections()
			// manually creating channel prevents next sequence from being set
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				s.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, path.EndpointA.ChannelConfig.Version),
			)
		}, errorsmod.Wrap(types.ErrSequenceSendNotFound, "")},
	}

	for i, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)

			// set default send packet arguments
			// sourceChannel is set after path is setup
			sourcePort = path.EndpointA.ChannelConfig.PortID
			timeoutHeight = defaultTimeoutHeight
			timeoutTimestamp = disabledTimeoutTimestamp
			packetData = ibctesting.MockPacketData

			// malleate may modify send packet arguments above
			tc.malleate()

			// only check if nextSequenceSend exists in no error case since it is a tested error case above.
			expectedSequence, ok := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceSend(s.chainA.GetContext(), sourcePort, sourceChannel)

			sequence, err := s.chainA.App.GetIBCKeeper().ChannelKeeper.SendPacket(s.chainA.GetContext(),
				sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, packetData)

			if tc.expErr == nil {
				s.Require().NoError(err)
				// verify that the returned sequence matches expected value
				s.Require().True(ok)
				s.Require().Equal(expectedSequence, sequence, "send packet did not return the expected sequence of the outgoing packet")
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestRecvPacket test RecvPacket on chainB. Since packet commitment verification will always
// occur last (resource instensive), only tests expected to succeed and packet commitment
// verification tests need to simulate sending a packet from chainA to chainB.
func (s *KeeperTestSuite) TestRecvPacket() {
	var (
		path   *ibctesting.Path
		packet types.Packet
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: ORDERED channel",
			func() {
				path.SetChannelOrdered()
				path.Setup()

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			nil,
		},
		{
			"success UNORDERED channel",
			func() {
				// setup uses an UNORDERED channel
				path.Setup()
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			nil,
		},
		{
			"success with out of order packet: UNORDERED channel",
			func() {
				// setup uses an UNORDERED channel
				path.Setup()
				// send 2 packets
				_, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				// attempts to receive packet 2 without receiving packet 1
			},
			nil,
		},
		{
			"packet already relayed ORDERED channel (no-op)",
			func() {
				path.SetChannelOrdered()
				path.Setup()

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)
			},
			types.ErrNoOpMsg,
		},
		{
			"packet already relayed UNORDERED channel (no-op)",
			func() {
				// setup uses an UNORDERED channel
				path.Setup()
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)
			},
			types.ErrNoOpMsg,
		},
		{
			"out of order packet failure with ORDERED channel",
			func() {
				path.SetChannelOrdered()
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				// send 2 packets
				_, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				// attempts to receive packet 2 without receiving packet 1
			},
			types.ErrPacketSequenceOutOfOrder,
		},
		{
			"channel not found",
			func() {
				// use wrong channel naming
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			types.ErrChannelNotFound,
		},
		{
			"channel not open",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
			},
			types.ErrInvalidChannelState,
		},
		{
			"packet source port ≠ channel counterparty port",
			func() {
				path.Setup()

				// use wrong port for dest
				packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			types.ErrInvalidPacket,
		},
		{
			"packet source channel ID ≠ channel counterparty channel ID",
			func() {
				path.Setup()

				// use wrong port for dest
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, ibctesting.InvalidID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			types.ErrInvalidPacket,
		},
		{
			"connection not found",
			func() {
				path.Setup()

				// pass channel check
				s.chainB.App.GetIBCKeeper().ChannelKeeper.SetChannel(
					s.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
					types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{connIDB}, path.EndpointB.ChannelConfig.Version),
				)
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			connectiontypes.ErrConnectionNotFound,
		},
		{
			"connection not OPEN",
			func() {
				path.SetupClients()

				// connection on chainB is in INIT
				err := path.EndpointB.ConnOpenInit()
				s.Require().NoError(err)

				// pass channel check
				s.chainB.App.GetIBCKeeper().ChannelKeeper.SetChannel(
					s.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
					types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, path.EndpointB.ChannelConfig.Version),
				)
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			connectiontypes.ErrInvalidConnectionState,
		},
		{
			"timeout height passed",
			func() {
				path.Setup()

				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.GetSelfHeight(s.chainB.GetContext()), disabledTimeoutTimestamp)
			},
			types.ErrTimeoutElapsed,
		},
		{
			"timeout timestamp passed",
			func() {
				path.Setup()

				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, disabledTimeoutHeight, uint64(s.chainB.GetContext().BlockTime().UnixNano()))
			},
			types.ErrTimeoutElapsed,
		},
		{
			"next receive sequence is not found",
			func() {
				path.SetupConnections()

				path.EndpointA.ChannelID = ibctesting.FirstChannelID
				path.EndpointB.ChannelID = ibctesting.FirstChannelID

				// manually creating channel prevents next recv sequence from being set
				s.chainB.App.GetIBCKeeper().ChannelKeeper.SetChannel(
					s.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
					types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, path.EndpointB.ChannelConfig.Version),
				)

				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				// manually set packet commitment
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, packet.GetSequence(), types.CommitPacket(packet))

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)
				err = path.EndpointB.UpdateClient()
				s.Require().NoError(err)
			},
			types.ErrSequenceReceiveNotFound,
		},
		{
			"packet already received",
			func() {
				path.Setup()

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				// set recv seq start to indicate packet was processed in previous upgrade
				s.chainB.App.GetIBCKeeper().ChannelKeeper.SetRecvStartSequence(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sequence+1)
			},
			types.ErrPacketReceived,
		},
		{
			"receipt already stored",
			func() {
				path.Setup()

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)
				s.chainB.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sequence)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			types.ErrNoOpMsg,
		},
		{
			"validation failed",
			func() {
				// packet commitment not set resulting in invalid proof
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			// get proof of packet commitment from chainA
			packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			proof, proofHeight := path.EndpointA.QueryProof(packetKey)

			channelVersion, err := s.chainB.App.GetIBCKeeper().ChannelKeeper.RecvPacket(s.chainB.GetContext(), packet, proof, proofHeight)

			if tc.expError == nil {
				s.Require().NoError(err)
				s.Require().Equal(path.EndpointA.GetChannel().Version, channelVersion, "channel version is incorrect")

				channelB, _ := s.chainB.App.GetIBCKeeper().ChannelKeeper.GetChannel(s.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel())
				nextSeqRecv, found := s.chainB.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(s.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel())
				s.Require().True(found)
				receipt, receiptStored := s.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketReceipt(s.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

				if channelB.Ordering == types.ORDERED {
					s.Require().Equal(packet.GetSequence()+1, nextSeqRecv, "sequence not incremented in ordered channel")
					s.Require().False(receiptStored, "packet receipt stored on ORDERED channel")
				} else {
					s.Require().Equal(uint64(1), nextSeqRecv, "sequence incremented for UNORDERED channel")
					s.Require().True(receiptStored, "packet receipt not stored after RecvPacket in UNORDERED channel")
					s.Require().Equal(string([]byte{byte(1)}), receipt, "packet receipt is not empty string")
				}
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expError)
				s.Require().Equal("", channelVersion)
			}
		})
	}
}

func (s *KeeperTestSuite) TestWriteAcknowledgement() {
	var (
		path   *ibctesting.Path
		ack    exported.Acknowledgement
		packet exported.PacketI
	)

	testCases := []testCase{
		{
			"success",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.MockAcknowledgement
			},
			nil,
		},
		{"channel not found", func() {
			// use wrong channel naming
			path.Setup()
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			ack = ibcmock.MockAcknowledgement
		}, types.ErrChannelNotFound},
		{"channel not open", func() {
			path.Setup()
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			ack = ibcmock.MockAcknowledgement

			path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
		}, types.ErrInvalidChannelState},
		{
			"no-op, already acked",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.MockAcknowledgement
				s.chainB.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(s.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(), ack.Acknowledgement())
			},
			types.ErrAcknowledgementExists,
		},
		{
			"empty acknowledgement",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.NewEmptyAcknowledgement()
			},
			errorsmod.Wrap(types.ErrInvalidAcknowledgement, ""),
		},
		{
			"acknowledgement is nil",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = nil
			},
			errorsmod.Wrap(types.ErrInvalidAcknowledgement, ""),
		},
		{
			"packet already received",
			func() {
				path.Setup()

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.MockAcknowledgement

				// set recv seq start to indicate packet was processed in previous upgrade
				s.chainB.App.GetIBCKeeper().ChannelKeeper.SetRecvStartSequence(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sequence+1)
			},
			errorsmod.Wrap(types.ErrPacketReceived, ""),
		},
	}
	for i, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			err := s.chainB.App.GetIBCKeeper().ChannelKeeper.WriteAcknowledgement(s.chainB.GetContext(), packet, ack)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestAcknowledgePacket tests the call AcknowledgePacket on chainA.
func (s *KeeperTestSuite) TestAcknowledgePacket() {
	var (
		path   *ibctesting.Path
		packet types.Packet
		ack    []byte
	)

	assertErr := func(errType *errorsmod.Error) func(commitment []byte, channelVersion string, err error) {
		return func(commitment []byte, channelVersion string, err error) {
			s.Require().Error(err)
			s.Require().ErrorIs(err, errType)
			s.Require().NotNil(commitment)
			s.Require().Equal("", channelVersion)
		}
	}

	assertNoOp := func(commitment []byte, channelVersion string, err error) {
		s.Require().Error(err)
		s.Require().ErrorIs(err, types.ErrNoOpMsg)
		s.Require().Nil(commitment)
		s.Require().Equal("", channelVersion)
	}

	assertSuccess := func(seq func() uint64, msg string) func(commitment []byte, channelVersion string, err error) {
		return func(commitment []byte, channelVersion string, err error) {
			s.Require().NoError(err)
			s.Require().Nil(commitment)
			s.Require().Equal(path.EndpointA.GetChannel().Version, channelVersion)

			nextSequenceAck, found := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceAck(s.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel())

			s.Require().True(found)
			s.Require().Equal(seq(), nextSequenceAck, msg)
		}
	}

	testCases := []struct {
		name      string
		malleate  func()
		expResult func(commitment []byte, channelVersion string, err error)
		expEvents func(path *ibctesting.Path) []abci.Event
	}{
		{
			name: "success on ordered channel",
			malleate: func() {
				path.SetChannelOrdered()
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)
			},
			expResult: assertSuccess(func() uint64 { return packet.GetSequence() + 1 }, "sequence not incremented in ordered channel"),
		},
		{
			name: "success on unordered channel",
			malleate: func() {
				// setup uses an UNORDERED channel
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)
			},
			expResult: assertSuccess(func() uint64 { return uint64(1) }, "sequence incremented for UNORDERED channel"),
		},
		{
			name: "packet already acknowledged ordered channel (no-op)",
			malleate: func() {
				path.SetChannelOrdered()
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)

				err = path.EndpointA.AcknowledgePacket(packet, ack)
				s.Require().NoError(err)
			},
			expResult: assertNoOp,
		},
		{
			name: "packet already acknowledged unordered channel (no-op)",
			malleate: func() {
				// setup uses an UNORDERED channel
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)

				err = path.EndpointA.AcknowledgePacket(packet, ack)
				s.Require().NoError(err)
			},
			expResult: assertNoOp,
		},
		{
			name: "fake acknowledgement",
			malleate: func() {
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// create packet
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				// write packet acknowledgement directly
				// Create a valid acknowledgement using deterministic serialization.
				ack = types.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement()
				// Introduce non-determinism: insert an extra space after the first character '{'
				// This will deserialize correctly but fail to re-serialize to the expected bytes.
				if len(ack) > 0 && ack[0] == '{' {
					ack = []byte("{ " + string(ack[1:]))
				}
				path.EndpointB.Chain.Coordinator.UpdateTimeForChain(path.EndpointB.Chain)

				path.EndpointB.Chain.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(path.EndpointB.Chain.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sequence, types.CommitAcknowledgement(ack))

				path.EndpointB.Chain.NextBlock()
				s.Require().NoError(path.EndpointA.UpdateClient())
			},
			expResult: func(commitment []byte, channelVersion string, err error) {
				s.Require().Error(err)
				s.Require().ErrorIs(err, types.ErrInvalidAcknowledgement)
				s.Require().Equal("", channelVersion)
				s.Require().NotNil(commitment)
			},
		},
		{
			name: "non-standard acknowledgement",
			malleate: func() {
				// setup uses an UNORDERED channel
				s.coordinator.Setup(path)

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// create packet
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				// write packet acknowledgement directly
				ack = []byte(`{"somethingelse":"anything"}`)
				path.EndpointB.Chain.Coordinator.UpdateTimeForChain(path.EndpointB.Chain)

				path.EndpointB.Chain.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(path.EndpointB.Chain.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sequence, types.CommitAcknowledgement(ack))

				path.EndpointB.Chain.NextBlock()
				s.Require().NoError(path.EndpointA.UpdateClient())
			},
			expResult: func(commitment []byte, channelVersion string, err error) {
				s.Require().NoError(err)
				channel := path.EndpointA.GetChannel()
				s.Require().Equal(channel.Version, channelVersion)
				s.Require().Nil(commitment)
			},
		},
		{
			name: "channel not found",
			malleate: func() {
				// use wrong channel naming
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, ibctesting.InvalidID, ibctesting.InvalidID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			expResult: assertErr(types.ErrChannelNotFound),
		},
		{
			name: "channel not open",
			malleate: func() {
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
			},
			expResult: assertErr(types.ErrInvalidChannelState),
		},
		{
			name: "packet destination port ≠ channel counterparty port",
			malleate: func() {
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// use wrong port for dest
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			expResult: assertErr(types.ErrInvalidPacket),
		},
		{
			name: "packet destination channel ID ≠ channel counterparty channel ID",
			malleate: func() {
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// use wrong channel for dest
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			expResult: assertErr(types.ErrInvalidPacket),
		},
		{
			name: "connection not found",
			malleate: func() {
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				// pass channel check
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
					s.chainA.GetContext(),
					path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
					types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{"connection-1000"}, path.EndpointA.GetChannel().Version),
				)
			},
			expResult: assertErr(connectiontypes.ErrConnectionNotFound),
		},
		{
			name: "connection not OPEN",
			malleate: func() {
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				// connection on chainA is in INIT
				err = path.EndpointA.ConnOpenInit()
				s.Require().NoError(err)

				// pass channel check
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
					s.chainA.GetContext(),
					path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
					types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, path.EndpointA.GetChannel().Version),
				)
			},
			expResult: assertErr(connectiontypes.ErrInvalidConnectionState),
		},
		{
			name: "packet hasn't been sent",
			malleate: func() {
				// packet commitment never written
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			expResult: assertNoOp, // NOTE: ibc core does not distinguish between unsent and already relayed packets.
		},
		{
			name: "packet ack verification failed",
			malleate: func() {
				// skip error code check since error occurs in light-clients

				// ack never written
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			},
			expResult: assertErr(commitmenttypes.ErrInvalidProof),
		},
		{
			name: "packet commitment bytes do not match",
			malleate: func() {
				// setup uses an UNORDERED channel
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)

				packet.Data = []byte("invalid packet commitment")
			},
			expResult: assertErr(types.ErrInvalidPacket),
		},
		{
			name: "next ack sequence not found",
			malleate: func() {
				path.SetChannelOrdered()
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)

				// manually delete the next sequence ack in the ibc store
				storeKey := s.chainA.GetSimApp().GetKey(exported.ModuleName)
				ibcStore := s.chainA.GetContext().KVStore(storeKey)

				ibcStore.Delete(host.NextSequenceAckKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			},
			expResult: assertErr(types.ErrSequenceAckNotFound),
		},
		{
			name: "next ack sequence mismatch ORDERED",
			malleate: func() {
				path.SetChannelOrdered()
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// create packet acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)

				// set next sequence ack wrong
				s.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceAck(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 10)
			},
			expResult: assertErr(types.ErrPacketSequenceOutOfOrder),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			// reset ack
			ack = ibcmock.MockAcknowledgement.Acknowledgement()

			path = ibctesting.NewPath(s.chainA, s.chainB)
			ctx := s.chainA.GetContext()

			tc.malleate()

			packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			proof, proofHeight := path.EndpointB.QueryProof(packetKey)

			channelVersion, err := s.chainA.App.GetIBCKeeper().ChannelKeeper.AcknowledgePacket(ctx, packet, ack, proof, proofHeight)

			commitment := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(ctx, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, packet.GetSequence())
			tc.expResult(commitment, channelVersion, err)
			if tc.expEvents != nil {
				events := ctx.EventManager().ABCIEvents()

				expEvents := tc.expEvents(path)

				ibctesting.AssertEvents(&s.Suite, expEvents, events)
			}
		})
	}
}
