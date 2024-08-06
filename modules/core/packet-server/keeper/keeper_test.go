package keeper_test

import (
	"fmt"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	tmtypes "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
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
