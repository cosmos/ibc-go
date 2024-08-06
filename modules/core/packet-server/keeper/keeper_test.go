package keeper_test

import (
	"fmt"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	tmtypes "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

var (
	defaultTimeoutHeight     = clienttypes.NewHeight(1, 100)
	disabledTimeoutTimestamp = uint64(0)
)

// KeeperTestSuite is a testing suite to test keeper functions.
type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

// TestKeeperTestSuite runs all the tests within this package.
func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

// SetupTest creates a coordinator with 2 test chains.
func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
}

func (suite *KeeperTestSuite) TestSendPacket() {
	var (
		path   *ibctesting.Path
		packet channeltypes.Packet
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{"success", func() {}, nil},
		{"counterparty not found", func() {
			packet.SourceChannel = ibctesting.FirstChannelID
		}, channeltypes.ErrChannelNotFound},
		{"packet failed basic validation", func() {
			// invalid data
			packet.Data = nil
		}, channeltypes.ErrInvalidPacket},
		{"client status invalid", func() {
			// make underlying client Frozen to get invalid client status
			clientState, ok := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
			suite.Require().True(ok, "could not retrieve client state")
			tmClientState, ok := clientState.(*tmtypes.ClientState)
			suite.Require().True(ok, "client is not tendermint client")
			tmClientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID, tmClientState)
		}, clienttypes.ErrClientNotActive},
		{"timeout elapsed", func() {
			packet.TimeoutTimestamp = 1
		}, channeltypes.ErrTimeoutElapsed},
	}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.name, i, len(testCases)), func() {
			suite.SetupTest() // reset

			// create clients and set counterparties on both chains
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			// create standard packet that can be malleated
			packet = channeltypes.NewPacketWithVersion(mock.MockPacketData, 1, mock.PortID,
				path.EndpointA.ClientID, mock.PortID, path.EndpointB.ClientID, clienttypes.NewHeight(1, 100), 0, mock.Version)

			// malleate the test case
			tc.malleate()

			// send packet
			seq, err := suite.chainA.App.GetPacketServer().SendPacket(suite.chainA.GetContext(), nil, packet.SourceChannel, packet.SourcePort,
				packet.DestinationPort, packet.TimeoutHeight, packet.TimeoutTimestamp, packet.AppVersion, packet.Data)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(uint64(1), seq)
				expCommitment := channeltypes.CommitPacket(packet)
				suite.Require().Equal(expCommitment, suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(suite.chainA.GetContext(), packet.SourcePort, packet.SourceChannel, seq))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Equal(uint64(0), seq)
				suite.Require().Nil(suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(suite.chainA.GetContext(), packet.SourcePort, packet.SourceChannel, seq))

			}
		})
	}
}

