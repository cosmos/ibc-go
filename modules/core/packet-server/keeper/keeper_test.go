package keeper_test

import (
	"fmt"
	"testing"

	"testing"

	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"

	testifysuite "github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v9/testing"
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

	tests := []struct {
		name     string
		malleate func()
		expErr   err
	}{
		{"success", func() {
			// set the counterparties
			path.SetupCounterparties()
		}, nil},
		{"counterparty not found", func() {}, channeltypes.ErrChannelNotFound},
		{"packet failed basic validation", func() {
			// set the counterparties
			path.SetupCounterparties()
			// invalid port ID
			packet.DestPort = ""
		}, channeltypes.ErrInvalidPacket},
		{"client status invalid", func() {
			// set the counterparties
			path.SetupCounterparties()
			// change source channel id to get invalid status
			packet.SourceChannel = "invalidClientID"
		}, clienttypes.ErrClientNotActive},
		{"timeout elapsed", func() {
			// set the counterparties
			path.SetupCounterparties()
			packet.TimeoutTimestamp = 1
		}, channeltypes.ErrTimeoutElapsed},
	}

	for i, tc := range tests {
		tc := tc
		suite.Run(fmt.Sprintf("Case %s, %d/%d tests", tc.msg, i, len(testCases)), func() {
			suite.SetupTest() // reset

			// create clients on both chains
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			// create standard packet that can be malleated
			packet := channeltypes.NewPacketWithVersion(mock.MockPacketData, 1, mock.PortID,
				path.EndpointA.ClientID, mock.PortID, path.EndpointB.ClientID, clienttypes.NewHeight(0, 1), 0, mock.Version)

			// malleate the test case
			tc.malleate()

			// send packet
			seq, err := suite.chainA.App.PacketServer.SendPacket(suite.chainA.GetContext(), nil, packet.SourceChannel, packet.SourcePort,
				packet.DestPort, packet.TimeoutHeight, packet.TimeoutTimestamp, packet.Version, packet.Data)

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
