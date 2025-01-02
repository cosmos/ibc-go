package localhost_test

import (
	"errors"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	localhost "github.com/cosmos/ibc-go/v9/modules/light-clients/09-localhost"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
)

type LocalhostTestSuite struct {
	testifysuite.Suite

	coordinator ibctesting.Coordinator
	chain       *ibctesting.TestChain
}

func (suite *LocalhostTestSuite) SetupTest() {
	suite.coordinator = *ibctesting.NewCoordinator(suite.T(), 1)
	suite.chain = suite.coordinator.GetChain(ibctesting.GetChainID(1))
}

func TestLocalhostTestSuite(t *testing.T) {
	testifysuite.Run(t, new(LocalhostTestSuite))
}

func (suite *LocalhostTestSuite) TestInitialize() {
	lightClientModule, err := suite.chain.App.GetIBCKeeper().ClientKeeper.Route(suite.chain.GetContext(), exported.LocalhostClientID)
	suite.Require().NoError(err)

	err = lightClientModule.Initialize(suite.chain.GetContext(), exported.LocalhostClientID, nil, nil)
	suite.Require().Error(err)
}

func (suite *LocalhostTestSuite) TestVerifyClientMessage() {
	lightClientModule, err := suite.chain.App.GetIBCKeeper().ClientKeeper.Route(suite.chain.GetContext(), exported.LocalhostClientID)
	suite.Require().NoError(err)

	err = lightClientModule.VerifyClientMessage(suite.chain.GetContext(), exported.LocalhostClientID, nil)
	suite.Require().Error(err)
}

func (suite *LocalhostTestSuite) TestVerifyCheckForMisbehaviour() {
	lightClientModule, err := suite.chain.App.GetIBCKeeper().ClientKeeper.Route(suite.chain.GetContext(), exported.LocalhostClientID)
	suite.Require().NoError(err)

	suite.Require().False(lightClientModule.CheckForMisbehaviour(suite.chain.GetContext(), exported.LocalhostClientID, nil))
}

func (suite *LocalhostTestSuite) TestUpdateState() {
	lightClientModule, err := suite.chain.App.GetIBCKeeper().ClientKeeper.Route(suite.chain.GetContext(), exported.LocalhostClientID)
	suite.Require().NoError(err)

	heights := lightClientModule.UpdateState(suite.chain.GetContext(), exported.LocalhostClientID, nil)

	expHeight := clienttypes.NewHeight(1, uint64(suite.chain.GetContext().BlockHeight()))
	suite.Require().True(heights[0].EQ(expHeight))
}

