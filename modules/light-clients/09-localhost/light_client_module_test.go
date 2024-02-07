package localhost_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	localhost "github.com/cosmos/ibc-go/v8/modules/light-clients/09-localhost"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/cosmos/ibc-go/v8/testing/mock"
)

func (s *LocalhostTestSuite) TestStatus() {
	lightClientModule, found := s.chain.GetSimApp().IBCKeeper.ClientKeeper.GetRouter().GetRoute(exported.LocalhostClientID)
	s.Require().True(found)
	s.Require().Equal(exported.Active, lightClientModule.Status(s.chain.GetContext(), exported.LocalhostClientID))
}

func (s *LocalhostTestSuite) TestVerifyMembership() {
	var (
		path  exported.Path
		value []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: client state verification",
			func() {
				clientState := s.chain.GetClientState(exported.LocalhostClientID)

				merklePath := commitmenttypes.NewMerklePath(host.FullClientStatePath(exported.LocalhostClientID))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = clienttypes.MustMarshalClientState(s.chain.Codec, clientState)
			},
			true,
		},
		{
			"success: connection state verification",
			func() {
				connectionEnd := connectiontypes.NewConnectionEnd(
					connectiontypes.OPEN,
					exported.LocalhostClientID,
					connectiontypes.NewCounterparty(exported.LocalhostClientID, exported.LocalhostConnectionID, s.chain.GetPrefix()),
					connectiontypes.GetCompatibleVersions(), 0,
				)

				s.chain.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(s.chain.GetContext(), exported.LocalhostConnectionID, connectionEnd)

				merklePath := commitmenttypes.NewMerklePath(host.ConnectionPath(exported.LocalhostConnectionID))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = s.chain.Codec.MustMarshal(&connectionEnd)
			},
			true,
		},
		{
			"success: channel state verification",
			func() {
				channel := channeltypes.NewChannel(
					channeltypes.OPEN,
					channeltypes.UNORDERED,
					channeltypes.NewCounterparty(mock.PortID, ibctesting.FirstChannelID),
					[]string{exported.LocalhostConnectionID},
					mock.Version,
				)

				s.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(s.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, channel)

				merklePath := commitmenttypes.NewMerklePath(host.ChannelPath(mock.PortID, ibctesting.FirstChannelID))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = s.chain.Codec.MustMarshal(&channel)
			},
			true,
		},
		{
			"success: next sequence recv verification",
			func() {
				nextSeqRecv := uint64(100)
				s.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetNextSequenceRecv(s.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, nextSeqRecv)

				merklePath := commitmenttypes.NewMerklePath(host.NextSequenceRecvPath(mock.PortID, ibctesting.FirstChannelID))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = sdk.Uint64ToBigEndian(nextSeqRecv)
			},
			true,
		},
		{
			"success: packet commitment verification",
			func() {
				packet := channeltypes.NewPacket(
					ibctesting.MockPacketData,
					1,
					ibctesting.MockPort,
					ibctesting.FirstChannelID,
					ibctesting.MockPort,
					ibctesting.FirstChannelID,
					clienttypes.NewHeight(0, 10),
					0,
				)

				commitmentBz := channeltypes.CommitPacket(s.chain.Codec, packet)
				s.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetPacketCommitment(s.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, 1, commitmentBz)

				merklePath := commitmenttypes.NewMerklePath(host.PacketCommitmentPath(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence()))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = commitmentBz
			},
			true,
		},
		{
			"success: packet acknowledgement verification",
			func() {
				s.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(s.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, 1, ibctesting.MockAcknowledgement)

				merklePath := commitmenttypes.NewMerklePath(host.PacketAcknowledgementPath(mock.PortID, ibctesting.FirstChannelID, 1))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = ibctesting.MockAcknowledgement
			},
			true,
		},
		{
			"invalid type for key path",
			func() {
				path = mock.KeyPath{}
			},
			false,
		},
		{
			"key path has too many elements",
			func() {
				path = commitmenttypes.NewMerklePath("ibc", "test", "key")
			},
			false,
		},
		{
			"no value found at provided key path",
			func() {
				merklePath := commitmenttypes.NewMerklePath(host.PacketAcknowledgementPath(mock.PortID, ibctesting.FirstChannelID, 100))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = ibctesting.MockAcknowledgement
			},
			false,
		},
		{
			"invalid value, bytes are not equal",
			func() {
				channel := channeltypes.NewChannel(
					channeltypes.OPEN,
					channeltypes.UNORDERED,
					channeltypes.NewCounterparty(mock.PortID, ibctesting.FirstChannelID),
					[]string{exported.LocalhostConnectionID},
					mock.Version,
				)

				s.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(s.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, channel)

				merklePath := commitmenttypes.NewMerklePath(host.ChannelPath(mock.PortID, ibctesting.FirstChannelID))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath

				// modify the channel before marshalling to value bz
				channel.State = channeltypes.CLOSED
				value = s.chain.Codec.MustMarshal(&channel)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()

			tc.malleate()

			lightClientModule, found := s.chain.GetSimApp().IBCKeeper.ClientKeeper.GetRouter().GetRoute(exported.LocalhostClientID)
			s.Require().True(found)

			err := lightClientModule.VerifyMembership(
				s.chain.GetContext(),
				exported.LocalhostClientID,
				clienttypes.ZeroHeight(),
				0, 0, // use zero values for delay periods
				localhost.SentinelProof,
				path,
				value,
			)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *LocalhostTestSuite) TestVerifyNonMembership() {
	var path exported.Path

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: packet receipt absence verification",
			func() {
				merklePath := commitmenttypes.NewMerklePath(host.PacketReceiptPath(mock.PortID, ibctesting.FirstChannelID, 1))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
			},
			true,
		},
		{
			"packet receipt absence verification fails",
			func() {
				s.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetPacketReceipt(s.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, 1)

				merklePath := commitmenttypes.NewMerklePath(host.PacketReceiptPath(mock.PortID, ibctesting.FirstChannelID, 1))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
			},
			false,
		},
		{
			"invalid type for key path",
			func() {
				path = mock.KeyPath{}
			},
			false,
		},
		{
			"key path has too many elements",
			func() {
				path = commitmenttypes.NewMerklePath("ibc", "test", "key")
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()

			tc.malleate()

			lightClientModule, found := s.chain.GetSimApp().IBCKeeper.ClientKeeper.GetRouter().GetRoute(exported.LocalhostClientID)
			s.Require().True(found)

			err := lightClientModule.VerifyNonMembership(
				s.chain.GetContext(),
				exported.LocalhostClientID,
				clienttypes.ZeroHeight(),
				0, 0, // use zero values for delay periods
				localhost.SentinelProof,
				path,
			)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