func (suite *KeeperTestSuite) TestRecvPacket() {
	var (
		path   *ibctesting.Path
		packet channeltypes.Packet
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
			"failure: protocol version is not V2",
			func() {
				packet.ProtocolVersion = channeltypes.IBC_VERSION_1
			},
			channeltypes.ErrInvalidPacket,
		},
		{
			"failure: counterparty not found",
			func() {
				packet.DestinationChannel = ibctesting.FirstChannelID
			},
			channeltypes.ErrChannelNotFound,
		},
		{
			"failure: client is not active",
			func() {
				clientState, ok := suite.chainB.GetClientState(packet.DestinationChannel).(*tmtypes.ClientState)
				suite.Require().True(ok)
				clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
				suite.chainB.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainB.GetContext(), packet.DestinationChannel, clientState)
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"failure: counterparty client identifier different than source channel",
			func() {
				packet.SourceChannel = ibctesting.FirstChannelID
			},
			channeltypes.ErrInvalidChannelIdentifier,
		},
		{
			"failure: packet has timed out",
			func() {
				packet.TimeoutHeight = clienttypes.ZeroHeight()
				packet.TimeoutTimestamp = uint64(suite.chainB.GetContext().BlockTime().UnixNano())
			},
			channeltypes.ErrTimeoutElapsed,
		},
		{
			"failure: packet already received",
			func() {
				suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, packet.Sequence)
			},
			channeltypes.ErrNoOpMsg,
		},
		{
			"failure: verify membership failed",
			func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), packet.SourcePort, packet.SourceChannel, packet.Sequence, []byte(""))
				suite.coordinator.CommitBlock(path.EndpointA.Chain)
				suite.Require().NoError(path.EndpointB.UpdateClient())
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			packet = channeltypes.NewPacketWithVersion(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ClientID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, defaultTimeoutHeight, disabledTimeoutTimestamp, "")

			// For now, set packet commitment on A for each case and update clients. Use SendPacket after 7048.
			suite.chainA.App.GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.chainA.GetContext(), packet.SourcePort, packet.SourceChannel, packet.Sequence, channeltypes.CommitPacket(packet))

			suite.coordinator.CommitBlock(path.EndpointA.Chain)
			suite.Require().NoError(path.EndpointB.UpdateClient())

			tc.malleate()

			// get proof of packet commitment from chainA
			packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			proof, proofHeight := path.EndpointA.QueryProof(packetKey)

			err := suite.chainB.App.GetPacketServer().RecvPacket(suite.chainB.GetContext(), nil, packet, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				_, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketReceipt(suite.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, packet.Sequence)
				suite.Require().True(found)
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
		packet       channeltypes.Packet
		freezeClient bool
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success with timeout height",
			func() {
				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacket(suite.chainA.GetContext(), nil, packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.AppVersion, packet.Data)
				suite.Require().NoError(err, "send packet failed")
			},
			nil,
		},
		{
			"success with timeout timestamp",
			func() {
				// disable timeout height and set timeout timestamp to a past timestamp
				packet.TimeoutHeight = clienttypes.Height{}
				packet.TimeoutTimestamp = uint64(suite.chainB.GetContext().BlockTime().UnixNano())

				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacket(suite.chainA.GetContext(), nil, packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.AppVersion, packet.Data)
				suite.Require().NoError(err, "send packet failed")
			},
			nil,
		},
		{
			"failure: invalid protocol version",
			func() {
				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacket(suite.chainA.GetContext(), nil, packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.AppVersion, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				packet.ProtocolVersion = channeltypes.IBC_VERSION_1
				packet.AppVersion = ""
			},
			channeltypes.ErrInvalidPacket,
		},
		{
			"failure: counterparty not found",
			func() {
				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacket(suite.chainA.GetContext(), nil, packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.AppVersion, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				packet.SourceChannel = ibctesting.FirstChannelID
			},
			channeltypes.ErrChannelNotFound,
		},
		{
			"failure: counterparty client identifier different than source channel",
			func() {
				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacket(suite.chainA.GetContext(), nil, packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.AppVersion, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				packet.DestinationChannel = ibctesting.FirstChannelID
			},
			channeltypes.ErrInvalidChannelIdentifier,
		},
		{
			"failure: packet has not timed out yet",
			func() {
				packet.TimeoutHeight = defaultTimeoutHeight

				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacket(suite.chainA.GetContext(), nil, packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.AppVersion, packet.Data)
				suite.Require().NoError(err, "send packet failed")
			},
			channeltypes.ErrTimeoutNotReached,
		},
		{
			"failure: packet already timed out",
			func() {}, // equivalent to not sending packet at all
			channeltypes.ErrNoOpMsg,
		},
		{
			"failure: packet does not match commitment",
			func() {
				// send a different packet
				_, err := suite.chainA.App.GetPacketServer().SendPacket(suite.chainA.GetContext(), nil, packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.AppVersion, []byte("different data"))
				suite.Require().NoError(err, "send packet failed")
			},
			channeltypes.ErrInvalidPacket,
		},
		{
			"failure: client status invalid",
			func() {
				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacket(suite.chainA.GetContext(), nil, packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.AppVersion, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				freezeClient = true
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"failure: verify non-membership failed",
			func() {
				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacket(suite.chainA.GetContext(), nil, packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.AppVersion, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				// set packet receipt to mock a valid past receive
				suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, packet.Sequence)
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			// intialize freezeClient to false
			freezeClient = false

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			// create default packet with a timed out height
			// test cases may mutate timeout values
			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			packet = channeltypes.NewPacketWithVersion(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ClientID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, timeoutHeight, disabledTimeoutTimestamp, "")

			tc.malleate()

			// need to update chainA's client representing chainB to prove missing ack
			// commit the changes and update the clients
			suite.coordinator.CommitBlock(path.EndpointA.Chain)
			suite.Require().NoError(path.EndpointB.UpdateClient())
			suite.Require().NoError(path.EndpointA.UpdateClient())

			// get proof of packet receipt absence from chainB
			receiptKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			proof, proofHeight := path.EndpointB.QueryProof(receiptKey)

			if freezeClient {
				// make underlying client Frozen to get invalid client status
				clientState, ok := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID)
				suite.Require().True(ok, "could not retrieve client state")
				tmClientState, ok := clientState.(*tmtypes.ClientState)
				suite.Require().True(ok, "client is not tendermint client")
				tmClientState.FrozenHeight = clienttypes.NewHeight(0, 1)
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), path.EndpointA.ClientID, tmClientState)
			}

			err := suite.chainA.App.GetPacketServer().TimeoutPacket(suite.chainA.GetContext(), packet, proof, proofHeight, 0)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				commitment := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(suite.chainA.GetContext(), packet.DestinationPort, packet.DestinationChannel, packet.Sequence)
				suite.Require().Nil(commitment, "packet commitment not deleted")
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
