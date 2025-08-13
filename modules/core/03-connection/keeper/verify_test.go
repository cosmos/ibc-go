package keeper_test

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

var defaultTimeoutHeight = clienttypes.NewHeight(1, 100000)

// TestVerifyConnectionState verifies the connection state of the connection
// on chainB. The connections on chainA and chainB are fully opened.
func (s *KeeperTestSuite) TestVerifyConnectionState() {
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
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed connection state verification for client (07-tendermint-0): client state height < proof height ({1 9} < {1 14}), please ensure the client has been updated")},
		{"verification failed - connection state is different than proof", func() {
			path.EndpointA.UpdateConnection(func(c *types.ConnectionEnd) { c.State = types.TRYOPEN })
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed connection state verification for client (07-tendermint-0): client state height < proof height ({1 9} < {1 14}), please ensure the client has been updated")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupConnections()

			connectionKey := host.ConnectionKey(path.EndpointB.ConnectionID)
			proof, proofHeight := s.chainB.QueryProof(connectionKey)

			tc.malleate()

			connection := path.EndpointA.GetConnection()

			expectedConnection := path.EndpointB.GetConnection()

			err := s.chainA.App.GetIBCKeeper().ConnectionKeeper.VerifyConnectionState(
				s.chainA.GetContext(), connection,
				malleateHeight(proofHeight, heightDiff), proof, path.EndpointB.ConnectionID, expectedConnection,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestVerifyChannelState verifies the channel state of the channel on
// chainB. The channels on chainA and chainB are fully opened.
func (s *KeeperTestSuite) TestVerifyChannelState() {
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
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed channel state verification for client (07-tendermint-0): client state height < proof height ({1 15} < {1 20}), please ensure the client has been updated")},
		{"verification failed - changed channel state", func() {
			path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.TRYOPEN })
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed channel state verification for client (07-tendermint-0): client state height < proof height ({1 15} < {1 20}), please ensure the client has been updated")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.Setup()

			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			proof, proofHeight := s.chainB.QueryProof(channelKey)

			tc.malleate()
			connection := path.EndpointA.GetConnection()

			channel := path.EndpointB.GetChannel()

			err := s.chainA.App.GetIBCKeeper().ConnectionKeeper.VerifyChannelState(
				s.chainA.GetContext(), connection, malleateHeight(proofHeight, heightDiff), proof,
				path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, channel,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestVerifyPacketCommitment has chainB verify the packet commitment
// on channelA. The channels on chainA and chainB are fully opened and a
// packet is sent from chainA to chainB, but has not been received.
func (s *KeeperTestSuite) TestVerifyPacketCommitment() {
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
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed packet commitment verification for client (07-tendermint-0): client state height < proof height ({1 17} < {1 22}), please ensure the client has been updated")},
		{"verification failed - changed packet commitment state", func() {
			packet.Data = []byte(ibctesting.InvalidID)
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed packet commitment verification for client (07-tendermint-0): failed to verify membership proof at index 0: provided value doesn't match proof")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointB.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointB.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.Setup()

			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, 0)

			// reset variables
			heightDiff = 0
			delayTimePeriod = 0
			timePerBlock = 0
			tc.malleate()

			connection := path.EndpointB.GetConnection()
			connection.DelayPeriod = delayTimePeriod
			commitmentKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			proof, proofHeight := s.chainA.QueryProof(commitmentKey)

			// set time per block param
			if timePerBlock != 0 {
				s.chainB.App.GetIBCKeeper().ConnectionKeeper.SetParams(s.chainB.GetContext(), types.NewParams(timePerBlock))
			}

			commitment := channeltypes.CommitPacket(packet)
			err = s.chainB.App.GetIBCKeeper().ConnectionKeeper.VerifyPacketCommitment(
				s.chainB.GetContext(), connection, malleateHeight(proofHeight, heightDiff), proof,
				packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence(), commitment,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestVerifyPacketAcknowledgement has chainA verify the acknowledgement on
// channelB. The channels on chainA and chainB are fully opened and a packet
// is sent from chainA to chainB and received.
func (s *KeeperTestSuite) TestVerifyPacketAcknowledgement() {
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
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed packet acknowledgement verification for client (07-tendermint-0): client state height < proof height ({1 19} < {1 24}), please ensure the client has been updated")},
		{"verification failed - changed acknowledgement", func() {
			ack = ibcmock.MockFailAcknowledgement
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed packet acknowledgement verification for client (07-tendermint-0): failed to verify membership proof at index 0: provided value doesn't match proof")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()                     // reset
			ack = ibcmock.MockAcknowledgement // must be explicitly changed

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.Setup()

			// send and receive packet
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// increment receiving chain's (chainB) time by 2 hour to always pass receive
			s.coordinator.IncrementTimeBy(time.Hour * 2)
			s.coordinator.CommitBlock(s.chainB)

			packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			packetAckKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			proof, proofHeight := s.chainB.QueryProof(packetAckKey)

			// reset variables
			heightDiff = 0
			delayTimePeriod = 0
			timePerBlock = 0
			tc.malleate()

			connection := path.EndpointA.GetConnection()
			connection.DelayPeriod = delayTimePeriod

			// set time per block param
			if timePerBlock != 0 {
				s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetParams(s.chainA.GetContext(), types.NewParams(timePerBlock))
			}

			err = s.chainA.App.GetIBCKeeper().ConnectionKeeper.VerifyPacketAcknowledgement(
				s.chainA.GetContext(), connection, malleateHeight(proofHeight, heightDiff), proof,
				packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(), ack.Acknowledgement(),
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestVerifyPacketReceiptAbsence has chainA verify the receipt
// absence on channelB. The channels on chainA and chainB are fully opened and
// a packet is sent from chainA to chainB and not received.
func (s *KeeperTestSuite) TestVerifyPacketReceiptAbsence() {
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
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed packet commitment verification for client (07-tendermint-0): client state height < proof height ({1 17} < {1 22}), please ensure the client has been updated")},
		{"verification failed - acknowledgement was received", func() {
			// increment receiving chain's (chainB) time by 2 hour to always pass receive
			s.coordinator.IncrementTimeBy(time.Hour * 2)
			s.coordinator.CommitBlock(s.chainB)

			err := path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed packet commitment verification for client (07-tendermint-0): failed to verify membership proof at index 0: provided value doesn't match proof")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.Setup()

			// send, only receive in malleate if applicable
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, 0)

			// reset variables
			heightDiff = 0
			delayTimePeriod = 0
			timePerBlock = 0
			tc.malleate()

			connection := path.EndpointA.GetConnection()
			connection.DelayPeriod = delayTimePeriod

			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			if clientState.FrozenHeight.IsZero() {
				// need to update height to prove absence or receipt
				s.coordinator.CommitBlock(s.chainA, s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)
			}

			packetReceiptKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			proof, proofHeight := s.chainB.QueryProof(packetReceiptKey)

			// set time per block param
			if timePerBlock != 0 {
				s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetParams(s.chainA.GetContext(), types.NewParams(timePerBlock))
			}

			err = s.chainA.App.GetIBCKeeper().ConnectionKeeper.VerifyPacketReceiptAbsence(
				s.chainA.GetContext(), connection, malleateHeight(proofHeight, heightDiff), proof,
				packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

// TestVerifyNextSequenceRecv has chainA verify the next sequence receive on
// channelB. The channels on chainA and chainB are fully opened and a packet
// is sent from chainA to chainB and received.
func (s *KeeperTestSuite) TestVerifyNextSequenceRecv() {
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
		{"consensus state not found - increased proof height", func() {
			heightDiff = 5
		}, errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "failed packet commitment verification for client (07-tendermint-0): client state height < proof height ({1 17} < {1 22}), please ensure the client has been updated")},
		{"verification failed - wrong expected next seq recv", func() {
			offsetSeq = 1
		}, errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "failed packet commitment verification for client (07-tendermint-0): failed to verify membership proof at index 0: provided value doesn't match proof")},
		{"client status is not active - client is expired", func() {
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, errorsmod.Wrap(clienttypes.ErrClientNotActive, "client (07-tendermint-0) status is Frozen")},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.Setup()

			// send and receive packet
			sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)

			// increment receiving chain's (chainB) time by 2 hour to always pass receive
			s.coordinator.IncrementTimeBy(time.Hour * 2)
			s.coordinator.CommitBlock(s.chainB)

			packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, defaultTimeoutHeight, 0)
			err = path.EndpointB.RecvPacket(packet)
			s.Require().NoError(err)

			nextSeqRecvKey := host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
			proof, proofHeight := s.chainB.QueryProof(nextSeqRecvKey)

			// reset variables
			heightDiff = 0
			delayTimePeriod = 0
			timePerBlock = 0
			tc.malleate()

			// set time per block param
			if timePerBlock != 0 {
				s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetParams(s.chainA.GetContext(), types.NewParams(timePerBlock))
			}

			connection := path.EndpointA.GetConnection()
			connection.DelayPeriod = delayTimePeriod
			err = s.chainA.App.GetIBCKeeper().ConnectionKeeper.VerifyNextSequenceRecv(
				s.chainA.GetContext(), connection, malleateHeight(proofHeight, heightDiff), proof,
				packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence()+offsetSeq,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func malleateHeight(height exported.Height, diff uint64) exported.Height {
	return clienttypes.NewHeight(height.GetRevisionNumber(), height.GetRevisionHeight()+diff)
}
