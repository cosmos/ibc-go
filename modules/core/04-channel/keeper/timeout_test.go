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
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

// TestTimeoutPacket test the TimeoutPacket call on chainA by ensuring the timeout has passed
// on chainB, but that no ack has been written yet. Test cases expected to reach proof
// verification must specify which proof to use using the ordered bool.
func (s *KeeperTestSuite) TestTimeoutPacket() {
	var (
		path        *ibctesting.Path
		packet      types.Packet
		nextSeqRecv uint64
		ordered     bool
		expError    *errorsmod.Error
	)

	testCases := []testCase{
		{"success: ORDERED", func() {
			ordered = true
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)
		}, true},
		{"success: UNORDERED", func() {
			ordered = false
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)
		}, true},
		{"packet already timed out: ORDERED", func() {
			expError = types.ErrNoOpMsg
			ordered = true
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			err = path.EndpointA.TimeoutPacket(packet)
			s.Require().NoError(err)
		}, false},
		{"packet already timed out: UNORDERED", func() {
			expError = types.ErrNoOpMsg
			ordered = false
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointA.TimeoutPacket(packet)
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

			timeoutHeight := path.EndpointA.GetClientState().GetLatestHeight().Increment().(clienttypes.Height)

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			err = path.EndpointA.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
		}, false},
		{"packet destination port ≠ channel counterparty port", func() {
			expError = types.ErrInvalidPacket
			s.coordinator.Setup(path)
			// use wrong port for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, false},
		{"packet destination channel ID ≠ channel counterparty channel ID", func() {
			expError = types.ErrInvalidPacket
			s.coordinator.Setup(path)
			// use wrong channel for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, false},
		{"connection not found", func() {
			expError = connectiontypes.ErrConnectionNotFound
			// pass channel check
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				s.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{connIDA}, path.EndpointA.ChannelConfig.Version),
			)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, false},
		{"timeout", func() {
			expError = types.ErrPacketTimeout
			s.coordinator.Setup(path)
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)
		}, false},
		{"packet already received ", func() {
			expError = types.ErrPacketReceived
			ordered = true
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			nextSeqRecv = 2
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, timeoutTimestamp)
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)
		}, false},
		{"packet hasn't been sent", func() {
			expError = types.ErrNoOpMsg
			ordered = true
			path.SetChannelOrdered()

			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, uint64(s.chainB.GetContext().BlockTime().UnixNano()))
			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)
		}, false},
		{"next seq receive verification failed", func() {
			// skip error check, error occurs in light-clients

			// set ordered to false resulting in wrong proof provided
			ordered = false

			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)
		}, false},
		{"packet ack verification failed", func() {
			// skip error check, error occurs in light-clients

			// set ordered to true resulting in wrong proof provided
			ordered = true

			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)
		}, false},
	}

	for i, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			var (
				proof       []byte
				proofHeight exported.Height
			)

			s.SetupTest()   // reset
			expError = nil  // must be expliticly changed by failed cases
			nextSeqRecv = 1 // must be explicitly changed
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			orderedPacketKey := host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
			unorderedPacketKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

			if path.EndpointB.ConnectionID != "" {
				if ordered {
					proof, proofHeight = path.EndpointB.QueryProof(orderedPacketKey)
				} else {
					proof, proofHeight = path.EndpointB.QueryProof(unorderedPacketKey)
				}
			}

			err := s.chainA.App.GetIBCKeeper().ChannelKeeper.TimeoutPacket(s.chainA.GetContext(), packet, proof, proofHeight, nextSeqRecv)

			if tc.expPass {
				s.Require().NoError(err)
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

// TestTimeoutExectued verifies that packet commitments are deleted on chainA after the
// channel capabilities are verified.
func (s *KeeperTestSuite) TestTimeoutExecuted() {
	var (
		path    *ibctesting.Path
		packet  types.Packet
		chanCap *capabilitytypes.Capability
	)

	testCases := []testCase{
		{"success ORDERED", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			chanCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"channel not found", func() {
			// use wrong channel naming
			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, ibctesting.InvalidID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, false},
		{"incorrect capability ORDERED", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			chanCap = capabilitytypes.NewCapability(100)
		}, false},
	}

	for i, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			err := s.chainA.App.GetIBCKeeper().ChannelKeeper.TimeoutExecuted(s.chainA.GetContext(), chanCap, packet)
			pc := s.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(s.chainA.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

			if tc.expPass {
				s.NoError(err)
				s.Nil(pc)
			} else {
				s.Error(err)
			}
		})
	}
}

