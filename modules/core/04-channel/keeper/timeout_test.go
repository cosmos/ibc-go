package keeper_test

import (
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	abci "github.com/cometbft/cometbft/v2/abci/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// TestTimeoutPacket test the TimeoutPacket call on chainA by ensuring the timeout has passed
// on chainB, but that no ack has been written yet. Test cases expected to reach proof
// verification must specify which proof to use using the ordered bool.
func (suite *KeeperTestSuite) TestTimeoutPacket() {
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
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)
		}, nil},
		{"success: UNORDERED", func() {
			ordered = false
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)
		}, nil},
		{"packet already timed out: ORDERED", func() {
			expError = types.ErrNoOpMsg
			ordered = true
			path.SetChannelOrdered()
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			err = path.EndpointA.TimeoutPacket(packet)
			suite.Require().NoError(err)
		}, errorsmod.Wrap(types.ErrNoOpMsg, "")},
		{"packet already timed out: UNORDERED", func() {
			expError = types.ErrNoOpMsg
			ordered = false
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointA.TimeoutPacket(packet)
			suite.Require().NoError(err)
		}, errorsmod.Wrap(types.ErrNoOpMsg, "")},
		{"channel not found", func() {
			expError = types.ErrChannelNotFound
			// use wrong channel naming
			path.Setup()
			packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, ibctesting.InvalidID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, types.ErrChannelNotFound},
		{"packet destination port ≠ channel counterparty port", func() {
			expError = types.ErrInvalidPacket
			path.Setup()
			// use wrong port for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, errorsmod.Wrap(types.ErrInvalidPacket, "")},
		{"packet destination channel ID ≠ channel counterparty channel ID", func() {
			expError = types.ErrInvalidPacket
			path.Setup()
			// use wrong channel for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, errorsmod.Wrap(types.ErrInvalidPacket, "")},
		{"connection not found", func() {
			expError = connectiontypes.ErrConnectionNotFound
			// pass channel check
			suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				suite.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{connIDA}, path.EndpointA.ChannelConfig.Version),
			)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, "")},
		{"timeout", func() {
			expError = types.ErrTimeoutNotReached
			path.Setup()
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)
		}, errorsmod.Wrap(types.ErrTimeoutNotReached, "")},
		{"packet already received ", func() {
			expError = types.ErrPacketReceived
			ordered = true
			path.SetChannelOrdered()
			path.Setup()

			nextSeqRecv = 2
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)

			suite.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, timeoutTimestamp)
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)
		}, errorsmod.Wrap(types.ErrPacketReceived, "")},
		{"packet hasn't been sent", func() {
			expError = types.ErrNoOpMsg
			ordered = true
			path.SetChannelOrdered()

			path.Setup()
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, uint64(suite.chainB.GetContext().BlockTime().UnixNano()))
			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)
		}, errorsmod.Wrap(types.ErrNoOpMsg, "")},
		{"next seq receive verification failed", func() {
			// skip error check, error occurs in light-clients

			// set ordered to false resulting in wrong proof provided
			ordered = false

			path.SetChannelOrdered()
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "")},
		{"packet ack verification failed", func() {
			// skip error check, error occurs in light-clients

			// set ordered to true resulting in wrong proof provided
			ordered = true

			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "")},
	}

	for _, tc := range testCases {
		suite.Run(tc.msg, func() {
			var (
				proof       []byte
				proofHeight exported.Height
			)

			suite.SetupTest() // reset
			expError = nil    // must be expliticly changed by failed cases
			nextSeqRecv = 1   // must be explicitly changed
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

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

			channelVersion, err := suite.chainA.App.GetIBCKeeper().ChannelKeeper.TimeoutPacket(suite.chainA.GetContext(), packet, proof, proofHeight, nextSeqRecv)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().Equal(path.EndpointA.GetChannel().Version, channelVersion)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
				suite.Require().Equal("", channelVersion)
				// only check if expError is set, since not all error codes can be known
				if expError != nil {
					suite.Require().True(errors.Is(err, expError))
				}
			}
		})
	}
}

// TestTimeoutExecuted verifies that packet commitments are deleted.
// In addition, the test verifies that the channel state
// after a timeout is updated accordingly.
func (suite *KeeperTestSuite) TestTimeoutExecuted() {
	var (
		path   *ibctesting.Path
		packet types.Packet
	)

	testCases := []struct {
		msg       string
		malleate  func()
		expResult func(packetCommitment []byte, err error)
		expEvents func(path *ibctesting.Path) []abci.Event
	}{
		{
			"success ORDERED",
			func() {
				path.SetChannelOrdered()
				path.Setup()

				timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
				timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

				sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
			},
			func(packetCommitment []byte, err error) {
				suite.Require().NoError(err)
				suite.Require().Nil(packetCommitment)

				// Check channel has been closed
				channel := path.EndpointA.GetChannel()
				suite.Require().Equal(channel.State, types.CLOSED)
			},
			nil,
		},
	}

	for i, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			ctx := suite.chainA.GetContext()

			tc.malleate()

			err := suite.chainA.App.GetIBCKeeper().ChannelKeeper.TimeoutExecuted(ctx, path.EndpointA.GetChannel(), packet)
			pc := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(ctx, packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

			tc.expResult(pc, err)
			if tc.expEvents != nil {
				events := ctx.EventManager().ABCIEvents()

				expEvents := tc.expEvents(path)

				ibctesting.AssertEvents(&suite.Suite, expEvents, events)
			}
		})
	}
}

