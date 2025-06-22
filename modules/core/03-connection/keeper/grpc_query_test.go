package keeper_test

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/types/query"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestQueryConnection() {
	var (
		req           *types.QueryConnectionRequest
		expConnection types.ConnectionEnd
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid connectionID",
			func() {
				req = &types.QueryConnectionRequest{}
			},
			status.Error(codes.InvalidArgument, errorsmod.Wrap(host.ErrInvalidID, "identifier cannot be blank").Error()),
		},
		{
			"connection not found",
			func() {
				req = &types.QueryConnectionRequest{
					ConnectionId: ibctesting.InvalidID,
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrap(types.ErrConnectionNotFound, "IDisInvalid").Error(),
			),
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupClients()
				err := path.EndpointA.ConnOpenInit()
				s.Require().NoError(err)

				counterparty := types.NewCounterparty(path.EndpointB.ClientID, "", s.chainB.GetPrefix())
				expConnection = types.NewConnectionEnd(types.INIT, path.EndpointA.ClientID, counterparty, types.GetCompatibleVersions(), 500)
				s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(s.chainA.GetContext(), path.EndpointA.ConnectionID, expConnection)

				req = &types.QueryConnectionRequest{
					ConnectionId: path.EndpointA.ConnectionID,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()

			queryServer := keeper.NewQueryServer(s.chainA.App.GetIBCKeeper().ConnectionKeeper)
			res, err := queryServer.Connection(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(&expConnection, res.Connection)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
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
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"empty pagination",
			func() {
				req = &types.QueryConnectionsRequest{}
			},
			nil,
		},
		{
			"success",
			func() {
				path1 := ibctesting.NewPath(s.chainA, s.chainB)
				path2 := ibctesting.NewPath(s.chainA, s.chainB)
				path3 := ibctesting.NewPath(s.chainA, s.chainB)
				path1.SetupConnections()
				path2.SetupConnections()
				path3.SetupClients()

				err := path3.EndpointA.ConnOpenInit()
				s.Require().NoError(err)

				counterparty1 := types.NewCounterparty(path1.EndpointB.ClientID, path1.EndpointB.ConnectionID, s.chainB.GetPrefix())
				counterparty2 := types.NewCounterparty(path2.EndpointB.ClientID, path2.EndpointB.ConnectionID, s.chainB.GetPrefix())
				// counterparty connection id is blank after open init
				counterparty3 := types.NewCounterparty(path3.EndpointB.ClientID, "", s.chainB.GetPrefix())

				conn1 := types.NewConnectionEnd(types.OPEN, path1.EndpointA.ClientID, counterparty1, types.GetCompatibleVersions(), 0)
				conn2 := types.NewConnectionEnd(types.OPEN, path2.EndpointA.ClientID, counterparty2, types.GetCompatibleVersions(), 0)
				conn3 := types.NewConnectionEnd(types.INIT, path3.EndpointA.ClientID, counterparty3, types.GetCompatibleVersions(), 0)

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
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()

			queryServer := keeper.NewQueryServer(s.chainA.App.GetIBCKeeper().ConnectionKeeper)
			res, err := queryServer.Connections(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expConnections, res.Connections)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
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
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid connectionID",
			func() {
				req = &types.QueryClientConnectionsRequest{}
			},
			status.Error(codes.InvalidArgument, errorsmod.Wrap(host.ErrInvalidID, "identifier cannot be blank").Error()),
		},
		{
			"connection not found",
			func() {
				req = &types.QueryClientConnectionsRequest{
					ClientId: ibctesting.InvalidID,
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrap(types.ErrClientConnectionPathsNotFound, "IDisInvalid").Error(),
			),
		},
		{
			"success",
			func() {
				path1 := ibctesting.NewPath(s.chainA, s.chainB)
				path1.SetupConnections()

				// create another connection using same underlying clients
				path2 := ibctesting.NewPath(s.chainA, s.chainB)
				path2.EndpointA.ClientID = path1.EndpointA.ClientID
				path2.EndpointB.ClientID = path1.EndpointB.ClientID

				path2.CreateConnections()

				expPaths = []string{path1.EndpointA.ConnectionID, path2.EndpointA.ConnectionID}
				s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetClientConnectionPaths(s.chainA.GetContext(), path1.EndpointA.ClientID, expPaths)

				req = &types.QueryClientConnectionsRequest{
					ClientId: path1.EndpointA.ClientID,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()

			queryServer := keeper.NewQueryServer(s.chainA.App.GetIBCKeeper().ConnectionKeeper)
			res, err := queryServer.ClientConnections(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expPaths, res.ConnectionPaths)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
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
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid connection ID",
			func() {
				req = &types.QueryConnectionClientStateRequest{
					ConnectionId: "",
				}
			},
			status.Error(codes.InvalidArgument, errorsmod.Wrap(host.ErrInvalidID, "identifier cannot be blank").Error()),
		},
		{
			"connection not found",
			func() {
				req = &types.QueryConnectionClientStateRequest{
					ConnectionId: "test-connection-id",
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrap(types.ErrConnectionNotFound, "connection-id: test-connection-id").Error(),
			),
		},
		{
			"client state not found",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.Setup()

				// set connection to empty so clientID is empty
				s.chainA.App.GetIBCKeeper().ConnectionKeeper.SetConnection(s.chainA.GetContext(), path.EndpointA.ConnectionID, types.ConnectionEnd{})

				req = &types.QueryConnectionClientStateRequest{
					ConnectionId: path.EndpointA.ConnectionID,
				}
			}, status.Error(
				codes.NotFound,
				errorsmod.Wrap(clienttypes.ErrClientNotFound, "client-id: ").Error(),
			),
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupConnections()

				expClientState := s.chainA.GetClientState(path.EndpointA.ClientID)
				expIdentifiedClientState = clienttypes.NewIdentifiedClientState(path.EndpointA.ClientID, expClientState)

				req = &types.QueryConnectionClientStateRequest{
					ConnectionId: path.EndpointA.ConnectionID,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()

			queryServer := keeper.NewQueryServer(s.chainA.App.GetIBCKeeper().ConnectionKeeper)
			res, err := queryServer.ConnectionClientState(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(&expIdentifiedClientState, res.IdentifiedClientState)

				// ensure UnpackInterfaces is defined
				cachedValue := res.IdentifiedClientState.ClientState.GetCachedValue()
				s.Require().NotNil(cachedValue)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
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
		expErr   error
	}{
		{
			"empty request",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
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
			status.Error(codes.InvalidArgument, errorsmod.Wrap(host.ErrInvalidID, "identifier cannot be blank").Error()),
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
			status.Error(
				codes.NotFound,
				errorsmod.Wrap(types.ErrConnectionNotFound, "connection-id: test-connection-id").Error(),
			),
		},
		{
			"consensus state not found",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.Setup()

				req = &types.QueryConnectionConsensusStateRequest{
					ConnectionId:   path.EndpointA.ConnectionID,
					RevisionNumber: 0,
					RevisionHeight: uint64(s.chainA.GetContext().BlockHeight()), // use current height
				}
			}, status.Error(
				codes.NotFound,
				errorsmod.Wrap(clienttypes.ErrConsensusStateNotFound, "client-id: 07-tendermint-0").Error(),
			),
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupConnections()

				clientHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
				s.Require().True(ok)
				expConsensusState, _ = s.chainA.GetConsensusState(path.EndpointA.ClientID, clientHeight)
				s.Require().NotNil(expConsensusState)
				expClientID = path.EndpointA.ClientID

				req = &types.QueryConnectionConsensusStateRequest{
					ConnectionId:   path.EndpointA.ConnectionID,
					RevisionNumber: clientHeight.GetRevisionNumber(),
					RevisionHeight: clientHeight.GetRevisionHeight(),
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()

			queryServer := keeper.NewQueryServer(s.chainA.App.GetIBCKeeper().ConnectionKeeper)
			res, err := queryServer.ConnectionConsensusState(ctx, req)

			if tc.expErr == nil {
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
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryConnectionParams() {
	expParams := types.DefaultParams()

	queryServer := keeper.NewQueryServer(s.chainA.App.GetIBCKeeper().ConnectionKeeper)
	res, err := queryServer.ConnectionParams(s.chainA.GetContext(), &types.QueryConnectionParamsRequest{})
	s.Require().NoError(err)
	s.Require().Equal(&expParams, res.Params)
}
