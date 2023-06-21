package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestSetAndGetConnection() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.SetupClients(path)
	firstConnection := "connection-0"

	// check first connection does not exist
	_, existed := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetConnection(s.chainA.GetContext(), firstConnection)
	s.Require().False(existed)

	s.coordinator.CreateConnections(path)
	_, existed = s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetConnection(s.chainA.GetContext(), firstConnection)
	s.Require().True(existed)
}

func (s *KeeperTestSuite) TestSetAndGetClientConnectionPaths() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.SetupClients(path)

	_, existed := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetClientConnectionPaths(s.chainA.GetContext(), path.EndpointA.ClientID)
	s.False(existed)

	connections := []string{"connectionA", "connectionB"}
	s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetClientConnectionPaths(s.chainA.GetContext(), path.EndpointA.ClientID, connections)
	paths, existed := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetClientConnectionPaths(s.chainA.GetContext(), path.EndpointA.ClientID)
	s.True(existed)
	s.EqualValues(connections, paths)
}

// create 2 connections: A0 - B0, A1 - B1
func (s KeeperTestSuite) TestGetAllConnections() { //nolint:govet // this is a test, we are okay with copying locks
	path1 := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path1)

	path2 := ibctesting.NewPath(s.chainA, s.chainB)
	path2.EndpointA.ClientID = path1.EndpointA.ClientID
	path2.EndpointB.ClientID = path1.EndpointB.ClientID

	s.coordinator.CreateConnections(path2)

	counterpartyB0 := types.NewCounterparty(path1.EndpointB.ClientID, path1.EndpointB.ConnectionID, s.chainB.GetPrefix()) // connection B0
	counterpartyB1 := types.NewCounterparty(path2.EndpointB.ClientID, path2.EndpointB.ConnectionID, s.chainB.GetPrefix()) // connection B1

	conn1 := types.NewConnectionEnd(types.OPEN, path1.EndpointA.ClientID, counterpartyB0, types.ExportedVersionsToProto(types.GetCompatibleVersions()), 0) // A0 - B0
	conn2 := types.NewConnectionEnd(types.OPEN, path2.EndpointA.ClientID, counterpartyB1, types.ExportedVersionsToProto(types.GetCompatibleVersions()), 0) // A1 - B1

	iconn1 := types.NewIdentifiedConnection(path1.EndpointA.ConnectionID, conn1)
	iconn2 := types.NewIdentifiedConnection(path2.EndpointA.ConnectionID, conn2)

	s.chainA.App.GetIBCKeeper().ConnectionKeeper.CreateSentinelLocalhostConnection(s.chainA.GetContext())
	localhostConn, found := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetConnection(s.chainA.GetContext(), exported.LocalhostConnectionID)
	s.Require().True(found)

	expConnections := []types.IdentifiedConnection{iconn1, iconn2, types.NewIdentifiedConnection(exported.LocalhostConnectionID, localhostConn)}

	connections := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetAllConnections(s.chainA.GetContext())
	s.Require().Len(connections, len(expConnections))
	s.Require().Equal(expConnections, connections)
}

// the test creates 2 clients path.EndpointA.ClientID0 and path.EndpointA.ClientID1. path.EndpointA.ClientID0 has a single
// connection and path.EndpointA.ClientID1 has 2 connections.
func (s KeeperTestSuite) TestGetAllClientConnectionPaths() { //nolint:govet // this is a test, we are okay with copying locks
	path1 := ibctesting.NewPath(s.chainA, s.chainB)
	path2 := ibctesting.NewPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path1)
	s.coordinator.SetupConnections(path2)

	path3 := ibctesting.NewPath(s.chainA, s.chainB)
	path3.EndpointA.ClientID = path2.EndpointA.ClientID
	path3.EndpointB.ClientID = path2.EndpointB.ClientID
	s.coordinator.CreateConnections(path3)

	expPaths := []types.ConnectionPaths{
		types.NewConnectionPaths(path1.EndpointA.ClientID, []string{path1.EndpointA.ConnectionID}),
		types.NewConnectionPaths(path2.EndpointA.ClientID, []string{path2.EndpointA.ConnectionID, path3.EndpointA.ConnectionID}),
	}

	connPaths := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetAllClientConnectionPaths(s.chainA.GetContext())
	s.Require().Len(connPaths, 2)
	s.Require().Equal(expPaths, connPaths)
}