// TestTimeoutOnClose tests the call TimeoutOnClose on chainA by closing the corresponding
// channel on chainB after the packet commitment has been created.
func (suite *KeeperTestSuite) TestTimeoutOnClose() {
	var (
		path        *ibctesting.Path
		packet      types.Packet
		nextSeqRecv uint64
		ordered     bool
	)

	testCases := []testCase{
		{"success: ORDERED", func() {
			ordered = true
			path.SetChannelOrdered()
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
		}, nil},
		{"success: UNORDERED", func() {
			ordered = false
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
		}, nil},
		{"channel not found", func() {
			// use wrong channel naming
			path.Setup()
			packet = types.NewPacket(ibctesting.MockPacketData, 1, ibctesting.InvalidID, ibctesting.InvalidID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, types.ErrChannelNotFound},
		{"packet dest port ≠ channel counterparty port", func() {
			path.Setup()
			// use wrong port for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ibctesting.InvalidID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, errorsmod.Wrap(types.ErrInvalidPacket, "")},
		{"packet dest channel ID ≠ channel counterparty channel ID", func() {
			path.Setup()
			// use wrong channel for dest
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, ibctesting.InvalidID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, errorsmod.Wrap(types.ErrInvalidPacket, "")},
		{"connection not found", func() {
			// pass channel check
			suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetChannel(
				suite.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				types.NewChannel(types.OPEN, types.ORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{connIDA}, path.EndpointA.ChannelConfig.Version),
			)
			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, disabledTimeoutTimestamp)
		}, errorsmod.Wrap(connectiontypes.ErrConnectionNotFound, "")},
		{"packet hasn't been sent ORDERED", func() {
			path.SetChannelOrdered()
			path.Setup()

			packet = types.NewPacket(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.GetSelfHeight(suite.chainB.GetContext()), uint64(suite.chainB.GetContext().BlockTime().UnixNano()))
		}, errorsmod.Wrap(types.ErrNoOpMsg, "")},
		{"packet already received ORDERED", func() {
			path.SetChannelOrdered()
			nextSeqRecv = 2
			ordered = true
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
			// need to update chainA's client representing chainB to prove missing ack
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
		}, errorsmod.Wrap(types.ErrInvalidPacket, "")},
		{"channel verification failed ORDERED", func() {
			ordered = true
			path.SetChannelOrdered()
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, timeoutTimestamp)
		}, ibcerrors.ErrInvalidHeight},
		{"next seq receive verification failed ORDERED", func() {
			// set ordered to false providing the wrong proof for ORDERED case
			ordered = false
			path.SetChannelOrdered()
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.GetSelfHeight(suite.chainB.GetContext()), uint64(suite.chainB.GetContext().BlockTime().UnixNano()))
		}, errorsmod.Wrap(types.ErrInvalidPacket, "")},
		{"packet ack verification failed", func() {
			// set ordered to true providing the wrong proof for UNORDERED case
			ordered = true
			path.Setup()

			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

			sequence, err := path.EndpointA.SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			path.EndpointB.UpdateChannel(func(channel *types.Channel) { channel.State = types.CLOSED })
			err = path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			packet = types.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, timeoutHeight, disabledTimeoutTimestamp)
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "")},
	}

	for i, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			var proof []byte

			suite.SetupTest() // reset
			nextSeqRecv = 1   // must be explicitly changed
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()

			channelKey := host.ChannelKey(packet.GetDestPort(), packet.GetDestChannel())
			unorderedPacketKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			orderedPacketKey := host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())

			closedProof, proofHeight := suite.chainB.QueryProof(channelKey)

			if ordered {
				proof, _ = suite.chainB.QueryProof(orderedPacketKey)
			} else {
				proof, _ = suite.chainB.QueryProof(unorderedPacketKey)
			}

			channelVersion, err := suite.chainA.App.GetIBCKeeper().ChannelKeeper.TimeoutOnClose(
				suite.chainA.GetContext(),
				packet,
				proof,
				closedProof,
				proofHeight,
				nextSeqRecv,
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().Equal(path.EndpointA.GetChannel().Version, channelVersion)
			} else {
				suite.Require().Error(err)
				suite.Require().Equal("", channelVersion)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
