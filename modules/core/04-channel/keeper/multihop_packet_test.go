package keeper_test

import (
	"errors"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
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
		packet         *types.Packet
		packetHeight   exported.Height
		channelCap     *capabilitytypes.Capability
		expError       *sdkerrors.Error
		err            error
	)

	testCases := []channelTestCase{
		{"success: ORDERED channel", true, func() {
			suite.SetupChannels()
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, true},
		{"success: UNORDERED channel", true, func() {
			suite.SetupChannels()
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, true},
		{"success with out of order packet: UNORDERED channel", false, func() {
			// setup uses an UNORDERED channel
			suite.SetupChannels()
			// send 2 packets
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, true},
		{"packet already relayed ORDERED channel (no-op)", false, func() {
			expError = types.ErrNoOpMsg

			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels()

			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)

			err = suite.Z().RecvPacket(packet, packetHeight)
			suite.Require().NoError(err)
		}, false},
		{"packet already relayed UNORDERED channel (no-op)", false, func() {
			expError = types.ErrNoOpMsg

			// setup uses an UNORDERED channel
			suite.SetupChannels()

			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)

			err = suite.Z().RecvPacket(packet, packetHeight)
			suite.Require().NoError(err)
		}, false},
		{"out of order packet failure with ORDERED channel", true, func() {
			expError = types.ErrPacketSequenceOutOfOrder

			suite.SetupChannels()

			// send 2 packets
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, false},
		{"channel not found", false, func() {
			expError = types.ErrChannelNotFound

			// use wrong channel naming
			suite.SetupChannels()
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, ibctesting.InvalidID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, false},
		{"channel not open", false, func() {
			expError = types.ErrInvalidChannelState

			suite.SetupChannels()
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)

			err = suite.Z().SetChannelState(types.CLOSED)
			suite.Require().NoError(err)
			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, false},
		{"capability cannot authenticate ORDERED", true, func() {
			expError = types.ErrInvalidChannelCapability

			suite.SetupChannels()

			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)

			channelCap = capabilitytypes.NewCapability(3)
		}, false},
		{"packet source port ≠ channel counterparty port", false, func() {
			expError = types.ErrInvalidPacket

			suite.SetupChannels()

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, false},
		{"packet source channel ID ≠ channel counterparty channel ID", false, func() {
			expError = types.ErrInvalidPacket

			suite.SetupChannels()

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, ibctesting.InvalidID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, false},
		{"connection not found", false, func() {
			expError = connectiontypes.ErrConnectionNotFound

			suite.SetupChannels()

			// pass channel check
			channel := types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(suite.A().ChannelConfig.PortID, suite.A().ChannelID), []string{connIDB}, suite.Z().ChannelConfig.Version)
			suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.Z().Chain.GetContext(), suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, channel)

			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)

			suite.Z().Chain.CreateChannelCapability(suite.Z().Chain.GetSimApp().ScopedIBCMockKeeper, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, false},
		{"connection not OPEN", false, func() {
			expError = connectiontypes.ErrInvalidConnectionState

			suite.coord.SetupClients(suite.chanPath)

			// connection on chainZ is in INIT
			err = suite.Z().ConnOpenInit()
			suite.Require().NoError(err)

			// pass channel check
			channel := types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(suite.A().ChannelConfig.PortID, suite.A().ChannelID), suite.Z().GetConnectionHops(), suite.Z().ChannelConfig.Version)
			suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.Z().Chain.GetContext(), suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, channel)
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			suite.Z().Chain.CreateChannelCapability(suite.Z().Chain.GetSimApp().ScopedIBCMockKeeper, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, false},
		{"timeout height passed", false, func() {
			expError = types.ErrPacketTimeout

			suite.SetupChannels()

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, clienttypes.GetSelfHeight(suite.Z().Chain.GetContext()), disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, false},
		{"timeout timestamp passed", false, func() {
			expError = types.ErrPacketTimeout

			suite.SetupChannels()

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, disabledTimeoutHeight, uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano()))
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			channelCap = suite.Z().Chain.GetChannelCapability(
				suite.Z().ChannelConfig.PortID, suite.Z().ChannelID,
			)
		}, false},
		{"next receive sequence is not found", false, func() {
			expError = types.ErrSequenceReceiveNotFound

			suite.SetupConnections()

			suite.A().ChannelID = ibctesting.FirstChannelID
			suite.Z().ChannelID = ibctesting.FirstChannelID

			// manually creating channel prevents next recv sequence from being set
			channel := types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(suite.A().ChannelConfig.PortID, suite.A().ChannelID), suite.Z().GetConnectionHops(), suite.Z().ChannelConfig.Version)
			suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.Z().Chain.GetContext(), suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, channel)

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			// manually set packet commitment
			suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.A().Chain.GetContext(), suite.A().ChannelConfig.PortID, suite.A().ChannelID, packet.GetSequence(), types.CommitPacket(suite.A().Chain.App.AppCodec(), packet))
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			suite.Z().Chain.CreateChannelCapability(suite.Z().Chain.GetSimApp().ScopedIBCMockKeeper, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)

			err = suite.A().UpdateClient()
			suite.Require().NoError(err)
			err = suite.Z().UpdateClient()
			suite.Require().NoError(err)
		}, false},
		{"receipt already stored", false, func() {
			expError = types.ErrNoOpMsg
			suite.SetupChannels()

			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			sequence := packet.GetSequence()
			suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(suite.Z().Chain.GetContext(), suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, sequence)
			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, false},
		{"validation failed", false, func() {
			// skip error code check, downstream error code is used from light-client implementations

			// packet commitment not set resulting in invalid proof
			suite.SetupChannels()

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			channelCap = suite.Z().Chain.GetChannelCapability(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)
		}, false},
	}

	packet = &types.Packet{}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			expError = nil    // must explicitly set for failed cases
			if tc.orderedChannel {
				suite.chanPath.SetChannelOrdered()
			}

			tc.malleate()

			proof, proofHeight, err := suite.A().QueryPacketProof(packet, packetHeight)
			suite.Require().NoError(err)

			err = suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.RecvPacket(
				suite.Z().Chain.GetContext(),
				channelCap,
				packet,
				proof,
				proofHeight,
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
					suite.Require().Equal(uint64(1), nextSeqRecv, "sequence incremented in UNORDERED channel")
					suite.Require().True(receiptStored, "packet receipt not stored in UNORDERED channel")
					suite.Require().Equal(string([]byte{byte(1)}), receipt, "packet receipt not stored in UNORDERED channel")
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
func (suite *MultihopTestSuite) TestAcknowledgePacket() {
	var (
		packet       *types.Packet
		packetHeight exported.Height
		ack          = ibcmock.MockAcknowledgement
		channelCap   *capabilitytypes.Capability
		expError     *sdkerrors.Error
		err          error
	)

	testCases := []channelTestCase{
		{"success: ORDERED channel", true, func() {
			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels() // setup multihop channels
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			suite.Require().NoError(suite.Z().RecvPacket(packet, packetHeight))
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
		{"success: UNORDERED channel", false, func() {
			suite.SetupChannels() // setup multihop channels
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			suite.Require().NoError(suite.Z().RecvPacket(packet, packetHeight))
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
		{"packet already acknowledged ordered channel (no-op)", true, func() {
			expError = types.ErrNoOpMsg

			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels()

			// create packet commitment
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// create packet receipt and acknowledgement
			err = suite.Z().RecvPacket(packet, packetHeight)
			suite.Require().NoError(err)

			channelCap = suite.A().Chain.GetChannelCapability(
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
			)

			err = suite.A().AcknowledgePacket(*packet, packetHeight, ack.Acknowledgement())
			suite.Require().NoError(err)
		}, false},
		{"packet already acknowledged unordered channel (no-op)", false, func() {
			expError = types.ErrNoOpMsg

			suite.SetupChannels()

			// create packet commitment
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// create packet receipt and acknowledgement
			err = suite.Z().RecvPacket(packet, packetHeight)
			suite.Require().NoError(err)

			channelCap = suite.A().Chain.GetChannelCapability(
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
			)

			err = suite.A().AcknowledgePacket(*packet, packetHeight, ack.Acknowledgement())
			suite.Require().NoError(err)
		}, false},
		{"channel not found", false, func() {
			expError = types.ErrChannelNotFound

			// use wrong channel naming
			suite.SetupChannels()
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, ibctesting.InvalidID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()
		}, false},
		{"channel not open", false, func() {
			expError = types.ErrInvalidChannelState

			suite.SetupChannels()

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			err := suite.A().SetChannelState(types.CLOSED)
			suite.Require().NoError(err)
			channelCap = suite.A().Chain.GetChannelCapability(
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
			)
		}, false},
		{"capability authentication failed ORDERED", true, func() {
			expError = types.ErrInvalidChannelCapability

			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels()

			// create packet commitment
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// create packet receipt and acknowledgement
			err = suite.Z().RecvPacket(packet, packetHeight)
			suite.Require().NoError(err)

			channelCap = suite.A().Chain.GetChannelCapability(
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
			)

			channelCap = capabilitytypes.NewCapability(3)
		}, false},
		{"packet destination port ≠ channel counterparty port", false, func() {
			expError = types.ErrInvalidPacket
			suite.SetupChannels()

			// use wrong port for dest
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, ibctesting.InvalidID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			channelCap = suite.A().Chain.GetChannelCapability(
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
			)
		}, false},
		{"packet destination channel ID ≠ channel counterparty channel ID", false, func() {
			expError = types.ErrInvalidPacket
			suite.SetupChannels()

			// use wrong channel for dest
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			channelCap = suite.A().Chain.GetChannelCapability(
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
			)
		}, false},
		{"connection not found", true, func() {
			expError = connectiontypes.ErrConnectionNotFound
			suite.SetupChannels()

			// pass channel check
			channel := types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID), []string{"connection-1000"}, suite.A().ChannelConfig.Version)
			suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.A().Chain.GetContext(), suite.A().ChannelConfig.PortID, suite.A().ChannelID, channel)

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			suite.A().Chain.CreateChannelCapability(suite.A().Chain.GetSimApp().ScopedIBCMockKeeper, suite.A().ChannelConfig.PortID, suite.A().ChannelID)
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"connection not OPEN", true, func() {
			expError = connectiontypes.ErrInvalidConnectionState
			suite.coord.SetupClients(suite.chanPath)
			// connection on chainA is in INIT
			err := suite.A().ConnOpenInit()
			suite.Require().NoError(err)

			// pass channel check
			channel := types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID), suite.A().GetConnectionHops(), suite.A().ChannelConfig.Version)
			suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.A().Chain.GetContext(), suite.A().ChannelConfig.PortID, suite.A().ChannelID, channel)

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			suite.A().Chain.CreateChannelCapability(suite.A().Chain.GetSimApp().ScopedIBCMockKeeper, suite.A().ChannelConfig.PortID, suite.A().ChannelID)
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"packet hasn't been sent", false, func() {
			expError = types.ErrNoOpMsg

			// packet commitment never written
			suite.SetupChannels()

			// use wrong channel for dest
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			packetHeight = suite.A().Chain.LastHeader.GetHeight()

			channelCap = suite.A().Chain.GetChannelCapability(
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
			)
		}, false},
		{"packet ack verification failed", false, func() {
			// skip error code check since error occurs in light-clients

			// ack never written
			suite.SetupChannels()

			// create packet commitment
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			channelCap = suite.A().Chain.GetChannelCapability(
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
			)
		}, false},
		{"packet commitment bytes do not match", false, func() {
			expError = types.ErrInvalidPacket

			// setup uses an UNORDERED channel
			suite.SetupChannels()

			// create packet commitment
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// create packet receipt and acknowledgement
			err = suite.Z().RecvPacket(packet, packetHeight)
			suite.Require().NoError(err)

			channelCap = suite.A().Chain.GetChannelCapability(
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
			)

			packet.Data = []byte("invalid packet commitment")
		}, false},
		{"next ack sequence not found", false, func() {
			expError = types.ErrSequenceAckNotFound
			suite.SetupConnections()

			suite.A().ChannelID = ibctesting.FirstChannelID
			suite.Z().ChannelID = ibctesting.FirstChannelID

			// manually creating channel prevents next sequence acknowledgement from being set
			channel := types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID), suite.A().GetConnectionHops(), suite.A().ChannelConfig.Version)
			suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.A().Chain.GetContext(), suite.A().ChannelConfig.PortID, suite.A().ChannelID, channel)

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			// manually set packet commitment
			suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.A().Chain.GetContext(), suite.A().ChannelConfig.PortID, suite.A().ChannelID, packet.GetSequence(), types.CommitPacket(suite.A().Chain.App.AppCodec(), packet))

			// manually set packet acknowledgement and capability
			suite.Z().Chain.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(suite.Z().Chain.GetContext(), suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, packet.GetSequence(), types.CommitAcknowledgement(ack.Acknowledgement()))

			suite.A().Chain.CreateChannelCapability(suite.A().Chain.GetSimApp().ScopedIBCMockKeeper, suite.A().ChannelConfig.PortID, suite.A().ChannelID)
			channelCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)

			suite.coord.CommitBlock(suite.A().Chain, suite.Z().Chain)
			packetHeight = suite.Z().Chain.LastHeader.GetHeight()

			err = suite.A().UpdateClient()
			suite.Require().NoError(err)
			err = suite.Z().UpdateClient()
			suite.Require().NoError(err)
		}, false},
		{"next ack sequence mismatch ORDERED", true, func() {
			expError = types.ErrPacketSequenceOutOfOrder

			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels()

			// create packet commitment
			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// create packet receipt and acknowledgement
			err = suite.Z().RecvPacket(packet, packetHeight)
			suite.Require().NoError(err)

			suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.SetNextSequenceAck(suite.A().Chain.GetContext(), suite.A().ChannelConfig.PortID, suite.A().ChannelID, 10)
			channelCap = suite.A().Chain.GetChannelCapability(
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
			)
		}, false},
	}

	packet = &types.Packet{}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			expError = nil    // must explcitly set error for failed cases

			tc.malleate()

			proof, proofHeight, err := suite.Z().QueryPacketAcknowledgementProof(packet, packetHeight)
			suite.Require().NoError(err)

			err = suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.AcknowledgePacket(
				suite.A().Chain.GetContext(),
				channelCap,
				packet,
				ack.Acknowledgement(),
				proof,
				proofHeight,
			)

			if tc.expPass {
				suite.Require().NoError(err)
				pc := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(
					suite.A().Chain.GetContext(),
					packet.GetSourcePort(),
					packet.GetSourceChannel(),
					packet.GetSequence(),
				)
				channelA, found := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.GetChannel(
					suite.A().Chain.GetContext(),
					packet.GetSourcePort(),
					packet.GetSourceChannel(),
				)
				suite.Require().True(found)
				seqAck, found := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.GetNextSequenceAck(
					suite.A().Chain.GetContext(),
					packet.GetSourcePort(),
					packet.GetSourceChannel(),
				)
				suite.Require().True(found)

				suite.Require().NoError(err)
				suite.Require().Nil(pc)
				suite.Require().NotNil(packet)

				if tc.orderedChannel {
					suite.Require().True(channelA.Ordering == types.ORDERED)
					suite.Require().
						Equal(packet.GetSequence()+1, seqAck, "sequence not incremented in ORDERED channel")
				} else {
					suite.Require().True(channelA.Ordering == types.UNORDERED)
					suite.Require().Equal(packet.GetSequence(), uint64(1), "sequence incremented in UNORDERED channel")
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
