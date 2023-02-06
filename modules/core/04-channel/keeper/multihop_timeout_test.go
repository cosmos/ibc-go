package keeper_test

import (
	"errors"
	"fmt"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
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
		packet      *types.Packet
		nextSeqRecv uint64
		err         error
		expError    *sdkerrors.Error
	)

	testCases := []timeoutTestCase{
		{"success: ORDERED", true, func() {
			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			timeoutTimestamp := uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano())
			packet, err = suite.A().SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
		}, true},
		{"success: UNORDERED", false, func() {
			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			packet, err = suite.A().SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
		}, true},
	}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			var (
				proof       []byte
				proofHeight exported.Height
			)

			suite.SetupTest() // reset
			if tc.orderedChannel {
				suite.chanPath.SetChannelOrdered()
			}
			expError = nil        // must be expliticly changed by failed cases
			nextSeqRecv = 1       // must be explicitly changed
			suite.SetupChannels() // setup multihop channels

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
				proof = suite.Z().QueryMultihopProof(key)
				proofHeight = suite.A().ProofHeight()
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
		packet      *types.Packet
		chanCap     *capabilitytypes.Capability
		nextSeqRecv uint64
		err         error
	)

	testCases := []timeoutTestCase{
		{"success: ORDERED", true, func() {
			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			timeoutTimestamp := uint64(suite.Z().Chain.GetContext().BlockTime().UnixNano())

			packet, err = suite.A().SendPacket(timeoutHeight, timeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			suite.Z().SetChannelClosed()
			// need to update chainA's client representing chainB to prove missing ack
			suite.A().UpdateAllClients()
			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
		{"success: UNORDERED", false, func() {
			timeoutHeight := clienttypes.GetSelfHeight(suite.Z().Chain.GetContext())
			packet, err = suite.A().SendPacket(timeoutHeight, disabledTimeoutTimestamp, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			suite.Z().SetChannelClosed()
			// need to update chainA's client representing chainB to prove missing ack
			suite.A().UpdateAllClients()
			chanCap = suite.A().Chain.GetChannelCapability(suite.A().ChannelConfig.PortID, suite.A().ChannelID)
		}, true},
	}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			var key []byte

			suite.SetupTest() // reset
			if tc.orderedChannel {
				suite.chanPath.SetChannelOrdered()
			}
			nextSeqRecv = 1       // must be explicitly changed
			suite.SetupChannels() // setup multihop channels

			tc.malleate()

			proofClosed := suite.Z().QueryChannelProof()
			proofHeight := suite.A().GetClientState().GetLatestHeight()

			if tc.orderedChannel {
				key = host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
			} else {
				// proof of absence of packet receipt
				key = host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			}

			proof := suite.Z().QueryMultihopProof(key)
			err := suite.A().Chain.App.GetIBCKeeper().ChannelKeeper.TimeoutOnClose(suite.A().Chain.GetContext(), chanCap, packet, proof, proofClosed, proofHeight, nextSeqRecv)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
