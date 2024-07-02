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

func (suite *LocalhostTestSuite) TestStatus() {
	lightClientModule, found := suite.chain.GetSimApp().IBCKeeper.ClientKeeper.Route(exported.LocalhostClientID)
	suite.Require().True(found)
	suite.Require().Equal(exported.Active, lightClientModule.Status(suite.chain.GetContext(), exported.LocalhostClientID))
}

func (suite *LocalhostTestSuite) TestVerifyMembership() {
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
				clientState := suite.chain.GetClientState(exported.LocalhostClientID)

				merklePath := commitmenttypes.NewMerklePath(host.FullClientStateKey(exported.LocalhostClientID))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				path = merklePath
				value = clienttypes.MustMarshalClientState(suite.chain.Codec, clientState)
			},
			true,
		},
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

				suite.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(suite.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, channel)

				merklePath := commitmenttypes.NewMerklePath(host.ChannelKey(mock.PortID, ibctesting.FirstChannelID))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				path = merklePath
				value = suite.chain.Codec.MustMarshal(&channel)
			},
			true,
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

				commitmentBz := channeltypes.CommitPacket(suite.chain.Codec, packet)
				suite.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetPacketCommitment(suite.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, 1, commitmentBz)

				merklePath := commitmenttypes.NewMerklePath(host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence()))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				path = merklePath
				value = commitmentBz
			},
			true,
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
				path = commitmenttypes.NewMerklePath([]byte("ibc"), []byte("test"), []byte("key"))
			},
			false,
		},
		{
			"no value found at provided key path",
			func() {
				merklePath := commitmenttypes.NewMerklePath(host.PacketAcknowledgementKey(mock.PortID, ibctesting.FirstChannelID, 100))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

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

				suite.chain.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(suite.chain.GetContext(), mock.PortID, ibctesting.FirstChannelID, channel)

				merklePath := commitmenttypes.NewMerklePath(host.ChannelKey(mock.PortID, ibctesting.FirstChannelID))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				path = merklePath

				// modify the channel before marshalling to value bz
				channel.State = channeltypes.CLOSED
				value = suite.chain.Codec.MustMarshal(&channel)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			lightClientModule, found := suite.chain.GetSimApp().IBCKeeper.ClientKeeper.Route(exported.LocalhostClientID)
			suite.Require().True(found)

			err := lightClientModule.VerifyMembership(
				suite.chain.GetContext(),
				exported.LocalhostClientID,
				clienttypes.ZeroHeight(),
				0, 0, // use zero values for delay periods
				localhost.SentinelProof,
				path,
				value,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *LocalhostTestSuite) TestVerifyNonMembership() {
	var path exported.Path

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: packet receipt absence verification",
			func() {
				merklePath := commitmenttypes.NewMerklePath(host.PacketReceiptKey(mock.PortID, ibctesting.FirstChannelID, 1))
				merklePath, err := commitmenttypes.ApplyPrefix(suite.chain.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				path = merklePath
			},
			true,
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
				path = commitmenttypes.NewMerklePath([]byte("ibc"), []byte("test"), []byte("key"))
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			lightClientModule, found := suite.chain.GetSimApp().IBCKeeper.ClientKeeper.Route(exported.LocalhostClientID)
			suite.Require().True(found)

			err := lightClientModule.VerifyNonMembership(
				suite.chain.GetContext(),
				exported.LocalhostClientID,
				clienttypes.ZeroHeight(),
				0, 0, // use zero values for delay periods
				localhost.SentinelProof,
				path,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *LocalhostTestSuite) TestRecoverClient() {
	lightClientModule, found := suite.chain.GetSimApp().IBCKeeper.ClientKeeper.Route(exported.LocalhostClientID)
	suite.Require().True(found)

	err := lightClientModule.RecoverClient(suite.chain.GetContext(), exported.LocalhostClientID, exported.LocalhostClientID)
	suite.Require().Error(err)
}

func (suite *LocalhostTestSuite) TestVerifyUpgradeAndUpdateState() {
	lightClientModule, found := suite.chain.GetSimApp().IBCKeeper.ClientKeeper.Route(exported.LocalhostClientID)
	suite.Require().True(found)

	err := lightClientModule.VerifyUpgradeAndUpdateState(suite.chain.GetContext(), exported.LocalhostClientID, nil, nil, nil, nil)
	suite.Require().Error(err)
}
