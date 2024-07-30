package keeper_test

import (
	"fmt"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	ibcmock "github.com/cosmos/ibc-go/v9/testing/mock"
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
func (suite *KeeperTestSuite) TestSendPacket() {
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
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"success: ORDERED channel", func() {
			path.SetChannelOrdered()
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"success with solomachine: UNORDERED channel", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			// swap client with solo machine
			solomachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachinesingle", "testing", 1)
			path.EndpointA.ClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 10)
			path.EndpointA.SetClientState(solomachine.ClientState())
			path.EndpointA.UpdateConnection(func(c *connectiontypes.ConnectionEnd) { c.ClientId = path.EndpointA.ClientID })

			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"success with solomachine: ORDERED channel", func() {
			path.SetChannelOrdered()
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			// swap client with solomachine
			solomachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachinesingle", "testing", 1)
			path.EndpointA.ClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 10)
			path.EndpointA.SetClientState(solomachine.ClientState())

			path.EndpointA.UpdateConnection(func(c *connectiontypes.ConnectionEnd) { c.ClientId = path.EndpointA.ClientID })

			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"packet basic validation failed, empty packet data", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			packetData = []byte{}
		}, false},
		{"channel not found", func() {
			// use wrong channel naming
			path.Setup()
			sourceChannel = ibctesting.InvalidID
			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"channel is in CLOSED state", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
		}, false},
		{"channel is in INIT state", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.INIT })
		}, false},
		{"channel is in TRYOPEN stage", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.TRYOPEN })
		}, false},
		{"connection not found", func() {
			// pass channel check
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.ConnectionHops[0] = "invalid-connection" })

			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"client state not found", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			// change connection client ID
			path.EndpointA.UpdateConnection(func(c *connectiontypes.ConnectionEnd) { c.ClientId = ibctesting.InvalidID })

			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"client state is frozen", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			connection := path.EndpointA.GetConnection()
			clientState := path.EndpointA.GetClientState()
			cs, ok := clientState.(*ibctm.ClientState)
			suite.Require().True(ok)

			// freeze client
			cs.FrozenHeight = clienttypes.NewHeight(0, 1)
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), connection.ClientId, cs)

			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"client state zero height", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			connection := path.EndpointA.GetConnection()
			clientState := path.EndpointA.GetClientState()
			cs, ok := clientState.(*ibctm.ClientState)
			suite.Require().True(ok)

			// force a consensus state into the store at height zero to allow client status check to pass.
			consensusState := path.EndpointA.GetConsensusState(cs.LatestHeight)
			path.EndpointA.SetConsensusState(consensusState, clienttypes.ZeroHeight())

			cs.LatestHeight = clienttypes.ZeroHeight()
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), connection.ClientId, cs)

			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"timeout height passed", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			var ok bool
			timeoutHeight, ok = path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			suite.Require().True(ok)
			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"timeout timestamp passed", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			connection := path.EndpointA.GetConnection()
			timestamp, err := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientTimestampAtHeight(suite.chainA.GetContext(), connection.ClientId, path.EndpointA.GetClientLatestHeight())
			suite.Require().NoError(err)

			timeoutHeight = disabledTimeoutHeight
			timeoutTimestamp = timestamp
			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"timeout timestamp passed with solomachine", func() {
			path.Setup()
			// swap client with solomachine
			solomachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachinesingle", "testing", 1)
			path.EndpointA.ClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 10)
			path.EndpointA.SetClientState(solomachine.ClientState())

			path.EndpointA.UpdateConnection(func(c *connectiontypes.ConnectionEnd) { c.ClientId = path.EndpointA.ClientID })

			connection := path.EndpointA.GetConnection()
			timestamp, err := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientTimestampAtHeight(suite.chainA.GetContext(), connection.ClientId, path.EndpointA.GetClientLatestHeight())
			suite.Require().NoError(err)

			sourceChannel = path.EndpointA.ChannelID
			timeoutHeight = disabledTimeoutHeight
			timeoutTimestamp = timestamp

			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"next sequence send not found", func() {
			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			sourceChannel = path.EndpointA.ChannelID

			path.SetupConnections()
			// manually creating channel prevents next sequence from being set
			suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				suite.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, path.EndpointA.ChannelConfig.Version),
			)
			suite.chainA.CreateChannelCapability(suite.chainA.GetSimApp().ScopedIBCMockKeeper, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"channel capability not found", func() {
			path.Setup()
			sourceChannel = path.EndpointA.ChannelID

			channelCap = capabilitytypes.NewCapability(5)
		}, false},
		{
			"channel is in FLUSH_COMPLETE state",
			func() {
				path.Setup()
				sourceChannel = path.EndpointA.ChannelID

				path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHCOMPLETE })
			},
			false,
		},
		{
			"channel is in FLUSHING state",
			func() {
				path.Setup()

				path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion
				path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibcmock.UpgradeVersion

				err := path.EndpointB.ChanUpgradeInit()
				suite.Require().NoError(err)

				err = path.EndpointA.ChanUpgradeTry()
				suite.Require().NoError(err)
			},
			false,
		},
	}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			// set default send packet arguments
			// sourceChannel is set after path is setup
			sourcePort = path.EndpointA.ChannelConfig.PortID
			timeoutHeight = defaultTimeoutHeight
			timeoutTimestamp = disabledTimeoutTimestamp
			packetData = ibctesting.MockPacketData

			// malleate may modify send packet arguments above
			tc.malleate()

			// only check if nextSequenceSend exists in no error case since it is a tested error case above.
			expectedSequence, ok := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceSend(suite.chainA.GetContext(), sourcePort, sourceChannel)

			sequence, err := suite.chainA.App.GetIBCKeeper().ChannelKeeper.SendPacket(suite.chainA.GetContext(), channelCap,
				sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, packetData)

			if tc.expPass {
				suite.Require().NoError(err)
				// verify that the returned sequence matches expected value
				suite.Require().True(ok)
				suite.Require().Equal(expectedSequence, sequence, "send packet did not return the expected sequence of the outgoing packet")
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestRecvPacket test RecvPacket on chainB. Since packet commitment verification will always
// occur last (resource instensive), only tests expected to succeed and packet commitment
// verification tests need to simulate sending a packet from chainA to chainB.
func (suite *KeeperTestSuite) TestRecvPacket() {
	var (
		path       *ibctesting.Path
		packet     types.Packet
		channelCap *capabilitytypes.Capability
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
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			nil,
		},
		{
			"success UNORDERED channel",
			func() {
				// setup uses an UNORDERED channel
				path.Setup()
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			nil,
		},
		{
			"success UNORDERED channel in FLUSHING",
			func() {
				// setup uses an UNORDERED channel
				path.Setup()
				path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHING })

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			nil,
		},
		{
			"success UNORDERED channel in FLUSHCOMPLETE",
			func() {
				// setup uses an UNORDERED channel
				path.Setup()
				path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHCOMPLETE })

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
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
				suite.Require().NoError(err)
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				// attempts to receive packet 2 without receiving packet 1
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			nil,
		},
		{
			"success with counterpartyNextSequenceSend higher than packet sequence",
			func() {
				path.Setup()
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

				path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHING })

				// set upgrade next sequence send to sequence + 1
				counterpartyUpgrade := types.Upgrade{NextSequenceSend: sequence + 1}
				path.EndpointB.SetChannelCounterpartyUpgrade(counterpartyUpgrade)
			},
			nil,
		},
		{
			"success with counterparty upgrade not found",
			func() {
				path.Setup()
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

				path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHING })
			},
			nil,
		},
		{
			"failure while upgrading channel, packet sequence ≥ counterparty next send sequence",
			func() {
				path.Setup()
				// send 2 packets so that when NextSequenceSend is set to sequence - 1, it is not 0.
				_, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

				path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHING })

				// set upgrade next sequence send to sequence - 1
				counterpartyUpgrade := types.Upgrade{NextSequenceSend: sequence - 1}
				path.EndpointB.SetChannelCounterpartyUpgrade(counterpartyUpgrade)
			},
			types.ErrInvalidPacket,
		},
		{
			"packet already relayed ORDERED channel (no-op)",
			func() {
				path.SetChannelOrdered()
				path.Setup()

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)
			},
			types.ErrNoOpMsg,
		},
		{
			"packet already relayed UNORDERED channel (no-op)",
			func() {
				// setup uses an UNORDERED channel
				path.Setup()
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)
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
				suite.Require().NoError(err)
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				// attempts to receive packet 2 without receiving packet 1
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			types.ErrPacketSequenceOutOfOrder,
		},
		{
			"channel not found",
			func() {
				// use wrong channel naming
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			types.ErrChannelNotFound,
		},
		{
			"channel not open",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			types.ErrInvalidChannelState,
		},
		{
			"capability cannot authenticate ORDERED",
			func() {
				path.SetChannelOrdered()
				path.Setup()

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = capabilitytypes.NewCapability(3)
			},
			types.ErrInvalidChannelCapability,
		},
		{
			"packet source port ≠ channel counterparty port",
			func() {
				path.Setup()

				// use wrong port for dest
				packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			types.ErrInvalidPacket,
		},
		{
			"packet source channel ID ≠ channel counterparty channel ID",
			func() {
				path.Setup()

				// use wrong port for dest
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, ibctesting.InvalidID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			types.ErrInvalidPacket,
		},
		{
			"connection not found",
			func() {
				path.Setup()

				// pass channel check
				suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetChannel(
					suite.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
					types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{connIDB}, path.EndpointB.ChannelConfig.Version),
				)
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				suite.chainB.CreateChannelCapability(suite.chainB.GetSimApp().ScopedIBCMockKeeper, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			connectiontypes.ErrConnectionNotFound,
		},
		{
			"connection not OPEN",
			func() {
				path.SetupClients()

				// connection on chainB is in INIT
				err := path.EndpointB.ConnOpenInit()
				suite.Require().NoError(err)

				// pass channel check
				suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetChannel(
					suite.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
					types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, path.EndpointB.ChannelConfig.Version),
				)
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				suite.chainB.CreateChannelCapability(suite.chainB.GetSimApp().ScopedIBCMockKeeper, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			connectiontypes.ErrInvalidConnectionState,
		},
		{
			"timeout height passed",
			func() {
				path.Setup()

				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.GetSelfHeight(suite.chainB.GetContext()), disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			types.ErrTimeoutElapsed,
		},
		{
			"timeout timestamp passed",
			func() {
				path.Setup()

				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, disabledTimeoutHeight, uint64(suite.chainB.GetContext().BlockTime().UnixNano()))
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
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
				suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetChannel(
					suite.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
					types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, path.EndpointB.ChannelConfig.Version),
				)

				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				// manually set packet commitment
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, packet.GetSequence(), types.CommitPacket(packet))
				suite.chainB.CreateChannelCapability(suite.chainB.GetSimApp().ScopedIBCMockKeeper, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
				err = path.EndpointB.UpdateClient()
				suite.Require().NoError(err)
			},
			types.ErrSequenceReceiveNotFound,
		},
		{
			"packet already received",
			func() {
				path.Setup()

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

				// set recv seq start to indicate packet was processed in previous upgrade
				suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetRecvStartSequence(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sequence+1)
			},
			types.ErrPacketReceived,
		},
		{
			"receipt already stored",
			func() {
				path.Setup()

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sequence)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			types.ErrNoOpMsg,
		},
		{
			"validation failed",
			func() {
				// packet commitment not set resulting in invalid proof
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			// get proof of packet commitment from chainA
			packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			proof, proofHeight := path.EndpointA.QueryProof(packetKey)

			err := suite.chainB.App.GetIBCKeeper().ChannelKeeper.RecvPacket(suite.chainB.GetContext(), channelCap, packet, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				channelB, _ := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetChannel(suite.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel())
				nextSeqRecv, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceRecv(suite.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel())
				suite.Require().True(found)
				receipt, receiptStored := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketReceipt(suite.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

				if channelB.Ordering == types.ORDERED {
					suite.Require().Equal(packet.GetSequence()+1, nextSeqRecv, "sequence not incremented in ordered channel")
					suite.Require().False(receiptStored, "packet receipt stored on ORDERED channel")
				} else {
					suite.Require().Equal(uint64(1), nextSeqRecv, "sequence incremented for UNORDERED channel")
					suite.Require().True(receiptStored, "packet receipt not stored after RecvPacket in UNORDERED channel")
					suite.Require().Equal(string([]byte{byte(1)}), receipt, "packet receipt is not empty string")
				}
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestWriteAcknowledgement() {
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
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.MockAcknowledgement
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			true,
		},
		{
			"success: channel flushing",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.MockAcknowledgement

				path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHING })
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			true,
		},
		{
			"success: channel flush complete",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.MockAcknowledgement

				path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHCOMPLETE })
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			true,
		},
		{"channel not found", func() {
			// use wrong channel naming
			path.Setup()
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			ack = ibcmock.MockAcknowledgement
			channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{"channel not open", func() {
			path.Setup()
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			ack = ibcmock.MockAcknowledgement

			path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
			channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		}, false},
		{
			"capability authentication failed",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.MockAcknowledgement
				channelCap = capabilitytypes.NewCapability(3)
			},
			false,
		},
		{
			"no-op, already acked",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.MockAcknowledgement
				suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(suite.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(), ack.Acknowledgement())
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			false,
		},
		{
			"empty acknowledgement",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = ibcmock.NewEmptyAcknowledgement()
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			false,
		},
		{
			"acknowledgement is nil",
			func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				ack = nil
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			},
			false,
		},
		{
			"packet already received",
			func() {
				path.Setup()

				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainB.GetChannelCapability(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				ack = ibcmock.MockAcknowledgement

				// set recv seq start to indicate packet was processed in previous upgrade
				suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetRecvStartSequence(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sequence+1)
			},
			false,
		},
	}
	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			err := suite.chainB.App.GetIBCKeeper().ChannelKeeper.WriteAcknowledgement(suite.chainB.GetContext(), channelCap, packet, ack)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestAcknowledgePacket tests the call AcknowledgePacket on chainA.
func (suite *KeeperTestSuite) TestAcknowledgePacket() {
	var (
		path   *ibctesting.Path
		packet types.Packet
		ack    = ibcmock.MockAcknowledgement

		channelCap *capabilitytypes.Capability
	)

	assertErr := func(errType *errors.Error) func(commitment []byte, err error) {
		return func(commitment []byte, err error) {
			suite.Require().Error(err)
			suite.Require().ErrorIs(err, errType)
			suite.Require().NotNil(commitment)
		}
	}

	assertNoOp := func(commitment []byte, err error) {
		suite.Require().Error(err)
		suite.Require().ErrorIs(err, types.ErrNoOpMsg)
		suite.Require().Nil(commitment)
	}

	assertSuccess := func(seq func() uint64, msg string) func(commitment []byte, err error) {
		return func(commitment []byte, err error) {
			suite.Require().NoError(err)
			suite.Require().Nil(commitment)

			nextSequenceAck, found := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceAck(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel())

			suite.Require().True(found)
			suite.Require().Equal(seq(), nextSequenceAck, msg)
		}
	}

	testCases := []struct {
		name      string
		malleate  func()
		expResult func(commitment []byte, err error)
		expEvents func(path *ibctesting.Path) []abci.Event
	}{
		{
			name: "success on ordered channel",
			malleate: func() {
				path.SetChannelOrdered()
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
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
				suite.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			},
			expResult: assertSuccess(func() uint64 { return uint64(1) }, "sequence incremented for UNORDERED channel"),
		},
		{
			name: "success on channel in flushing state",
			malleate: func() {
				// setup uses an UNORDERED channel
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHING })
			},
			expResult: func(commitment []byte, err error) {
				suite.Require().NoError(err)
				suite.Require().Nil(commitment)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(types.FLUSHING, channel.State)

				nextSequenceAck, found := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceAck(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel())
				suite.Require().True(found)
				suite.Require().Equal(uint64(1), nextSequenceAck, "sequence incremented for UNORDERED channel")
			},
		},
		{
			name: "success on channel in flushing state with valid timeout",
			malleate: func() {
				// setup uses an UNORDERED channel
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHING })

				counterpartyUpgrade := types.Upgrade{
					Timeout: types.NewTimeout(suite.chainB.GetTimeoutHeight(), 0),
				}

				path.EndpointA.SetChannelCounterpartyUpgrade(counterpartyUpgrade)
			},
			expResult: func(commitment []byte, err error) {
				suite.Require().NoError(err)
				suite.Require().Nil(commitment)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(types.FLUSHCOMPLETE, channel.State)

				nextSequenceAck, found := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceAck(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel())
				suite.Require().True(found)
				suite.Require().Equal(uint64(1), nextSequenceAck, "sequence incremented for UNORDERED channel")
			},
			expEvents: func(path *ibctesting.Path) []abci.Event {
				return sdk.Events{
					sdk.NewEvent(
						types.EventTypeChannelFlushComplete,
						sdk.NewAttribute(types.AttributeKeyPortID, path.EndpointA.ChannelConfig.PortID),
						sdk.NewAttribute(types.AttributeKeyChannelID, path.EndpointA.ChannelID),
						sdk.NewAttribute(types.AttributeCounterpartyPortID, path.EndpointB.ChannelConfig.PortID),
						sdk.NewAttribute(types.AttributeCounterpartyChannelID, path.EndpointB.ChannelID),
						sdk.NewAttribute(types.AttributeKeyChannelState, path.EndpointA.GetChannel().State.String()),
					),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
					),
				}.ToABCIEvents()
			},
		},
		{
			name: "success on channel in flushing state with timeout passed",
			malleate: func() {
				// setup uses an UNORDERED channel
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHING })

				upgrade := types.Upgrade{
					Fields:  types.NewUpgradeFields(types.UNORDERED, []string{ibctesting.FirstConnectionID}, ibcmock.UpgradeVersion),
					Timeout: types.NewTimeout(clienttypes.ZeroHeight(), 1),
				}

				counterpartyUpgrade := types.Upgrade{
					Fields:  types.NewUpgradeFields(types.UNORDERED, []string{ibctesting.FirstConnectionID}, ibcmock.UpgradeVersion),
					Timeout: types.NewTimeout(clienttypes.ZeroHeight(), 1),
				}

				path.EndpointA.SetChannelUpgrade(upgrade)
				path.EndpointA.SetChannelCounterpartyUpgrade(counterpartyUpgrade)
			},
			expResult: func(commitment []byte, err error) {
				suite.Require().NoError(err)
				suite.Require().Nil(commitment)

				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(types.OPEN, channel.State)

				nextSequenceAck, found := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceAck(suite.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel())
				suite.Require().True(found)
				suite.Require().Equal(uint64(1), nextSequenceAck, "sequence incremented for UNORDERED channel")

				errorReceipt, found := path.EndpointA.Chain.App.GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found)
				suite.Require().NotEmpty(errorReceipt)
			},
		},
		{
			name: "packet already acknowledged ordered channel (no-op)",
			malleate: func() {
				path.SetChannelOrdered()
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				err = path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
				suite.Require().NoError(err)
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
				suite.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				err = path.EndpointA.AcknowledgePacket(packet, ack.Acknowledgement())
				suite.Require().NoError(err)
			},
			expResult: assertNoOp,
		},
		{
			name: "channel not found",
			malleate: func() {
				// use wrong channel naming
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

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
				suite.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			},
			expResult: assertErr(types.ErrInvalidChannelState),
		},
		{
			name: "channel in flush complete state",
			malleate: func() {
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				path.EndpointA.UpdateChannel(func(channel *types.Channel) { channel.State = types.FLUSHCOMPLETE })
			},
			expResult: func(commitment []byte, err error) {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, types.ErrInvalidChannelState)
				suite.Require().Nil(commitment)
			},
		},
		{
			name: "capability authentication failed ORDERED",
			malleate: func() {
				path.SetChannelOrdered()
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				channelCap = capabilitytypes.NewCapability(3)
			},
			expResult: assertErr(types.ErrInvalidChannelCapability),
		},
		{
			name: "packet destination port ≠ channel counterparty port",
			malleate: func() {
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// use wrong port for dest
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			},
			expResult: assertErr(types.ErrInvalidPacket),
		},
		{
			name: "packet destination channel ID ≠ channel counterparty channel ID",
			malleate: func() {
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// use wrong channel for dest
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			},
			expResult: assertErr(types.ErrInvalidPacket),
		},
		{
			name: "connection not found",
			malleate: func() {
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				// pass channel check
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
					suite.chainA.GetContext(),
					path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
					types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{"connection-1000"}, path.EndpointA.ChannelConfig.Version),
				)

				suite.chainA.CreateChannelCapability(suite.chainA.GetSimApp().ScopedIBCMockKeeper, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			},
			expResult: assertErr(connectiontypes.ErrConnectionNotFound),
		},
		{
			name: "connection not OPEN",
			malleate: func() {
				path.Setup()

				// create packet commitment
				sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

				// connection on chainA is in INIT
				err = path.EndpointA.ConnOpenInit()
				suite.Require().NoError(err)

				// pass channel check
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
					suite.chainA.GetContext(),
					path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
					types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, path.EndpointA.ChannelConfig.Version),
				)
				suite.chainA.CreateChannelCapability(suite.chainA.GetSimApp().ScopedIBCMockKeeper, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			},
			expResult: assertErr(connectiontypes.ErrInvalidConnectionState),
		},
		{
			name: "packet hasn't been sent",
			malleate: func() {
				// packet commitment never written
				path.Setup()
				packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
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
				suite.Require().NoError(err)
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
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
				suite.Require().NoError(err)

				// create packet receipt and acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

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
				suite.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				// manually delete the next sequence ack in the ibc store
				storeKey := suite.chainA.GetSimApp().GetKey(exported.ModuleName)
				ibcStore := suite.chainA.GetContext().KVStore(storeKey)

				ibcStore.Delete(host.NextSequenceAckKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))

				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
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
				suite.Require().NoError(err)

				// create packet acknowledgement
				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
				err = path.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				// set next sequence ack wrong
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceAck(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 10)
				channelCap = suite.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			},
			expResult: assertErr(types.ErrPacketSequenceOutOfOrder),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			ctx := suite.chainA.GetContext()

			tc.malleate()

			packetKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			proof, proofHeight := path.EndpointB.QueryProof(packetKey)

			err := suite.chainA.App.GetIBCKeeper().ChannelKeeper.AcknowledgePacket(ctx, channelCap, packet, ack.Acknowledgement(), proof, proofHeight)

			commitment := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(ctx, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, packet.GetSequence())
			tc.expResult(commitment, err)
			if tc.expEvents != nil {
				events := ctx.EventManager().ABCIEvents()

				expEvents := tc.expEvents(path)

				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			}
		})
	}
}
