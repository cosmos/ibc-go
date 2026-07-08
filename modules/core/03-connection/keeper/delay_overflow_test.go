package keeper_test

import (
	connectiontypes "github.com/cosmos/ibc-go/v11/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v11/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v11/modules/core/24-host"
	ibctm "github.com/cosmos/ibc-go/v11/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
)

func (s *KeeperTestSuite) TestVerifyPacketCommitmentDelayPeriodOverflowFailsClosed() {
	s.SetupTest()

	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.Setup()

	sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, 0, ibctesting.MockPacketData)
	s.Require().NoError(err)

	packet := channeltypes.NewPacket(
		ibctesting.MockPacketData,
		sequence,
		path.EndpointA.ChannelConfig.PortID,
		path.EndpointA.ChannelID,
		path.EndpointB.ChannelConfig.PortID,
		path.EndpointB.ChannelID,
		defaultTimeoutHeight,
		0,
	)

	connection := path.EndpointB.GetConnection()
	connection.DelayPeriod = ^uint64(0)

	// Make the block-delay side evaluate to one block so this isolates the time-delay overflow.
	s.chainB.App.GetIBCKeeper().ConnectionKeeper.SetParams(s.chainB.GetContext(), connectiontypes.NewParams(^uint64(0)))

	commitmentKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := s.chainA.QueryProof(commitmentKey)
	commitment := channeltypes.CommitPacket(packet)

	err = s.chainB.App.GetIBCKeeper().ConnectionKeeper.VerifyPacketCommitment(
		s.chainB.GetContext(),
		connection,
		proofHeight,
		proof,
		packet.GetSourcePort(),
		packet.GetSourceChannel(),
		packet.GetSequence(),
		commitment,
	)
	s.Require().ErrorIs(err, ibctm.ErrDelayPeriodNotPassed)
}

func (s *KeeperTestSuite) TestVerifyMembershipDelayBlockPeriodOverflowFailsClosed() {
	s.SetupTest()

	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.Setup()

	sequence, err := path.EndpointA.SendPacket(defaultTimeoutHeight, 0, ibctesting.MockPacketData)
	s.Require().NoError(err)

	packet := channeltypes.NewPacket(
		ibctesting.MockPacketData,
		sequence,
		path.EndpointA.ChannelConfig.PortID,
		path.EndpointA.ChannelID,
		path.EndpointB.ChannelConfig.PortID,
		path.EndpointB.ChannelID,
		defaultTimeoutHeight,
		0,
	)

	connection := path.EndpointB.GetConnection()
	commitmentKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := s.chainA.QueryProof(commitmentKey)
	commitment := channeltypes.CommitPacket(packet)
	merklePath := commitmenttypes.NewMerklePath(commitmentKey)
	merklePath, err = commitmenttypes.ApplyPrefix(connection.Counterparty.Prefix, merklePath)
	s.Require().NoError(err)

	err = s.chainB.App.GetIBCKeeper().ClientKeeper.VerifyMembership(
		s.chainB.GetContext(),
		connection.ClientId,
		proofHeight,
		0,
		^uint64(0),
		proof,
		merklePath,
		commitment,
	)
	s.Require().ErrorIs(err, ibctm.ErrDelayPeriodNotPassed)
}