func (suite *LocalhostTestSuite) TestVerifyMembership() {
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
					connectiontypes.NewCounterparty(exported.LocalhostClientID, exported.LocalhostConnectionID, suite.chain.GetPrefix()),
					connectiontypes.GetCompatibleVersions(), 0,
				)

				suite.chain.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chain.GetContext(), exported.LocalhostConnectionID, connectionEnd)

				merklePath := commitmenttypes.NewMerklePath(host.ConnectionKey(exported.LocalhostConnectionID))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				path = merklePath
				value = suite.chain.Codec.MustMarshal(&connectionEnd)
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

				suite.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(suite.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, channel)

				merklePath := commitmenttypes.NewMerklePath(host.ChannelKey(mock.PortID, ibctesting.FirstChannelID))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				path = merklePath
				value = suite.chain.Codec.MustMarshal(&channel)
			},
			nil,
		},
		{
			"success: next sequence recv verification",
			func() {
				nextSeqRecv := uint64(100)
				suite.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetNextSequenceRecv(suite.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, nextSeqRecv)

				merklePath := commitmenttypes.NewMerklePath(host.NextSequenceRecvKey(mock.PortID, ibctesting.FirstChannelID))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

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

				commitmentBz := channeltypes.CommitPacket(suite.chain.Codec, packet)
				suite.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, 1, commitmentBz)

				merklePath := commitmenttypes.NewMerklePath(host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence()))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				path = merklePath
				value = commitmentBz
			},
			nil,
		},
		{
			"success: packet acknowledgement verification",
			func() {
				suite.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetPacketAcknowledgement(suite.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, 1, ibctesting.MockAcknowledgement)

				merklePath := commitmenttypes.NewMerklePath(host.PacketAcknowledgementKey(mock.PortID, ibctesting.FirstChannelID, 1))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

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
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

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

				suite.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(suite.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, channel)

				merklePath := commitmenttypes.NewMerklePath(host.ChannelKey(mock.PortID, ibctesting.FirstChannelID))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				path = merklePath

				// modify the channel before marshalling to value bz
				channel.State = channeltypes.CLOSED
				value = suite.chain.Codec.MustMarshal(&channel)
			},
			errors.New("value provided does not equal value stored at path"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			lightClientModule, err := suite.chain.App.GetIBCKeeper().ClientKeeper.Route(suite.chain.GetContext(), exported.LocalhostClientID)
			suite.Require().NoError(err)

			err = lightClientModule.VerifyMembership(
				suite.chain.GetContext(),
				exported.LocalhostClientID,
				clienttypes.ZeroHeight(),
				0, 0, // use zero values for delay periods
				localhost.SentinelProof,
				path,
				value,
			)

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}

func (suite *LocalhostTestSuite) TestVerifyNonMembership() {
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
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				path = merklePath
			},
			nil,
		},
		{
			"packet receipt absence verification fails",
			func() {
				suite.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetPacketReceipt(suite.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, 1)

				merklePath := commitmenttypes.NewMerklePath(host.PacketReceiptKey(mock.PortID, ibctesting.FirstChannelID, 1))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

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
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			lightClientModule, err := suite.chain.App.GetIBCKeeper().ClientKeeper.Route(suite.chain.GetContext(), exported.LocalhostClientID)
			suite.Require().NoError(err)

			err = lightClientModule.VerifyNonMembership(
				suite.chain.GetContext(),
				exported.LocalhostClientID,
				clienttypes.ZeroHeight(),
				0, 0, // use zero values for delay periods
				localhost.SentinelProof,
				path,
			)

			if tc.expError == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.ErrorContains(err, tc.expError.Error())
			}
		})
	}
}

func (suite *LocalhostTestSuite) TestStatus() {
	lightClientModule, err := suite.chain.App.GetIBCKeeper().ClientKeeper.Route(suite.chain.GetContext(), exported.LocalhostClientID)
	suite.Require().NoError(err)

	suite.Require().Equal(exported.Active, lightClientModule.Status(suite.chain.GetContext(), exported.LocalhostClientID))
}

func (suite *LocalhostTestSuite) TestGetTimestampAtHeight() {
	lightClientModule, err := suite.chain.App.GetIBCKeeper().ClientKeeper.Route(suite.chain.GetContext(), exported.LocalhostClientID)
	suite.Require().NoError(err)

	ctx := suite.chain.GetContext()
	timestamp, err := lightClientModule.TimestampAtHeight(ctx, exported.LocalhostClientID, nil)
	suite.Require().NoError(err)
	suite.Require().Equal(uint64(ctx.BlockTime().UnixNano()), timestamp)
}

func (suite *LocalhostTestSuite) TestRecoverClient() {
	lightClientModule, err := suite.chain.App.GetIBCKeeper().ClientKeeper.Route(suite.chain.GetContext(), exported.LocalhostClientID)
	suite.Require().NoError(err)

	err = lightClientModule.RecoverClient(suite.chain.GetContext(), exported.LocalhostClientID, exported.LocalhostClientID)
	suite.Require().Error(err)
}

func (suite *LocalhostTestSuite) TestVerifyUpgradeAndUpdateState() {
	lightClientModule, err := suite.chain.App.GetIBCKeeper().ClientKeeper.Route(suite.chain.GetContext(), exported.LocalhostClientID)
	suite.Require().NoError(err)

	err = lightClientModule.VerifyUpgradeAndUpdateState(suite.chain.GetContext(), exported.LocalhostClientID, nil, nil, nil, nil)
	suite.Require().Error(err)
}