// TestGetTimestampAtHeight verifies if the clients on each chain return the
// correct timestamp for the other chain.
func (s *KeeperTestSuite) TestGetTimestampAtHeight() {
	var (
		connection types.ConnectionEnd
		height     exported.Height
	)

	cases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{"verification success", func() {
			path := ibctesting.NewPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)
			connection = path.EndpointA.GetConnection()
			height = s.chainB.LastHeader.GetHeight()
		}, true},
		{"client state not found", func() {}, false},
		{"consensus state not found", func() {
			path := ibctesting.NewPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)
			connection = path.EndpointA.GetConnection()
			height = s.chainB.LastHeader.GetHeight().Increment()
		}, false},
	}

	for _, tc := range cases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()

			actualTimestamp, err := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetTimestampAtHeight(
				s.chainA.GetContext(), connection, height,
			)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().EqualValues(uint64(s.chainB.LastHeader.GetTime().UnixNano()), actualTimestamp)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestLocalhostConnectionEndCreation() {
	ctx := s.chainA.GetContext()
	connectionKeeper := s.chainA.App.GetIBCKeeper().ConnectionKeeper
	connectionKeeper.CreateSentinelLocalhostConnection(ctx)

	connectionEnd, found := connectionKeeper.GetConnection(ctx, exported.LocalhostConnectionID)

	s.Require().True(found)
	s.Require().Equal(types.OPEN, connectionEnd.State)
	s.Require().Equal(exported.LocalhostClientID, connectionEnd.ClientId)
	s.Require().Equal(types.ExportedVersionsToProto(types.GetCompatibleVersions()), connectionEnd.Versions)
	expectedCounterParty := types.NewCounterparty(exported.LocalhostClientID, exported.LocalhostConnectionID, commitmenttypes.NewMerklePrefix(connectionKeeper.GetCommitmentPrefix().Bytes()))
	s.Require().Equal(expectedCounterParty, connectionEnd.Counterparty)
}

// TestDefaultSetParams tests the default params set are what is expected
func (s *KeeperTestSuite) TestDefaultSetParams() {
	expParams := types.DefaultParams()

	params := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetParams(s.chainA.GetContext())
	s.Require().Equal(expParams, params)
}

// TestParams tests that param setting and retrieval works properly
func (s *KeeperTestSuite) TestSetAndGetParams() {
	testCases := []struct {
		name    string
		input   types.Params
		expPass bool
	}{
		{"success: set default params", types.DefaultParams(), true},
		{"success: valid value for MaxExpectedTimePerBlock", types.NewParams(10), true},
		{"failure: invalid value for MaxExpectedTimePerBlock", types.NewParams(0), false},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset
			ctx := s.chainA.GetContext()
			err := tc.input.Validate()
			s.chainA.GetSimApp().IBCKeeper.ConnectionKeeper.SetParams(ctx, tc.input)
			if tc.expPass {
				s.Require().NoError(err)
				expected := tc.input
				p := s.chainA.GetSimApp().IBCKeeper.ConnectionKeeper.GetParams(ctx)
				s.Require().Equal(expected, p)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// TestUnsetParams tests that trying to get params that are not set panics.
func (s *KeeperTestSuite) TestUnsetParams() {
	s.SetupTest()
	ctx := s.chainA.GetContext()
	store := ctx.KVStore(s.chainA.GetSimApp().GetKey(exported.StoreKey))
	store.Delete([]byte(types.ParamsKey))

	s.Require().Panics(func() {
		s.chainA.GetSimApp().IBCKeeper.ConnectionKeeper.GetParams(ctx)
	})
}
