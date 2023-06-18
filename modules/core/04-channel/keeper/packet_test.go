package keeper_test

import (
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
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
		channelCap       *capabilitytypes.Capability
	)

	testCases := []testCase{
		{"success: UNORDERED channel", func() {
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"success: ORDERED channel", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"success with solomachine: UNORDERED channel", func() {
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			// swap client with solo machine
			solomachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachinesingle", "testing", 1)
			path.EndpointA.ClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 10)
			path.EndpointA.SetClientState(solomachine.ClientState())
			connection := path.EndpointA.GetConnection()
			connection.ClientId = path.EndpointA.ClientID
			path.EndpointA.SetConnection(connection)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"success with solomachine: ORDERED channel", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			// swap client with solomachine
			solomachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachinesingle", "testing", 1)
			path.EndpointA.ClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 10)
			path.EndpointA.SetClientState(solomachine.ClientState())
			connection := path.EndpointA.GetConnection()
			connection.ClientId = path.EndpointA.ClientID
			path.EndpointA.SetConnection(connection)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"packet basic validation failed, empty packet data", func() {
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			packetData = []byte{}
		}, false},
		{"channel not found", func() {
			// use wrong channel naming
			s.coordinator.Setup(path)
			sourceChannel = ibctesting.InvalidID
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"channel is in CLOSED state", func() {
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			err := path.EndpointA.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
		}, false},
		{"channel is in INIT state", func() {
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			err := path.EndpointA.SetChannelState(types.INIT)
			s.Require().NoError(err)
		}, false},
		{"channel is in TRYOPEN stage", func() {
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			err := path.EndpointA.SetChannelState(types.TRYOPEN)
			s.Require().NoError(err)
		}, false},
		{"connection not found", func() {
			// pass channel check
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			channel := path.EndpointA.GetChannel()
			channel.ConnectionHops[0] = "invalid-connection"
			path.EndpointA.SetChannel(channel)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"client state not found", func() {
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			// change connection client ID
			connection := path.EndpointA.GetConnection()
			connection.ClientId = ibctesting.InvalidID
			s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(s.chainA.GetContext(), path.EndpointA.ConnectionID, connection)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"client state is frozen", func() {
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			connection := path.EndpointA.GetConnection()
			clientState := path.EndpointA.GetClientState()
			cs, ok := clientState.(*ibctm.ClientState)
			s.Require().True(ok)

			// freeze client
			cs.FrozenHeight = clienttypes.NewHeight(0, 1)
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(s.chainA.GetContext(), connection.ClientId, cs)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},

		{"timeout height passed", func() {
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			// use client state latest height for timeout
			clientState := path.EndpointA.GetClientState()
			timeoutHeight = clientState.GetLatestHeight().(clienttypes.Height)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"timeout timestamp passed", func() {
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			// use latest time on client state
			clientState := path.EndpointA.GetClientState()
			connection := path.EndpointA.GetConnection()
			timestamp, err := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetTimestampAtHeight(s.chainA.GetContext(), connection, clientState.GetLatestHeight())
			s.Require().NoError(err)

			timeoutHeight = disabledTimeoutHeight
			timeoutTimestamp = timestamp
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"timeout timestamp passed with solomachine", func() {
			s.coordinator.Setup(path)
			// swap client with solomachine
			solomachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachinesingle", "testing", 1)
			path.EndpointA.ClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 10)
			path.EndpointA.SetClientState(solomachine.ClientState())
			connection := path.EndpointA.GetConnection()
			connection.ClientId = path.EndpointA.ClientID
			path.EndpointA.SetConnection(connection)

			clientState := path.EndpointA.GetClientState()
			timestamp, err := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetTimestampAtHeight(s.chainA.GetContext(), connection, clientState.GetLatestHeight())
			s.Require().NoError(err)

			sourceChannel = path.EndpointA.ChannelID
			timeoutHeight = disabledTimeoutHeight
			timeoutTimestamp = timestamp

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"next sequence send not found", func() {
			path := ibctesting.NewPath(s.chainA, s.chainB)
			sourceChannel = path.EndpointA.ChannelID

			s.coordinator.SetupConnections(path)
			// manually creating channel prevents next sequence from being set
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				s.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, path.EndpointA.ChannelConfig.Version),
			)
			s.chainA.CreateChannelCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"channel capability not found", func() {
			s.coordinator.Setup(path)
			sourceChannel = path.EndpointA.ChannelID

			channelCap = capabilitytypes.NewCapability(5)
		}, false},
	}

	for i, tc := range testCases {
		tc := tc
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

			sequence, err := s.chainA.App.GetIBCKeeper().ChannelKeeper.SendPacket(s.chainA.GetContext(), channelCap,
				sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, packetData)

			if tc.expPass {
				s.Require().NoError(err)
				// verify that the returned sequence matches expected value
				s.Require().True(ok)
				s.Require().Equal(expectedSequence, sequence, "send packet did not return the expected sequence of the outgoing packet")
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// TestRecvPacket test RecvPacket on chainB. Since packet commitment verification will always
// occur last (resource instensive), only tests expected to succeed and packet commitment
// verification tests need to simulate sending a packet from chainA to chainB.
func (s *KeeperTestSuite) TestRecvPacket() {
	var (
		path       *ibctesting.Path
		packet     exported.PacketI
		channelCap *capabilitytypes.Capability
		expError   *errorsmod.Error
	)

	testCases := []testCase{
		{"success: ORDERED channel", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, true},
		{"success UNORDERED channel", func() {
			// setup uses an UNORDERED channel
			s.coordinator.Setup(path)
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, true},
		{"success with out of order packet: UNORDERED channel", func() {
			// setup uses an UNORDERED channel
			s.coordinator.Setup(path)
			// send 2 packets
			_, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			// attempts to receive packet 2 without receiving packet 1
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, true},
		{"packet already relayed ORDERED channel (no-op)", func() {
			expError = types.ErrNoOpMsg

			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointB.RecvPacket(packet.(types.Packet))
			s.Require().NoError(err)
		}, false},
		{"packet already relayed UNORDERED channel (no-op)", func() {
			expError = types.ErrNoOpMsg

			// setup uses an UNORDERED channel
			s.coordinator.Setup(path)
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointB.RecvPacket(packet.(types.Packet))
			s.Require().NoError(err)
		}, false},
		{"out of order packet failure with ORDERED channel", func() {
			expError = types.ErrPacketSequenceOutOfOrder

			path.SetChannelOrdered()
			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			// send 2 packets
			_, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			// attempts to receive packet 2 without receiving packet 1
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"channel not found", func() {
			expError = types.ErrChannelNotFound

			// use wrong channel naming
			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"channel not open", func() {
			expError = types.ErrInvalidChannelState

			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			err := path.EndpointB.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"capability cannot authenticate ORDERED", func() {
			expError = types.ErrInvalidChannelCapability

			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = capabilitytypes.NewCapability(3)
		}, false},
		{"packet source port ≠ channel counterparty port", func() {
			expError = types.ErrInvalidPacket
			s.coordinator.Setup(path)

			// use wrong port for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"packet source channel ID ≠ channel counterparty channel ID", func() {
			expError = types.ErrInvalidPacket
			s.coordinator.Setup(path)

			// use wrong port for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, ibctesting.InvalidID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"connection not found", func() {
			expError = connectiontypes.ErrConnectionNotFound
			s.coordinator.Setup(path)

			// pass channel check
			s.chainB.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				s.chainB.GetContext(),
				path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{connIDB}, path.EndpointB.ChannelConfig.Version),
			)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			s.chainB.CreateChannelCapability(s.chainB.GetSimApp().ScopedIBCMockKeeper, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"connection not OPEN", func() {
			expError = connectiontypes.ErrInvalidConnectionState
			s.coordinator.SetupClients(path)

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
			s.chainB.CreateChannelCapability(s.chainB.GetSimApp().ScopedIBCMockKeeper, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"timeout height passed", func() {
			expError = types.ErrPacketTimeout
			s.coordinator.Setup(path)

			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.GetSelfHeight(s.chainB.GetContext()), disabledTimeoutTimestamp)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"timeout timestamp passed", func() {
			expError = types.ErrPacketTimeout
			s.coordinator.Setup(path)

			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, disabledTimeoutHeight, uint64(s.chainB.GetContext().BlockTime().UnixNano()))
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"next receive sequence is not found", func() {
			expError = types.ErrSequenceReceiveNotFound
			s.coordinator.SetupConnections(path)

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
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, packet.GetSequence(), types.CommitPacket(s.chainA.App.AppCodec(), packet))
			s.chainB.CreateChannelCapability(s.chainB.GetSimApp().ScopedIBCMockKeeper, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)
			err = path.EndpointB.UpdateClient()
			s.Require().NoError(err)
		}, false},
		{"receipt already stored", func() {
			expError = types.ErrNoOpMsg
			s.coordinator.Setup(path)

			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			s.chainB.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sequence)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"validation failed", func() {
			// skip error code check, downstream error code is used from light-client implementations

			// packet commitment not set resulting in invalid proof
			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
	}

	for i, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			s.SetupTest()  // reset
			expError = nil // must explicitly set for failed cases
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			// get proof of packet commitment from chainA
			packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			proof, proofHeight := path.EndpointA.QueryProof(packetKey)

			err := s.chainB.App.GetIBCKeeper().ChannelKeeper.RecvPacket(s.chainB.GetContext(), channelCap, packet, proof, proofHeight)

			if tc.expPass {
				s.Require().NoError(err)

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

				// only check if expError is set, since not all error codes can be known
				if expError != nil {
					s.Require().True(errors.Is(err, expError))
				}
			}
		})
	}
}

func (s *KeeperTestSuite) TestWriteAcknowledgement() {
	var (
		path       *ibctesting.Path
		ack        exported.Acknowledgement
		packet     exported.PacketI
		channelCap *capabilitytypes.Capability
	)

	testCases := []testCase{
		{
			"success",
			func() {
				s.coordinator.Setup(path)
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.MockAcknowledgement
				channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			true,
		},
		{"channel not found", func() {
			// use wrong channel naming
			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			ack = ibcmock.MockAcknowledgement
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"channel not open", func() {
			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			ack = ibcmock.MockAcknowledgement

			err := path.EndpointB.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
			channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{
			"capability authentication failed",
			func() {
				s.coordinator.Setup(path)
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.MockAcknowledgement
				channelCap = capabilitytypes.NewCapability(3)
			},
			false,
		},
		{
			"no-op, already acked",
			func() {
				s.coordinator.Setup(path)
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.MockAcknowledgement
				s.chainB.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(s.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(), ack.Acknowledgement())
				channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			false,
		},
		{
			"empty acknowledgement",
			func() {
				s.coordinator.Setup(path)
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.NewEmptyAcknowledgement()
				channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			false,
		},
		{
			"acknowledgement is nil",
			func() {
				s.coordinator.Setup(path)
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = nil
				channelCap = s.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			false,
		},
	}
	for i, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			err := s.chainB.App.GetIBCKeeper().ChannelKeeper.WriteAcknowledgement(s.chainB.GetContext(), channelCap, packet, ack)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// TestAcknowledgePacket tests the call AcknowledgePacket on chainA.
func (s *KeeperTestSuite) TestAcknowledgePacket() {
	var (
		path   *ibctesting.Path
		packet types.Packet
		ack    = ibcmock.MockAcknowledgement

		channelCap *capabilitytypes.Capability
		expError   *errorsmod.Error
	)

	testCases := []testCase{
		{"success on ordered channel", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// create packet receipt and acknowledgement
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"success on unordered channel", func() {
			// setup uses an UNORDERED channel
			s.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// create packet receipt and acknowledgement
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"packet already acknowledged ordered channel (no-op)", func() {
			expError = types.ErrNoOpMsg

			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// create packet receipt and acknowledgement
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			err = path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
			s.Require().NoError(err)
		}, false},
		{"packet already acknowledged unordered channel (no-op)", func() {
			expError = types.ErrNoOpMsg

			// setup uses an UNORDERED channel
			s.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// create packet receipt and acknowledgement
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			err = path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
			s.Require().NoError(err)
		}, false},
		{"channel not found", func() {
			expError = types.ErrChannelNotFound

			// use wrong channel naming
			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, ibctesting.InvalidID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, false},
		{"channel not open", func() {
			expError = types.ErrInvalidChannelState

			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			err := path.EndpointA.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"capability authentication failed ORDERED", func() {
			expError = types.ErrInvalidChannelCapability

			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// create packet receipt and acknowledgement
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			channelCap = capabilitytypes.NewCapability(3)
		}, false},
		{"packet destination port ≠ channel counterparty port", func() {
			expError = types.ErrInvalidPacket
			s.coordinator.Setup(path)

			// use wrong port for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"packet destination channel ID ≠ channel counterparty channel ID", func() {
			expError = types.ErrInvalidPacket
			s.coordinator.Setup(path)

			// use wrong channel for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"connection not found", func() {
			expError = connectiontypes.ErrConnectionNotFound
			s.coordinator.Setup(path)

			// pass channel check
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				s.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{"connection-1000"}, path.EndpointA.ChannelConfig.Version),
			)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			s.chainA.CreateChannelCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"connection not OPEN", func() {
			expError = connectiontypes.ErrInvalidConnectionState
			s.coordinator.SetupClients(path)
			// connection on chainA is in INIT
			err := path.EndpointA.ConnOpenInit()
			s.Require().NoError(err)

			// pass channel check
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				s.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, path.EndpointA.ChannelConfig.Version),
			)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			s.chainA.CreateChannelCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"packet hasn't been sent", func() {
			expError = types.ErrNoOpMsg

			// packet commitment never written
			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"packet ack verification failed", func() {
			// skip error code check since error occurs in light-clients

			// ack never written
			s.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"packet commitment bytes do not match", func() {
			expError = types.ErrInvalidPacket

			// setup uses an UNORDERED channel
			s.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// create packet receipt and acknowledgement
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			packet.Data = []byte("invalid packet commitment")
		}, false},
		{"next ack sequence not found", func() {
			expError = types.ErrSequenceAckNotFound
			s.coordinator.SetupConnections(path)

			path.EndpointA.ChannelID = ibctesting.FirstChannelID
			path.EndpointB.ChannelID = ibctesting.FirstChannelID

			// manually creating channel prevents next sequence acknowledgement from being set
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				s.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, path.EndpointA.ChannelConfig.Version),
			)

			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			// manually set packet commitment
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, packet.GetSequence(), types.CommitPacket(s.chainA.App.AppCodec(), packet))

			// manually set packet acknowledgement and capability
			s.chainB.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, packet.GetSequence(), types.CommitAcknowledgement(ack.Acknowledgement()))

			s.chainA.CreateChannelCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			s.coordinator.CommitBlock(path.EndpointA.Chain, path.EndpointB.Chain)

			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)
			err = path.EndpointB.UpdateClient()
			s.Require().NoError(err)
		}, false},
		{"next ack sequence mismatch ORDERED", func() {
			expError = types.ErrPacketSequenceOutOfOrder
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			// create packet commitment
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// create packet acknowledgement
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			// set next sequence ack wrong
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceAck(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 10)
			channelCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
	}

	for i, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			s.SetupTest()  // reset
			expError = nil // must explcitly set error for failed cases
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			proof, proofHeight := path.EndpointB.QueryProof(packetKey)

			err := s.chainA.App.GetIBCKeeper().ChannelKeeper.AcknowledgePacket(s.chainA.GetContext(), channelCap, packet, ack.Acknowledgement(), proof, proofHeight)
			pc := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(s.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

			channelA, _ := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetChannel(s.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel())
			sequenceAck, _ := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceAck(s.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel())

			if tc.expPass {
				s.NoError(err)
				s.Nil(pc)

				if channelA.Ordering == types.ORDERED {
					s.Require().Equal(packet.GetSequence()+1, sequenceAck, "sequence not incremented in ordered channel")
				} else {
					s.Require().Equal(uint64(1), sequenceAck, "sequence incremented for UNORDERED channel")
				}
			} else {
				s.Error(err)
				// only check if expError is set, since not all error codes can be known
				if expError != nil {
					s.Require().True(errors.Is(err, expError))
				}
			}
		})
	}
}