// TestTimeoutOnClose tests the call TimeoutOnClose on chainA by closing the corresponding
// channel on chainB after the packet commitment has been created.
func (s *KeeperTestSuite) TestTimeoutOnClose() {
	var (
		path        *ibctesting.Path
		packet      types.Packet
		chanCap     *capabilitytypes.Capability
		nextSeqRecv uint64
		ordered     bool
	)

	testCases := []testCase{
		{"success: ORDERED", func() {
			ordered = true
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			err = path.EndpointB.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			chanCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"success: UNORDERED", func() {
			ordered = false
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			err = path.EndpointB.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
			chanCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, true},
		{"channel not found", func() {
			// use wrong channel naming
			s.coordinator.Setup(path)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, ibctesting.InvalidID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, false},
		{"packet dest port ≠ channel counterparty port", func() {
			s.coordinator.Setup(path)
			// use wrong port for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			chanCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"packet dest channel ID ≠ channel counterparty channel ID", func() {
			s.coordinator.Setup(path)
			// use wrong channel for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			chanCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"connection not found", func() {
			// pass channel check
			s.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				s.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{connIDA}, path.EndpointA.ChannelConfig.Version),
			)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			// create chancap
			s.chainA.CreateChannelCapability(s.chainA.GetSimApp().ScopedIBCMockKeeper, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			chanCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"packet hasn't been sent ORDERED", func() {
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.GetSelfHeight(s.chainB.GetContext()), uint64(s.chainB.GetContext().BlockTime().UnixNano()))
			chanCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"packet already received ORDERED", func() {
			path.SetChannelOrdered()
			nextSeqRecv = 2
			ordered = true
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			err = path.EndpointB.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			chanCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"channel verification failed ORDERED", func() {
			ordered = true
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			chanCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"next seq receive verification failed ORDERED", func() {
			// set ordered to false providing the wrong proof for ORDERED case
			ordered = false
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			err = path.EndpointB.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.GetSelfHeight(s.chainB.GetContext()), uint64(s.chainB.GetContext().BlockTime().UnixNano()))
			chanCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"packet ack verification failed", func() {
			// set ordered to true providing the wrong proof for UNORDERED case
			ordered = true
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			err = path.EndpointB.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
			chanCap = s.chainA.GetChannelCapability(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		}, false},
		{"channel capability not found ORDERED", func() {
			ordered = true
			path.SetChannelOrdered()
			s.coordinator.Setup(path)

			timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			s.Require().NoError(err)
			err = path.EndpointB.SetChannelState(types.CLOSED)
			s.Require().NoError(err)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.GetSelfHeight(s.chainB.GetContext()), uint64(s.chainB.GetContext().BlockTime().UnixNano()))
			chanCap = capabilitytypes.NewCapability(100)
		}, false},
	}

	for i, tc := range testCases {
		tc := tc
		s.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			var proof []byte

			s.SetupTest()   // reset
			nextSeqRecv = 1 // must be explicitly changed
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()

			channelKey := host.ChannelKey(packet.GetDestPort(), packet.GetDestChannel())
			unorderedPacketKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			orderedPacketKey := host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			proofClosed, proofHeight := s.chainB.QueryProof(channelKey)

			if ordered {
				proof, _ = s.chainB.QueryProof(orderedPacketKey)
			} else {
				proof, _ = s.chainB.QueryProof(unorderedPacketKey)
			}

			err := s.chainA.App.GetIBCKeeper().ChannelKeeper.TimeoutOnClose(s.chainA.GetContext(), chanCap, packet, proof, proofClosed, proofHeight, nextSeqRecv)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
