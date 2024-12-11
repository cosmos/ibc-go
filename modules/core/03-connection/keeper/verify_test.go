package keeper_test

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	ibcmock "github.com/cosmos/ibc-go/v9/testing/mock"
)

var defaultTimeoutHeight = clienttypes.NewHeight(1, 100000)

// TestVerifyConnectionState verifies the connection state of the connection
// on chainB. The connections on chainA and chainB are fully opened.
func (suite *KeeperTestSuite) TestVerifyConnectionState() {
	var (
		path       *ibctesting.Path
		heightDiff uint64
	)
	cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{"verification success", func() {}, nil},
		{"client state not found - changed client ID", func() {
			path.EndpointA.UpdateConnection(func(c *types.ConnectionEnd) { c.ClientId = ibctesting.InvalidID })
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (IDisInvalid) status is Unauthorized")},
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed connection state verification for client (07-tendermint-0): client state height < proof height ({1 9} < {1 14}), please ensure the client has been updated")},
		{"verification failed - connection state is different than proof", func() {
			path.EndpointA.UpdateConnection(func(c *types.ConnectionEnd) { c.State = types.TRYOPEN })
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed connection state verification for client (07-tendermint-0): client state height < proof height ({1 9} < {1 14}), please ensure the client has been updated")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupConnections()

			connectionKey := host.ConnectionKey(path.EndpointB.ConnectionID)
			proof, proofHeight := suite.chainB.QueryProof(connectionKey)

			tc.malleate()

			connection := path.EndpointA.GetConnection()

			expectedConnection := path.EndpointB.GetConnection()

			err := suite.chainA.App.GetIBCKeeper().ConnectionKeeper.VerifyConnectionState(
				suite.chainA.GetContext(), connection,
				malleateHeight(proofHeight, heightDiff), proof, path.EndpointB.ConnectionID, expectedConnection,
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestVerifyChannelState verifies the channel state of the channel on
// chainB. The channels on chainA and chainB are fully opened.
func (suite *KeeperTestSuite) TestVerifyChannelState() {
	var (
		path       *ibctesting.Path
		heightDiff uint64
	)
	cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{"verification success", func() {}, nil},
		{"client state not found- changed client ID", func() {
			path.EndpointA.UpdateConnection(func(c *types.ConnectionEnd) { c.ClientId = ibctesting.InvalidID })
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (IDisInvalid) status is Unauthorized")},
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed channel state verification for client (07-tendermint-0): client state height < proof height ({1 15} < {1 20}), please ensure the client has been updated")},
		{"verification failed - changed channel state", func() {
			path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.TRYOPEN })
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed channel state verification for client (07-tendermint-0): client state height < proof height ({1 15} < {1 20}), please ensure the client has been updated")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			proof, proofHeight := suite.chainB.QueryProof(channelKey)

			tc.malleate()
			connection := path.EndpointA.GetConnection()

			channel := path.EndpointB.GetChannel()

			err := suite.chainA.App.GetIBCKeeper().ConnectionKeeper.VerifyChannelState(
				suite.chainA.GetContext(), connection, malleateHeight(proofHeight, heightDiff), proof,
				path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, channel,
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestVerifyPacketCommitment has chainB verify the packet commitment
// on channelA. The channels on chainA and chainB are fully opened and a
// packet is sent from chainA to chainB, but has not been received.
func (suite *KeeperTestSuite) TestVerifyPacketCommitment() {
	var (
		path            *ibctesting.Path
		packet          channeltypes.Packet
		heightDiff      uint64
		delayTimePeriod uint64
		timePerBlock    uint64
	)
	cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{"verification success", func() {}, nil},
		{"verification success: delay period passed", func() {
			delayTimePeriod = uint64(1 * time.Second.Nanoseconds())
		}, nil},
		{"delay time period has not passed", func() {
			delayTimePeriod = uint64(1 * time.Hour.Nanoseconds())
		}, errorsmod.Wrap(ibctm.ErrDelayPeriodNotPassed, "failed packet commitment verification for client (07-tendermint-0): cannot verify packet until time: 1577926940000000000, current time: 1577923345000000000")},
		{"delay block period has not passed", func() {
			// make timePerBlock 1 nanosecond so that block delay is not passed.
			// must also set a non-zero time delay to ensure block delay is enforced.
			delayTimePeriod = uint64(1 * time.Second.Nanoseconds())
			timePerBlock = 1
		}, errorsmod.Wrap(ibctm.ErrDelayPeriodNotPassed, "failed packet commitment verification for client (07-tendermint-0): cannot verify packet until height: 1-1000000016, current height: 1-17")},
		{"client state not found- changed client ID", func() {
			path.EndpointB.UpdateConnection(func(c *types.ConnectionEnd) { c.ClientId = ibctesting.InvalidID })
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed packet commitment verification for client (07-tendermint-0): client state height < proof height ({1 17} < {1 22}), please ensure the client has been updated")},
		{"verification failed - changed packet commitment state", func() {
			packet.Data = []byte(ibctesting.InvalidID)
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed packet commitment verification for client (07-tendermint-0): failed to verify membership proof at index 0: provided value doesn't match proof")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointB.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointB.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, 0)

			// reset variables
			heightDiff = 0
			delayTimePeriod = 0
			timePerBlock = 0
			tc.malleate()

			connection := path.EndpointB.GetConnection()
			connection.DelayPeriod = delayTimePeriod
			commitmentKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			proof, proofHeight := suite.chainA.QueryProof(commitmentKey)

			// set time per block param
			if timePerBlock != 0 {
				suite.chainB.App.GetIBCKeeper().ConnectionKeeper.SetParams(suite.chainB.GetContext(), types.NewParams(timePerBlock))
			}

			commitment := channeltypes.CommitPacket(suite.chainB.App.GetIBCKeeper().Codec(), packet)
			err = suite.chainB.App.GetIBCKeeper().ConnectionKeeper.VerifyPacketCommitment(
				suite.chainB.GetContext(), connection, malleateHeight(proofHeight, heightDiff), proof,
				packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence(), commitment,
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestVerifyPacketAcknowledgement has chainA verify the acknowledgement on
// channelB. The channels on chainA and chainB are fully opened and a packet
// is sent from chainA to chainB and received.
func (suite *KeeperTestSuite) TestVerifyPacketAcknowledgement() {
	var (
		path            *ibctesting.Path
		ack             exported.Acknowledgement
		heightDiff      uint64
		delayTimePeriod uint64
		timePerBlock    uint64
	)

	cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{"verification success", func() {}, nil},
		{"verification success: delay period passed", func() {
			delayTimePeriod = uint64(1 * time.Second.Nanoseconds())
		}, nil},
		{"delay time period has not passed", func() {
			delayTimePeriod = uint64(1 * time.Hour.Nanoseconds())
		}, errorsmod.Wrap(ibctm.ErrDelayPeriodNotPassed, "failed packet acknowledgement verification for client (07-tendermint-0): cannot verify packet until time: 1577934160000000000, current time: 1577930565000000000")},
		{"delay block period has not passed", func() {
			// make timePerBlock 1 nanosecond so that block delay is not passed.
			// must also set a non-zero time delay to ensure block delay is enforced.
			delayTimePeriod = uint64(1 * time.Second.Nanoseconds())
			timePerBlock = 1
		}, errorsmod.Wrap(ibctm.ErrDelayPeriodNotPassed, "failed packet acknowledgement verification for client (07-tendermint-0): cannot verify packet until height: 1-1000000018, current height: 1-19")},
		{"client state not found- changed client ID", func() {
			path.EndpointA.UpdateConnection(func(c *types.ConnectionEnd) { c.ClientId = ibctesting.InvalidID })
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed packet acknowledgement verification for client (07-tendermint-0): client state height < proof height ({1 19} < {1 24}), please ensure the client has been updated")},
		{"verification failed - changed acknowledgement", func() {
			ack = ibcmock.MockFailAcknowledgement
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed packet acknowledgement verification for client (07-tendermint-0): failed to verify membership proof at index 0: provided value doesn't match proof")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()                 // reset
			ack = ibcmock.MockAcknowledgement // must be explicitly changed

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			// send and receive packet
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// increment receiving chain's (chainB) time by 2 hour to always pass receive
			suite.coordinator.IncrementTimeBy(time.Hour * 2)
			suite.coordinator.CommitBlock(suite.chainB)

			packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)

			packetAckKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			proof, proofHeight := suite.chainB.QueryProof(packetAckKey)

			// reset variables
			heightDiff = 0
			delayTimePeriod = 0
			timePerBlock = 0
			tc.malleate()

			connection := path.EndpointA.GetConnection()
			connection.DelayPeriod = delayTimePeriod

			// set time per block param
			if timePerBlock != 0 {
				suite.chainA.App.GetIBCKeeper().ConnectionKeeper.SetParams(suite.chainA.GetContext(), types.NewParams(timePerBlock))
			}

			err = suite.chainA.App.GetIBCKeeper().ConnectionKeeper.VerifyPacketAcknowledgement(
				suite.chainA.GetContext(), connection, malleateHeight(proofHeight, heightDiff), proof,
				packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(), ack.Acknowledgement(),
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestVerifyPacketReceiptAbsence has chainA verify the receipt
// absence on channelB. The channels on chainA and chainB are fully opened and
// a packet is sent from chainA to chainB and not received.
func (suite *KeeperTestSuite) TestVerifyPacketReceiptAbsence() {
	var (
		path            *ibctesting.Path
		packet          channeltypes.Packet
		heightDiff      uint64
		delayTimePeriod uint64
		timePerBlock    uint64
	)

	cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{"verification success", func() {}, nil},
		{"verification success: delay period passed", func() {
			delayTimePeriod = uint64(1 * time.Second.Nanoseconds())
		}, nil},
		{"delay time period has not passed", func() {
			delayTimePeriod = uint64(1 * time.Hour.Nanoseconds())
		}, errorsmod.Wrap(ibctm.ErrDelayPeriodNotPassed, "failed packet commitment verification for client (07-tendermint-0): cannot verify packet until time: 1577926940000000000, current time: 1577923345000000000")},
		{"delay block period has not passed", func() {
			// make timePerBlock 1 nanosecond so that block delay is not passed.
			// must also set a non-zero time delay to ensure block delay is enforced.
			delayTimePeriod = uint64(1 * time.Second.Nanoseconds())
			timePerBlock = 1
		}, errorsmod.Wrap(ibctm.ErrDelayPeriodNotPassed, "failed packet commitment verification for client (07-tendermint-0): cannot verify packet until height: 1-1000000016, current height: 1-17")},
		{"client state not found - changed client ID", func() {
			path.EndpointA.UpdateConnection(func(c *types.ConnectionEnd) { c.ClientId = ibctesting.InvalidID })
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed packet commitment verification for client (07-tendermint-0): client state height < proof height ({1 17} < {1 22}), please ensure the client has been updated")},
		{"verification failed - acknowledgement was received", func() {
			// increment receiving chain's (chainB) time by 2 hour to always pass receive
			suite.coordinator.IncrementTimeBy(time.Hour * 2)
			suite.coordinator.CommitBlock(suite.chainB)

			err := path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed packet commitment verification for client (07-tendermint-0): failed to verify membership proof at index 0: provided value doesn't match proof")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			// send, only receive in malleate if applicable
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, 0)

			// reset variables
			heightDiff = 0
			delayTimePeriod = 0
			timePerBlock = 0
			tc.malleate()

			connection := path.EndpointA.GetConnection()
			connection.DelayPeriod = delayTimePeriod

			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			if clientState.FrozenHeight.IsZero() {
				// need to update height to prove absence or receipt
				suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
			}

			packetReceiptKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			proof, proofHeight := suite.chainB.QueryProof(packetReceiptKey)

			// set time per block param
			if timePerBlock != 0 {
				suite.chainA.App.GetIBCKeeper().ConnectionKeeper.SetParams(suite.chainA.GetContext(), types.NewParams(timePerBlock))
			}

			err = suite.chainA.App.GetIBCKeeper().ConnectionKeeper.VerifyPacketReceiptAbsence(
				suite.chainA.GetContext(), connection, malleateHeight(proofHeight, heightDiff), proof,
				packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestVerifyNextSequenceRecv has chainA verify the next sequence receive on
// channelB. The channels on chainA and chainB are fully opened and a packet
// is sent from chainA to chainB and received.
func (suite *KeeperTestSuite) TestVerifyNextSequenceRecv() {
	var (
		path            *ibctesting.Path
		heightDiff      uint64
		delayTimePeriod uint64
		timePerBlock    uint64
		offsetSeq       uint64
	)

	cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{"verification success", func() {}, nil},
		{"verification success: delay period passed", func() {
			delayTimePeriod = uint64(1 * time.Second.Nanoseconds())
		}, nil},
		{"delay time period has not passed", func() {
			delayTimePeriod = uint64(1 * time.Hour.Nanoseconds())
		}, errorsmod.Wrap(ibctm.ErrDelayPeriodNotPassed, "failed packet commitment verification for client (07-tendermint-0): cannot verify packet until time: 1577926940000000000, current time: 1577923345000000000")},
		{"delay block period has not passed", func() {
			// make timePerBlock 1 nanosecond so that block delay is not passed.
			// must also set a non-zero time delay to ensure block delay is enforced.
			delayTimePeriod = uint64(1 * time.Second.Nanoseconds())
			timePerBlock = 1
		}, errorsmod.Wrap(ibctm.ErrDelayPeriodNotPassed, "failed packet commitment verification for client (07-tendermint-0): cannot verify packet until height: 1-1000000016, current height: 1-17")},
		{"client state not found- changed client ID", func() {
			path.EndpointA.UpdateConnection(func(c *types.ConnectionEnd) { c.ClientId = ibctesting.InvalidID })
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed packet commitment verification for client (07-tendermint-0): client state height < proof height ({1 17} < {1 22}), please ensure the client has been updated")},
		{"verification failed - wrong expected next seq recv", func() {
			offsetSeq = 1
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed packet commitment verification for client (07-tendermint-0): failed to verify membership proof at index 0: provided value doesn't match proof")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			// send and receive packet
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)

			// increment receiving chain's (chainB) time by 2 hour to always pass receive
			suite.coordinator.IncrementTimeBy(time.Hour * 2)
			suite.coordinator.CommitBlock(suite.chainB)

			packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			suite.Require().NoError(err)

			nextSeqRecvKey := host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
			proof, proofHeight := suite.chainB.QueryProof(nextSeqRecvKey)

			// reset variables
			heightDiff = 0
			delayTimePeriod = 0
			timePerBlock = 0
			tc.malleate()

			// set time per block param
			if timePerBlock != 0 {
				suite.chainA.App.GetIBCKeeper().ConnectionKeeper.SetParams(suite.chainA.GetContext(), types.NewParams(timePerBlock))
			}

			connection := path.EndpointA.GetConnection()
			connection.DelayPeriod = delayTimePeriod
			err = suite.chainA.App.GetIBCKeeper().ConnectionKeeper.VerifyNextSequenceRecv(
				suite.chainA.GetContext(), connection, malleateHeight(proofHeight, heightDiff), proof,
				packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence()+offsetSeq,
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestVerifyUpgradeErrorReceipt() {
	var (
		path         *ibctesting.Path
		upgradeError *channeltypes.UpgradeError
	)

	cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			name:     "success",
			malleate: func() {},
			expErr:   nil,
		},
		{
			name: "fails when client state is frozen",
			malleate: func() {
				clientState, ok := path.EndpointB.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)
				clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
				path.EndpointB.SetClientState(clientState)
			},
			expErr: errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen"),
		},
		{
			name: "fails with bad client id",
			malleate: func() {
				path.EndpointB.UpdateConnection(func(c *types.ConnectionEnd) { c.ClientId = ibctesting.InvalidID })
			},
			expErr: errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (IDisInvalid) status is Unauthorized"),
		},
		{
			name: "verification fails when the key does not exist",
			malleate: func() {
				suite.chainA.DeleteKey(host.ChannelUpgradeErrorKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				suite.coordinator.CommitBlock(suite.chainA)
			},
			expErr: errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed upgrade error receipt verification for client (07-tendermint-0): client state height < proof height ({1 17} < {1 18}), please ensure the client has been updated"),
		},
		{
			name: "verification fails when message differs",
			malleate: func() {
				originalSequence := upgradeError.GetErrorReceipt().Sequence
				upgradeError = channeltypes.NewUpgradeError(originalSequence, fmt.Errorf("new error"))
			},
			expErr: errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed upgrade error receipt verification for client (07-tendermint-0): failed to verify membership proof at index 0: provided value doesn't match proof"),
		},
	}

	for _, tc := range cases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			upgradeError = channeltypes.NewUpgradeError(1, channeltypes.ErrInvalidChannel)
			suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.WriteErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgradeError)

			suite.chainA.Coordinator.CommitBlock(suite.chainA)
			suite.Require().NoError(path.EndpointB.UpdateClient())

			tc.malleate()

			upgradeErrorReceiptKey := host.ChannelUpgradeErrorKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proof, proofHeight := suite.chainA.QueryProof(upgradeErrorReceiptKey)

			err := suite.chainB.GetSimApp().IBCKeeper.ConnectionKeeper.VerifyChannelUpgradeError(suite.chainB.GetContext(), path.EndpointB.GetConnection(), proofHeight, proof, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgradeError.GetErrorReceipt())

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestVerifyUpgrade() {
	var (
		path    *ibctesting.Path
		upgrade channeltypes.Upgrade
	)

	cases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			name:     "success",
			malleate: func() {},
			expErr:   nil,
		},
		{
			name: "fails when client state is frozen",
			malleate: func() {
				clientState, ok := path.EndpointB.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)
				clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
				path.EndpointB.SetClientState(clientState)
			},
			expErr: errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen"),
		},
		{
			name: "fails with bad client id",
			malleate: func() {
				path.EndpointB.UpdateConnection(func(c *types.ConnectionEnd) { c.ClientId = ibctesting.InvalidID })
			},
			expErr: errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (IDisInvalid) status is Unauthorized"),
		},
		{
			name: "fails when the upgrade field is different",
			malleate: func() {
				upgrade.Fields.Ordering = channeltypes.ORDERED
			},
			expErr: errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed upgrade verification for client (07-tendermint-0) on channel (channel-7): failed to verify membership proof at index 0: provided value doesn't match proof"),
		},
	}

	for _, tc := range cases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			upgrade = channeltypes.NewUpgrade(
				channeltypes.NewUpgradeFields(channeltypes.UNORDERED, []string{path.EndpointA.ConnectionID}, "v1.0.0"),
				channeltypes.NewTimeout(clienttypes.ZeroHeight(), 100000),
				0,
			)

			suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgrade)

			suite.chainA.Coordinator.CommitBlock(suite.chainA)
			suite.Require().NoError(path.EndpointB.UpdateClient())

			tc.malleate()

			channelUpgradeKey := host.ChannelUpgradeKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proof, proofHeight := suite.chainA.QueryProof(channelUpgradeKey)

			err := suite.chainB.GetSimApp().IBCKeeper.ConnectionKeeper.VerifyChannelUpgrade(suite.chainB.GetContext(), path.EndpointB.GetConnection(), proofHeight, proof, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgrade)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func malleateHeight(height exported.Height, diff uint64) exported.Height {
	return clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+diff)
}
