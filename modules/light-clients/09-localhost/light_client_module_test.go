package localhost_test

import (
	"errors"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	localhost "github.com/cosmos/ibc-go/v10/modules/light-clients/09-localhost"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/mock"
)

type LocalhostTestSuite struct {
	testifysuite.Suite

	coordinator ibctesting.Coordinator
	chain       *ibctesting.TestChain
}

func (s *LocalhostTestSuite) SetupTest() {
	s.coordinator = *ibctesting.NewCoordinator(s.T(), 1)
	s.chain = s.coordinator.GetChain(ibctesting.GetChainID(1))
}

func TestLocalhostTestSuite(t *testing.T) {
	testifysuite.Run(t, new(LocalhostTestSuite))
}

func (s *LocalhostTestSuite) TestInitialize() {
	lightClientModule, err := s.chain.App.GetIBCKeeper().ClientKeeper.Route(s.chain.GetContext(), exported.LocalhostClientID)
	s.Require().NoError(err)

	err = lightClientModule.Initialize(s.chain.GetContext(), exported.LocalhostClientID, nil, nil)
	s.Require().Error(err)
}

func (s *LocalhostTestSuite) TestVerifyClientMessage() {
	lightClientModule, err := s.chain.App.GetIBCKeeper().ClientKeeper.Route(s.chain.GetContext(), exported.LocalhostClientID)
	s.Require().NoError(err)

	err = lightClientModule.VerifyClientMessage(s.chain.GetContext(), exported.LocalhostClientID, nil)
	s.Require().Error(err)
}

func (s *LocalhostTestSuite) TestVerifyCheckForMisbehaviour() {
	lightClientModule, err := s.chain.App.GetIBCKeeper().ClientKeeper.Route(s.chain.GetContext(), exported.LocalhostClientID)
	s.Require().NoError(err)

	s.Require().False(lightClientModule.CheckForMisbehaviour(s.chain.GetContext(), exported.LocalhostClientID, nil))
}

func (s *LocalhostTestSuite) TestUpdateState() {
	lightClientModule, err := s.chain.App.GetIBCKeeper().ClientKeeper.Route(s.chain.GetContext(), exported.LocalhostClientID)
	s.Require().NoError(err)

	heights := lightClientModule.UpdateState(s.chain.GetContext(), exported.LocalhostClientID, nil)

	expHeight := clienttypes.NewHeight(1, uint64(s.chain.GetContext().BlockHeight()))
	s.Require().True(heights[0].EQ(expHeight))
}

