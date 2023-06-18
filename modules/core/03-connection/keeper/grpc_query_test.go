package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestQueryConnection() {
	var (
		req           *types.QueryConnectionRequest
		expConnection types.ConnectionEnd
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid connectionID",
			func() {
				req = &types.QueryConnectionRequest{}
			},
			false,
		},
		{
			"connection not found",
			func() {
				req = &types.QueryConnectionRequest{
					ConnectionId: ibctesting.InvalidID,
				}
			},
			false,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(path)
				err := path.EndpointA.ConnOpenInit()
				s.Require().NoError(err)

				counterparty := types.NewCounterparty(path.EndpointB.ClientID, "", s.chainB.GetPrefix())
				expConnection = types.NewConnectionEnd(types.INIT, path.EndpointA.ClientID, counterparty, types.ExportedVersionsToProto(types.GetCompatibleVersions()), 500)
				s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(s.chainA.GetContext(), path.EndpointA.ConnectionID, expConnection)

				req = &types.QueryConnectionRequest{
					ConnectionId: path.EndpointA.ConnectionID,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.Connection(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(&expConnection, res.Connection)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryConnections() {
	s.chainA.App.GetIBCKeeper().ConnectionKeeper.CreateSentinelLocalhostConnection(s.chainA.GetContext())
	localhostConn, found := s.chainA.App.GetIBCKeeper().ConnectionKeeper.GetConnection(s.chainA.GetContext(), exported.LocalhostConnectionID)
	s.Require().True(found)

	identifiedConn := types.NewIdentifiedConnection(exported.LocalhostConnectionID, localhostConn)

	var (
		req            *types.QueryConnectionsRequest
		expConnections = []*types.IdentifiedConnection{&identifiedConn}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"empty pagination",
			func() {
				req = &types.QueryConnectionsRequest{}
			},
			true,
		},
		{
			"success",
			func() {
				path1 := ibctesting.NewPath(s.chainA, s.chainB)
				path2 := ibctesting.NewPath(s.chainA, s.chainB)
				path3 := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupConnections(path1)
				s.coordinator.SetupConnections(path2)
				s.coordinator.SetupClients(path3)

				err := path3.EndpointA.ConnOpenInit()
				s.Require().NoError(err)

				counterparty1 := types.NewCounterparty(path1.EndpointB.ClientID, path1.EndpointB.ConnectionID, s.chainB.GetPrefix())
				counterparty2 := types.NewCounterparty(path2.EndpointB.ClientID, path2.EndpointB.ConnectionID, s.chainB.GetPrefix())
				// counterparty connection id is blank after open init
				counterparty3 := types.NewCounterparty(path3.EndpointB.ClientID, "", s.chainB.GetPrefix())

				conn1 := types.NewConnectionEnd(types.OPEN, path1.EndpointA.ClientID, counterparty1, types.ExportedVersionsToProto(types.GetCompatibleVersions()), 0)
				conn2 := types.NewConnectionEnd(types.OPEN, path2.EndpointA.ClientID, counterparty2, types.ExportedVersionsToProto(types.GetCompatibleVersions()), 0)
				conn3 := types.NewConnectionEnd(types.INIT, path3.EndpointA.ClientID, counterparty3, types.ExportedVersionsToProto(types.GetCompatibleVersions()), 0)

				iconn1 := types.NewIdentifiedConnection(path1.EndpointA.ConnectionID, conn1)
				iconn2 := types.NewIdentifiedConnection(path2.EndpointA.ConnectionID, conn2)
				iconn3 := types.NewIdentifiedConnection(path3.EndpointA.ConnectionID, conn3)

				expConnections = []*types.IdentifiedConnection{&iconn1, &iconn2, &iconn3, &identifiedConn}

				req = &types.QueryConnectionsRequest{
					Pagination: &query.PageRequest{
						Limit:      4,
						CountTotal: true,
					},
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.Connections(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expConnections, res.Connections)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryClientConnections() {
	var (
		req      *types.QueryClientConnectionsRequest
		expPaths []string
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid connectionID",
			func() {
				req = &types.QueryClientConnectionsRequest{}
			},
			false,
		},
		{
			"connection not found",
			func() {
				req = &types.QueryClientConnectionsRequest{
					ClientId: ibctesting.InvalidID,
				}
			},
			false,
		},
		{
			"success",
			func() {
				path1 := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupConnections(path1)

				// create another connection using same underlying clients
				path2 := ibctesting.NewPath(s.chainA, s.chainB)
				path2.EndpointA.ClientID = path1.EndpointA.ClientID
				path2.EndpointB.ClientID = path1.EndpointB.ClientID

				s.coordinator.CreateConnections(path2)

				expPaths = []string{path1.EndpointA.ConnectionID, path2.EndpointA.ConnectionID}
				s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetClientConnectionPaths(s.chainA.GetContext(), path1.EndpointA.ClientID, expPaths)

				req = &types.QueryClientConnectionsRequest{
					ClientId: path1.EndpointA.ClientID,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.ClientConnections(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expPaths, res.ConnectionPaths)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryConnectionClientState() {
	var (
		req                      *types.QueryConnectionClientStateRequest
		expIdentifiedClientState clienttypes.IdentifiedClientState
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid connection ID",
			func() {
				req = &types.QueryConnectionClientStateRequest{
					ConnectionId: "",
				}
			},
			false,
		},
		{
			"connection not found",
			func() {
				req = &types.QueryConnectionClientStateRequest{
					ConnectionId: "test-connection-id",
				}
			},
			false,
		},
		{
			"client state not found",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				// set connection to empty so clientID is empty
				s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(s.chainA.GetContext(), path.EndpointA.ConnectionID, types.ConnectionEnd{})

				req = &types.QueryConnectionClientStateRequest{
					ConnectionId: path.EndpointA.ConnectionID,
				}
			}, false,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupConnections(path)

				expClientState := s.chainA.GetClientState(path.EndpointA.ClientID)
				expIdentifiedClientState = clienttypes.NewIdentifiedClientState(path.EndpointA.ClientID, expClientState)

				req = &types.QueryConnectionClientStateRequest{
					ConnectionId: path.EndpointA.ConnectionID,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.ConnectionClientState(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(&expIdentifiedClientState, res.IdentifiedClientState)

				// ensure UnpackInterfaces is defined
				cachedValue := res.IdentifiedClientState.ClientState.GetCachedValue()
				s.Require().NotNil(cachedValue)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryConnectionConsensusState() {
	var (
		req               *types.QueryConnectionConsensusStateRequest
		expConsensusState exported.ConsensusState
		expClientID       string
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid connection ID",
			func() {
				req = &types.QueryConnectionConsensusStateRequest{
					ConnectionId:   "",
					RevisionNumber: 0,
					RevisionHeight: 1,
				}
			},
			false,
		},
		{
			"connection not found",
			func() {
				req = &types.QueryConnectionConsensusStateRequest{
					ConnectionId:   "test-connection-id",
					RevisionNumber: 0,
					RevisionHeight: 1,
				}
			},
			false,
		},
		{
			"consensus state not found",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.Setup(path)

				req = &types.QueryConnectionConsensusStateRequest{
					ConnectionId:   path.EndpointA.ConnectionID,
					RevisionNumber: 0,
					RevisionHeight: uint64(s.chainA.GetContext().BlockHeight()), // use current height
				}
			}, false,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupConnections(path)

				clientState := s.chainA.GetClientState(path.EndpointA.ClientID)
				expConsensusState, _ = s.chainA.GetConsensusState(path.EndpointA.ClientID, clientState.GetLatestHeight())
				s.Require().NotNil(expConsensusState)
				expClientID = path.EndpointA.ClientID

				req = &types.QueryConnectionConsensusStateRequest{
					ConnectionId:   path.EndpointA.ConnectionID,
					RevisionNumber: clientState.GetLatestHeight().GetRevisionNumber(),
					RevisionHeight: clientState.GetLatestHeight().GetRevisionHeight(),
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.QueryServer.ConnectionConsensusState(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				consensusState, err := clienttypes.UnpackConsensusState(res.ConsensusState)
				s.Require().NoError(err)
				s.Require().Equal(expConsensusState, consensusState)
				s.Require().Equal(expClientID, res.ClientId)

				// ensure UnpackInterfaces is defined
				cachedValue := res.ConsensusState.GetCachedValue()
				s.Require().NotNil(cachedValue)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryConnectionParams() {
	ctx := sdk.WrapSDKContext(s.chainA.GetContext())
	expParams := types.DefaultParams()
	res, _ := s.chainA.QueryServer.ConnectionParams(ctx, &types.QueryConnectionParamsRequest{})
	s.Require().Equal(&expParams, res.Params)
}
