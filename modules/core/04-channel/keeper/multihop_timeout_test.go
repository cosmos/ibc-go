package keeper_test

import (
	"errors"
	"fmt"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type timeoutTestCase = struct {
	msg            string
	orderedChannel bool
	malleate       func()
	expPass        bool
}

// TestTimeoutPacket test the TimeoutPacket call on chainA by ensuring the timeout has passed
// on chainB, but that no ack has been written yet. Test cases expected to reach proof
// verification must specify which proof to use using the ordered bool.
func (suite *MultihopTestSuite) TestTimeoutPacket() {
	var (
		packet       *types.Packet
		packetHeight exported.Height
		nextSeqRecv  uint64
		err          error
		expError     *sdkerrors.Error
	)

	testCases := []timeoutTestCase{
		{"success: ORDERED", true, func() {
			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels() // setup multihop channels
			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			timeoutTimestamp := uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano())
			packet, packetHeight, err = suite.A().SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
		}, true},
		{"success: UNORDERED", false, func() {
			suite.SetupChannels() // setup multihop channels
			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			packet, packetHeight, err = suite.A().SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
		}, true},
		{"packet already timed out: ORDERED", true, func() {
			expError = types.ErrNoOpMsg

			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels()

			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			timeoutTimestamp := uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano())

			packet, packetHeight, err = suite.A().
				SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA's client representing chainZ to prove missing ack
			err = suite.A().UpdateClient()
			suite.Require().NoError(err)

			err = suite.A().TimeoutPacket(*packet, packetHeight)
			suite.Require().NoError(err)
		}, false},
		{"packet already timed out: UNORDERED", false, func() {
			expError = types.ErrNoOpMsg

			suite.SetupChannels()

			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())

			packet, packetHeight, err = suite.A().
				SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA's client representing chainZ to prove missing ack
			err = suite.A().UpdateClient()
			suite.Require().NoError(err)

			err = suite.A().TimeoutPacket(*packet, packetHeight)
			suite.Require().NoError(err)
		}, false},
		{"channel not found", false, func() {
			expError = types.ErrChannelNotFound
			// use wrong channel naming
			suite.SetupChannels()
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, ibctesting.InvalidID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, false},
		{"channel not open", false, func() {
			expError = types.ErrInvalidChannelState
			suite.SetupChannels()

			timeoutHeight := suite.A().GetClientState().GetLatestHeight().Increment().(clienttypes.Height)

			packet, packetHeight, err = suite.A().
				SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// need to update chainA's client representing chainB to prove missing ack
			err = suite.A().UpdateClient()
			suite.Require().NoError(err)

			err = suite.A().SetChannelState(types.CLOSED)
			suite.Require().NoError(err)
		}, false},
		{"packet destination port ≠ channel counterparty port", false, func() {
			expError = types.ErrInvalidPacket
			suite.SetupChannels()
			// use wrong port for dest
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, ibctesting.InvalidID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, false},
		{"packet destination channel ID ≠ channel counterparty channel ID", false, func() {
			expError = types.ErrInvalidPacket
			suite.SetupChannels()
			// use wrong channel for dest
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, false},
		{"connection not found", false, func() {
			expError = connectiontypes.ErrConnectionNotFound
			// pass channel check
			suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				suite.A().Chain.GetContext(),
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID), []string{connIDA}, suite.A().ChannelConfig.Version),
			)
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, false},
		{"timeout", false, func() {
			expError = types.ErrPacketTimeout
			suite.SetupChannels()

			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			err = suite.A().UpdateClient()
			suite.Require().NoError(err)
		}, false},
		{"packet already received ", true, func() {
			expError = types.ErrPacketReceived
			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels() // setup multihop channels

			nextSeqRecv = 2
			timeoutTimestamp := uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano())

			packet, packetHeight, err = suite.A().
				SendPacket(defaultTimeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			err = suite.A().UpdateClient()
			suite.Require().NoError(err)
		}, false},
		{"packet hasn't been sent", true, func() {
			expError = types.ErrNoOpMsg
			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels() // setup multihop channels

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano()))

			err = suite.A().UpdateClient()
			suite.Require().NoError(err)
		}, false},
		{"next seq receive verification failed", false, func() { // set ordered to false resulting in wrong proof provided
			// skip error check, error occurs in light-clients

			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels() // setup multihop channels

			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())

			packet, packetHeight, err = suite.A().
				SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			err = suite.A().UpdateClient()
			suite.Require().NoError(err)
		}, false},
		{"packet ack verification failed", true, func() { // set ordered to true resulting in wrong proof provided
			// skip error check, error occurs in light-clients

			suite.SetupChannels() // setup multihop channels

			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())

			packet, packetHeight, err = suite.A().
				SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			err = suite.A().UpdateClient()
			suite.Require().NoError(err)
		}, false},
	}

	packet = &types.Packet{}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			var (
				proof       []byte
				proofHeight exported.Height
			)

			suite.SetupTest() // reset
			expError = nil    // must be expliticly changed by failed cases
			nextSeqRecv = 1   // must be explicitly changed

			tc.malleate()

			if suite.Z().ConnectionID != "" {
				var key []byte
				if tc.orderedChannel {
					// proof of inclusion of next sequence number
					key = host.NextSequenceRecvKey(packet.SourcePort, packet.SourceChannel)
				} else {
					// proof of absence of packet receipt
					key = host.PacketReceiptKey(packet.SourcePort, packet.SourceChannel, packet.Sequence)
				}
				doUpdateClient := true
				proof, proofHeight, err = suite.Z().QueryMultihopProof(key, packetHeight, doUpdateClient)
				suite.Require().NoError(err)
			}

			err := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.TimeoutPacket(suite.A().Chain.GetContext(), packet, proof, proofHeight, nextSeqRecv)

			if tc.expPass {
				suite.Require().NoError(err)
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

// TestTimeoutOnClose tests the call TimeoutOnClose on chainA by closing the corresponding
// channel on chainB after the packet commitment has been created.
func (suite *MultihopTestSuite) TestTimeoutOnClose() {
	var (
		packet                           *types.Packet
		packetHeight                     exported.Height
		chanCap                          *capabilitytypes.Capability
		nextSeqRecv                      uint64
		err                              error
		queryMultihopProofExpectedToFail bool
		doUpdateClient                   bool
	)

	testCases := []timeoutTestCase{
		{"success: ORDERED", true, func() {
			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels() // setup multihop channels

			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			timeoutTimestamp := uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano())

			packet, packetHeight, err = suite.A().SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			suite.Z().SetChannelClosed()

			// need to update chainA's client representing chainB to prove missing ack
			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
		{"success: UNORDERED", false, func() {
			suite.SetupChannels() // setup multihop channels

			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			packet, packetHeight, err = suite.A().SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			suite.Z().SetChannelClosed()

			// need to update chainA's client representing chainB to prove missing ack
			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
		{"channel not found", false, func() {
			// use wrong channel naming
			suite.SetupChannels() // setup multihop channels
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, ibctesting.InvalidID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, false},
		{"packet dest port ≠ channel counterparty port", false, func() {
			suite.SetupChannels()
			// use wrong port for dest
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, ibctesting.InvalidID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"packet dest channel ID ≠ channel counterparty channel ID", false, func() {
			suite.SetupChannels()
			// use wrong channel for dest
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"connection not found - scenario 1", false, func() {
			connectionIdx := 0
			suite.A().SetupAllButTheSpecifiedConnection(uint(connectionIdx))
			// pass channel check
			suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				suite.A().Chain.GetContext(),
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID), []string{connIDA}, suite.A().ChannelConfig.Version),
			)
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			// create chancap
			suite.A().Chain.CreateChannelCapability(suite.A().Chain.GetSimApp().ScopedIBCMockKeeper, suite.A().ChannelConfig.PortID, suite.A().ChannelID)
			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
			packetHeight = suite.Z().Chain.LastHeader.GetHeight()
			doUpdateClient = false
		}, false},
		{"connection not found - scenario 2", false, func() {
			connectionIdx := 1
			suite.A().SetupAllButTheSpecifiedConnection(uint(connectionIdx))
			// pass channel check
			suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				suite.A().Chain.GetContext(),
				suite.A().ChannelConfig.PortID, suite.A().ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID), []string{connIDA}, suite.A().ChannelConfig.Version),
			)
			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			// create chancap
			suite.A().Chain.CreateChannelCapability(suite.A().Chain.GetSimApp().ScopedIBCMockKeeper, suite.A().ChannelConfig.PortID, suite.A().ChannelID)
			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
			packetHeight = suite.Z().Chain.LastHeader.GetHeight()
			queryMultihopProofExpectedToFail = true
		}, false},
		{"packet hasn't been sent ORDERED", true, func() {
			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels() // setup multihop channels

			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			timeoutTimestamp := uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano())

			*packet = types.NewPacket(ibctesting.MockPacketData, 1, suite.A().ChannelConfig.PortID, suite.A().ChannelID, suite.Z().ChannelConfig.PortID, suite.Z().ChannelID, timeoutHeight, timeoutTimestamp)
			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"packet already received ORDERED", true, func() {
			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels() // setup multihop channels

			nextSeqRecv = 2
			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			timeoutTimestamp := uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano())

			packet, packetHeight, err = suite.A().
				SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			err = suite.Z().SetChannelState(types.CLOSED)
			suite.Require().NoError(err)

			err = suite.A().UpdateClient()
			suite.Require().NoError(err)

			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"channel verification failed ORDERED", true, func() {
			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels() // setup multihop channels

			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			timeoutTimestamp := uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano())

			packet, packetHeight, err = suite.A().
				SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"next seq receive verification failed ORDERED", false, func() {
			// set ordered to false providing the wrong proof for ORDERED case
			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels() // setup multihop channels

			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			timeoutTimestamp := uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano())

			packet, packetHeight, err = suite.A().
				SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			err = suite.Z().SetChannelState(types.CLOSED)
			suite.Require().NoError(err)

			err = suite.A().UpdateClient()
			suite.Require().NoError(err)

			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"packet ack verification failed", true, func() {
			// set ordered to true providing the wrong proof for UNORDERED case
			suite.SetupChannels() // setup multihop channels

			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())

			packet, packetHeight, err = suite.A().
				SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			err = suite.Z().SetChannelState(types.CLOSED)
			suite.Require().NoError(err)

			err = suite.A().UpdateClient()
			suite.Require().NoError(err)

			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, false},
		{"channel capability not found ORDERED", true, func() {
			suite.chanPath.SetChannelOrdered()
			suite.SetupChannels() // setup multihop channels

			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			timeoutTimestamp := uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano())

			packet, packetHeight, err = suite.A().
				SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			err = suite.Z().SetChannelState(types.CLOSED)
			suite.Require().NoError(err)

			// need to update chainA's client representing chainZ to prove missing ack
			err = suite.A().UpdateClient()
			suite.Require().NoError(err)

			chanCap = capabilitytypes.NewCapability(100)
		}, false},
	}

	packet = &types.Packet{}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			var key []byte

			suite.SetupTest() // reset
			nextSeqRecv = 1   // must be explicitly changed
			queryMultihopProofExpectedToFail = false
			doUpdateClient = true

			tc.malleate()

			channelKey := host.ChannelKey(suite.Z().ChannelConfig.PortID, suite.Z().ChannelID)

			if queryMultihopProofExpectedToFail {
				_, _, err := suite.Z().QueryMultihopProof(channelKey, packetHeight, doUpdateClient)
				suite.Require().Error(err)
			} else {
				proofClosed, _, err := suite.Z().QueryMultihopProof(channelKey, packetHeight, doUpdateClient)
				suite.Require().NoError(err)

				if tc.orderedChannel {
					key = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
				} else {
					// proof of absence of packet receipt
					key = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
				}

				proof, proofHeight, err := suite.Z().QueryMultihopProof(key, packetHeight, doUpdateClient)
				suite.Require().NoError(err)

				err = suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.TimeoutOnClose(suite.A().Chain.GetContext(), chanCap, packet, proof, proofClosed, proofHeight, nextSeqRecv)

				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			}
		})
	}
}