func (s *LocalhostTestSuite) TestVerifyMembership() {
	var (
		path  exported.Path
		value []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
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

				merklePath := commitmenttypes.NewMerklePath(host.ConnectionKey(exported.LocalhostConnectionID))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = s.chain.Codec.MustMarshal(&connectionEnd)
			},
			nil,
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

				merklePath := commitmenttypes.NewMerklePath(host.ChannelKey(mock.PortID, ibctesting.FirstChannelID))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = s.chain.Codec.MustMarshal(&channel)
			},
			nil,
		},
		{
			"success: next sequence recv verification",
			func() {
				nextSeqRecv := uint64(100)
				s.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetNextSequenceRecv(s.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, nextSeqRecv)

				merklePath := commitmenttypes.NewMerklePath(host.NextSequenceRecvKey(mock.PortID, ibctesting.FirstChannelID))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = sdk.Uint64ToBigEndian(nextSeqRecv)
			},
			nil,
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

				commitmentBz := channeltypes.CommitPacket(packet)
				s.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetPacketCommitment(s.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, 1, commitmentBz)

				merklePath := commitmenttypes.NewMerklePath(host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence()))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = commitmentBz
			},
			nil,
		},
		{
			"success: packet acknowledgement verification",
			func() {
				s.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(s.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, 1, ibctesting.MockAcknowledgement)

				merklePath := commitmenttypes.NewMerklePath(host.PacketAcknowledgementKey(mock.PortID, ibctesting.FirstChannelID, 1))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = ibctesting.MockAcknowledgement
			},
			nil,
		},
		{
			"failure: invalid type for key path",
			func() {
				path = mock.KeyPath{}
			},
			errors.New("expected v2.MerklePath, got mock.KeyPath: invalid type"),
		},
		{
			"failure: key path has too many elements",
			func() {
				path = commitmenttypes.NewMerklePath([]byte("ibc"), []byte("test"), []byte("key"))
			},
			errors.New("invalid path"),
		},
		{
			"failure: no value found at provided key path",
			func() {
				merklePath := commitmenttypes.NewMerklePath(host.PacketAcknowledgementKey(mock.PortID, ibctesting.FirstChannelID, 100))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
				value = ibctesting.MockAcknowledgement
			},
			errors.New("value not found for path"),
		},
		{
			"failure: invalid value, bytes are not equal",
			func() {
				channel := channeltypes.NewChannel(
					channeltypes.OPEN,
					channeltypes.UNORDERED,
					channeltypes.NewCounterparty(mock.PortID, ibctesting.FirstChannelID),
					[]string{exported.LocalhostConnectionID},
					mock.Version,
				)

				s.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(s.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, channel)

				merklePath := commitmenttypes.NewMerklePath(host.ChannelKey(mock.PortID, ibctesting.FirstChannelID))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath

				// modify the channel before marshalling to value bz
				channel.State = channeltypes.CLOSED
				value = s.chain.Codec.MustMarshal(&channel)
			},
			errors.New("value provided does not equal value stored at path"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			tc.malleate()

			lightClientModule, err := s.chain.App.GetIBCKeeper().ClientKeeper.Route(s.chain.GetContext(), exported.LocalhostClientID)
			s.Require().NoError(err)

			err = lightClientModule.VerifyMembership(
				s.chain.GetContext(),
				exported.LocalhostClientID,
				clienttypes.ZeroHeight(),
				0, 0, // use zero values for delay periods
				localhost.SentinelProof,
				path,
				value,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}

func (s *LocalhostTestSuite) TestVerifyNonMembership() {
	var path exported.Path

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: packet receipt absence verification",
			func() {
				merklePath := commitmenttypes.NewMerklePath(host.PacketReceiptKey(mock.PortID, ibctesting.FirstChannelID, 1))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
			},
			nil,
		},
		{
			"packet receipt absence verification fails",
			func() {
				s.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetPacketReceipt(s.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, 1)

				merklePath := commitmenttypes.NewMerklePath(host.PacketReceiptKey(mock.PortID, ibctesting.FirstChannelID, 1))
				merklePath, err := commitmenttypes.ApplyPrefix(s.chain.GetPrefix(), merklePath)
				s.Require().NoError(err)

				path = merklePath
			},
			errors.New("non-membership verification failed"),
		},
		{
			"invalid type for key path",
			func() {
				path = mock.KeyPath{}
			},
			errors.New("expected v2.MerklePath, got mock.KeyPath: invalid type"),
		},
		{
			"key path has too many elements",
			func() {
				path = commitmenttypes.NewMerklePath([]byte("ibc"), []byte("test"), []byte("key"))
			},
			errors.New("invalid path"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			tc.malleate()

			lightClientModule, err := s.chain.App.GetIBCKeeper().ClientKeeper.Route(s.chain.GetContext(), exported.LocalhostClientID)
			s.Require().NoError(err)

			err = lightClientModule.VerifyNonMembership(
				s.chain.GetContext(),
				exported.LocalhostClientID,
				clienttypes.ZeroHeight(),
				0, 0, // use zero values for delay periods
				localhost.SentinelProof,
				path,
			)

			if tc.expError == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorContains(err, tc.expError.Error())
			}
		})
	}
}

func (s *LocalhostTestSuite) TestStatus() {
	lightClientModule, err := s.chain.App.GetIBCKeeper().ClientKeeper.Route(s.chain.GetContext(), exported.LocalhostClientID)
	s.Require().NoError(err)

	s.Require().Equal(exported.Active, lightClientModule.Status(s.chain.GetContext(), exported.LocalhostClientID))
}

func (s *LocalhostTestSuite) TestGetTimestampAtHeight() {
	lightClientModule, err := s.chain.App.GetIBCKeeper().ClientKeeper.Route(s.chain.GetContext(), exported.LocalhostClientID)
	s.Require().NoError(err)

	ctx := s.chain.GetContext()
	timestamp, err := lightClientModule.TimestampAtHeight(ctx, exported.LocalhostClientID, nil)
	s.Require().NoError(err)
	s.Require().Equal(uint64(ctx.BlockTime().UnixNano()), timestamp)
}

func (s *LocalhostTestSuite) TestRecoverClient() {
	lightClientModule, err := s.chain.App.GetIBCKeeper().ClientKeeper.Route(s.chain.GetContext(), exported.LocalhostClientID)
	s.Require().NoError(err)

	err = lightClientModule.RecoverClient(s.chain.GetContext(), exported.LocalhostClientID, exported.LocalhostClientID)
	s.Require().Error(err)
}

func (s *LocalhostTestSuite) TestVerifyUpgradeAndUpdateState() {
	lightClientModule, err := s.chain.App.GetIBCKeeper().ClientKeeper.Route(s.chain.GetContext(), exported.LocalhostClientID)
	s.Require().NoError(err)

	err = lightClientModule.VerifyUpgradeAndUpdateState(s.chain.GetContext(), exported.LocalhostClientID, nil, nil, nil, nil)
	s.Require().Error(err)
}
