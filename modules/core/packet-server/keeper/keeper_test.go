package keeper_test

import (
	"fmt"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
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
		packet channeltypes.PacketV2
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
			"counterparty not found",
			func() {
				packet.SourceChannel = ibctesting.FirstChannelID
			},
			clienttypes.ErrCounterpartyNotFound,
		},
		// TODO validation is not implemented yet
		// {
		// 	"packet failed basic validation",
		// 	func() {
		// 		// invalid data
		// 		packet.Data = nil
		// 	},
		// 	channeltypes.ErrInvalidPacket,
		// },
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
			"timeout elapsed", func() {
				packet.TimeoutTimestamp = 1
			},
			channeltypes.ErrTimeoutElapsed,
		},
	}

	for i, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.name, i, len(testCases)), func() {
			suite.SetupTest() // reset

			// create clients and set counterparties on both chains
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			// create standard packet that can be malleated
			packet = channeltypes.NewPacketV2(mock.MockChannelPacketData, 1, mock.PortID, path.EndpointA.ClientID, mock.PortID, path.EndpointB.ClientID, clienttypes.NewHeight(1, 100), 0)

			// malleate the test case
			tc.malleate()

			// send packet
			seq, err := suite.chainA.App.GetPacketServer().SendPacketV2(suite.chainA.GetContext(), packet.SourceChannel, packet.SourcePort,
				packet.DestinationPort, packet.TimeoutHeight, packet.TimeoutTimestamp, packet.Data)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(uint64(1), seq)
				expCommitment := channeltypes.CommitPacketV2(packet)
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
		packet channeltypes.PacketV2
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
			"failure: counterparty not found",
			func() {
				packet.DestinationChannel = ibctesting.FirstChannelID
			},
			clienttypes.ErrCounterpartyNotFound,
		},
		{
			"failure: client is not active",
			func() {
				path.EndpointB.FreezeClient()
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

			// send packet
			sequence, err := path.EndpointA.SendPacketV2(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockChannelPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacketV2(ibctesting.MockChannelPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ClientID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			tc.malleate()

			// get proof of packet commitment from chainA
			packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			proof, proofHeight := path.EndpointA.QueryProof(packetKey)

			err = suite.chainB.App.GetPacketServer().RecvPacketV2(suite.chainB.GetContext(), packet, proof, proofHeight)

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

func (suite *KeeperTestSuite) TestWriteAcknowledgement() {
	var (
		packet channeltypes.Packet
		ack    exported.Acknowledgement
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
			"failure: protocol version is not IBC_VERSION_2",
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
			clienttypes.ErrCounterpartyNotFound,
		},
		{
			"failure: counterparty client identifier different than source channel",
			func() {
				packet.SourceChannel = ibctesting.FirstChannelID
			},
			channeltypes.ErrInvalidChannelIdentifier,
		},
		{
			"failure: ack already exists",
			func() {
				ackBz := channeltypes.CommitAcknowledgement(ack.Acknowledgement())
				suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(suite.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, packet.Sequence, ackBz)
			},
			channeltypes.ErrAcknowledgementExists,
		},
		{
			"failure: ack is nil",
			func() {
				ack = nil
			},
			channeltypes.ErrInvalidAcknowledgement,
		},
		{
			"failure: empty ack",
			func() {
				ack = mock.NewEmptyAcknowledgement()
			},
			channeltypes.ErrInvalidAcknowledgement,
		},
		{
			"failure: receipt not found for packet",
			func() {
				packet.Sequence = 2
			},
			channeltypes.ErrInvalidPacket,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			packet = channeltypes.NewPacketWithVersion(ibctesting.MockPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ClientID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, defaultTimeoutHeight, disabledTimeoutTimestamp, mock.Version)
			ack = mock.MockAcknowledgement

			suite.chainB.App.GetIBCKeeper().ChannelKeeper.SetPacketReceipt(suite.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, packet.Sequence)

			tc.malleate()

			err := suite.chainB.App.GetPacketServer().WriteAcknowledgement(suite.chainB.GetContext(), nil, packet, ack)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				ackCommitment, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, packet.Sequence)
				suite.Require().True(found)
				suite.Require().Equal(channeltypes.CommitAcknowledgement(ack.Acknowledgement()), ackCommitment)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestAcknowledgePacket() {
	var (
		packet       channeltypes.PacketV2
		ack          = mock.MockMultiAcknowledgement
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
			"failure: counterparty not found",
			func() {
				packet.SourceChannel = ibctesting.FirstChannelID
			},
			clienttypes.ErrCounterpartyNotFound,
		},
		{
			"failure: counterparty client identifier different than source channel",
			func() {
				packet.DestinationChannel = ibctesting.FirstChannelID
			},
			channeltypes.ErrInvalidChannelIdentifier,
		},
		{
			"failure: packet commitment doesn't exist.",
			func() {
				suite.chainA.App.GetIBCKeeper().ChannelKeeper.DeletePacketCommitment(suite.chainA.GetContext(), packet.SourcePort, packet.SourceChannel, packet.Sequence)
			},
			channeltypes.ErrNoOpMsg,
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
				packet.Data = ibctesting.MockFailChannelPacketData
			},
			channeltypes.ErrInvalidPacket,
		},
		{
			"failure: verify membership fails",
			func() {
				ack = channeltypes.MultiAcknowledgement{
					AcknowledgementResults: []channeltypes.AcknowledgementResult{
						{
							AppName: "mockV2A",
							RecvPacketResult: channeltypes.RecvPacketResult{
								Status:          channeltypes.PacketStatus_Success,
								Acknowledgement: []byte("swapped acknowledgement"),
							},
						},
					},
				}
			},
			commitmenttypes.ErrInvalidProof,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			freezeClient = false

			// send packet
			sequence, err := path.EndpointA.SendPacketV2(defaultTimeoutHeight, disabledTimeoutTimestamp, ibctesting.MockChannelPacketData)
			suite.Require().NoError(err)

			packet = channeltypes.NewPacketV2(ibctesting.MockChannelPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ClientID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, defaultTimeoutHeight, disabledTimeoutTimestamp)

			err = path.EndpointB.RecvPacketV2(packet)
			suite.Require().NoError(err)

			tc.malleate()

			packetKey := host.PacketAcknowledgementKey(packet.DestinationPort, packet.DestinationChannel, packet.GetSequence())
			proof, proofHeight := path.EndpointB.QueryProof(packetKey)

			if freezeClient {
				path.EndpointA.FreezeClient()
			}

			err = suite.chainA.App.GetPacketServer().AcknowledgePacketV2(suite.chainA.GetContext(), packet, ack, proof, proofHeight)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				commitment := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(suite.chainA.GetContext(), packet.SourcePort, packet.SourceChannel, packet.Sequence)
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
		packet       channeltypes.PacketV2
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
				_, err := suite.chainA.App.GetPacketServer().SendPacketV2(suite.chainA.GetContext(), packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.Data)
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
				_, err := suite.chainA.App.GetPacketServer().SendPacketV2(suite.chainA.GetContext(), packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.Data)
				suite.Require().NoError(err, "send packet failed")
			},
			nil,
		},
		{
			"failure: counterparty not found",
			func() {
				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacketV2(suite.chainA.GetContext(), packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				packet.SourceChannel = ibctesting.FirstChannelID
			},
			clienttypes.ErrCounterpartyNotFound,
		},
		{
			"failure: counterparty client identifier different than source channel",
			func() {
				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacketV2(suite.chainA.GetContext(), packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.Data)
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
				_, err := suite.chainA.App.GetPacketServer().SendPacketV2(suite.chainA.GetContext(), packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.Data)
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
				_, err := suite.chainA.App.GetPacketServer().SendPacketV2(suite.chainA.GetContext(), packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, ibctesting.MockFailChannelPacketData)
				suite.Require().NoError(err, "send packet failed")
			},
			channeltypes.ErrInvalidPacket,
		},
		{
			"failure: client status invalid",
			func() {
				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacketV2(suite.chainA.GetContext(), packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.Data)
				suite.Require().NoError(err, "send packet failed")

				freezeClient = true
			},
			clienttypes.ErrClientNotActive,
		},
		{
			"failure: verify non-membership failed",
			func() {
				// send packet
				_, err := suite.chainA.App.GetPacketServer().SendPacketV2(suite.chainA.GetContext(), packet.SourceChannel, packet.SourcePort, packet.DestinationPort,
					packet.TimeoutHeight, packet.TimeoutTimestamp, packet.Data)
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
			// initialize freezeClient to false1
			freezeClient = false

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			// create default packet with a timed out height
			// test cases may mutate timeout values
			timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
			packet = channeltypes.NewPacketV2(ibctesting.MockChannelPacketData, 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ClientID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ClientID, timeoutHeight, disabledTimeoutTimestamp)

			tc.malleate()

			// need to update chainA's client representing chainB to prove missing ack
			// commit the changes and update the clients
			suite.coordinator.CommitBlock(path.EndpointA.Chain)
			suite.Require().NoError(path.EndpointB.UpdateClient())
			suite.Require().NoError(path.EndpointA.UpdateClient())

			// get proof of packet receipt absence from chainB
			receiptKey := host.PacketReceiptKey(packet.DestinationPort, packet.DestinationChannel, packet.GetSequence())
			proof, proofHeight := path.EndpointB.QueryProof(receiptKey)

			if freezeClient {
				path.EndpointA.FreezeClient()
			}

			err := suite.chainA.App.GetPacketServer().TimeoutPacketV2(suite.chainA.GetContext(), nil, packet, proof, proofHeight, 0)

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
