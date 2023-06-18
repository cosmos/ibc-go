package keeper_test

import (
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestQueryClientState() {
	var (
		req            *types.QueryClientStateRequest
		expClientState *codectypes.Any
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"req is nil",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid clientID",
			func() {
				req = &types.QueryClientStateRequest{}
			},
			false,
		},
		{
			"client not found",
			func() {
				req = &types.QueryClientStateRequest{
					ClientId: testClientID,
				}
			},
			false,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(path)

				var err error
				expClientState, err = types.PackClientState(path.EndpointA.GetClientState())
				s.Require().NoError(err)

				req = &types.QueryClientStateRequest{
					ClientId: path.EndpointA.ClientID,
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
			res, err := s.chainA.QueryServer.ClientState(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expClientState, res.ClientState)

				// ensure UnpackInterfaces is defined
				cachedValue := res.ClientState.GetCachedValue()
				s.Require().NotNil(cachedValue)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryClientStates() {
	var (
		req             *types.QueryClientStatesRequest
		expClientStates = types.IdentifiedClientStates{}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"req is nil",
			func() {
				req = nil
			},
			false,
		},
		{
			"empty pagination",
			func() {
				localhost := types.NewIdentifiedClientState(exported.LocalhostClientID, s.chainA.GetClientState(exported.LocalhostClientID))
				expClientStates = types.IdentifiedClientStates{localhost}
				req = &types.QueryClientStatesRequest{}
			},
			true,
		},
		{
			"success, only localhost",
			func() {
				localhost := types.NewIdentifiedClientState(exported.LocalhostClientID, s.chainA.GetClientState(exported.LocalhostClientID))
				expClientStates = types.IdentifiedClientStates{localhost}
				req = &types.QueryClientStatesRequest{
					Pagination: &query.PageRequest{
						Limit:      3,
						CountTotal: true,
					},
				}
			},
			true,
		},
		{
			"success",
			func() {
				path1 := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(path1)

				path2 := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(path2)

				clientStateA1 := path1.EndpointA.GetClientState()
				clientStateA2 := path2.EndpointA.GetClientState()

				localhost := types.NewIdentifiedClientState(exported.LocalhostClientID, s.chainA.GetClientState(exported.LocalhostClientID))
				idcs := types.NewIdentifiedClientState(path1.EndpointA.ClientID, clientStateA1)
				idcs2 := types.NewIdentifiedClientState(path2.EndpointA.ClientID, clientStateA2)

				// order is sorted by client id
				expClientStates = types.IdentifiedClientStates{localhost, idcs, idcs2}.Sort()
				req = &types.QueryClientStatesRequest{
					Pagination: &query.PageRequest{
						Limit:      20,
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
			res, err := s.chainA.QueryServer.ClientStates(ctx, req)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expClientStates.Sort(), res.ClientStates)
				s.Require().Equal(len(expClientStates), int(res.Pagination.Total))
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryConsensusState() {
	var (
		req               *types.QueryConsensusStateRequest
		expConsensusState *codectypes.Any
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"req is nil",
			func() {
				req = nil
			},
			false,
		},
		{
			"invalid clientID",
			func() {
				req = &types.QueryConsensusStateRequest{}
			},
			false,
		},
		{
			"invalid height",
			func() {
				req = &types.QueryConsensusStateRequest{
					ClientId:       testClientID,
					RevisionNumber: 0,
					RevisionHeight: 0,
					LatestHeight:   false,
				}
			},
			false,
		},
		{
			"consensus state not found",
			func() {
				req = &types.QueryConsensusStateRequest{
					ClientId:     ibctesting.FirstClientID,
					LatestHeight: true,
				}
			},
			false,
		},
		{
			"success latest height",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(path)
				cs := path.EndpointA.GetConsensusState(path.EndpointA.GetClientState().GetLatestHeight())

				var err error
				expConsensusState, err = types.PackConsensusState(cs)
				s.Require().NoError(err)

				req = &types.QueryConsensusStateRequest{
					ClientId:     path.EndpointA.ClientID,
					LatestHeight: true,
				}
			},
			true,
		},
		{
			"success with height",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(path)
				height := path.EndpointA.GetClientState().GetLatestHeight()
				cs := path.EndpointA.GetConsensusState(height)

				var err error
				expConsensusState, err = types.PackConsensusState(cs)
				s.Require().NoError(err)

				// update client to new height
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				req = &types.QueryConsensusStateRequest{
					ClientId:       path.EndpointA.ClientID,
					RevisionNumber: height.GetRevisionNumber(),
					RevisionHeight: height.GetRevisionHeight(),
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
			res, err := s.chainA.QueryServer.ConsensusState(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expConsensusState, res.ConsensusState)

				// ensure UnpackInterfaces is defined
				cachedValue := res.ConsensusState.GetCachedValue()
				s.Require().NotNil(cachedValue)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryConsensusStates() {
	var (
		req                *types.QueryConsensusStatesRequest
		expConsensusStates []types.ConsensusStateWithHeight
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"success: without pagination",
			func() {
				req = &types.QueryConsensusStatesRequest{
					ClientId: testClientID,
				}
			},
			true,
		},
		{
			"success, no results",
			func() {
				req = &types.QueryConsensusStatesRequest{
					ClientId: testClientID,
					Pagination: &query.PageRequest{
						Limit:      3,
						CountTotal: true,
					},
				}
			},
			true,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(path)

				height1 := path.EndpointA.GetClientState().GetLatestHeight().(types.Height)
				expConsensusStates = append(
					expConsensusStates,
					types.NewConsensusStateWithHeight(
						height1,
						path.EndpointA.GetConsensusState(height1),
					))

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				height2 := path.EndpointA.GetClientState().GetLatestHeight().(types.Height)
				expConsensusStates = append(
					expConsensusStates,
					types.NewConsensusStateWithHeight(
						height2,
						path.EndpointA.GetConsensusState(height2),
					))

				req = &types.QueryConsensusStatesRequest{
					ClientId: path.EndpointA.ClientID,
					Pagination: &query.PageRequest{
						Limit:      3,
						CountTotal: true,
					},
				}
			},
			true,
		},
		{
			"invalid client identifier",
			func() {
				req = &types.QueryConsensusStatesRequest{}
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.QueryServer.ConsensusStates(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(len(expConsensusStates), len(res.ConsensusStates))
				for i := range expConsensusStates {
					s.Require().NotNil(res.ConsensusStates[i])
					s.Require().Equal(expConsensusStates[i], res.ConsensusStates[i])
					// ensure UnpackInterfaces is defined
					cachedValue := res.ConsensusStates[i].ConsensusState.GetCachedValue()
					s.Require().NotNil(cachedValue)
				}
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryConsensusStateHeights() {
	var (
		req                      *types.QueryConsensusStateHeightsRequest
		expConsensusStateHeights []types.Height
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"success: without pagination",
			func() {
				req = &types.QueryConsensusStateHeightsRequest{
					ClientId: testClientID,
				}
			},
			true,
		},
		{
			"success: response contains no results",
			func() {
				req = &types.QueryConsensusStateHeightsRequest{
					ClientId: testClientID,
					Pagination: &query.PageRequest{
						Limit:      3,
						CountTotal: true,
					},
				}
			},
			true,
		},
		{
			"success: returns consensus heights",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(path)

				expConsensusStateHeights = append(expConsensusStateHeights, path.EndpointA.GetClientState().GetLatestHeight().(types.Height))

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				expConsensusStateHeights = append(expConsensusStateHeights, path.EndpointA.GetClientState().GetLatestHeight().(types.Height))

				req = &types.QueryConsensusStateHeightsRequest{
					ClientId: path.EndpointA.ClientID,
					Pagination: &query.PageRequest{
						Limit:      3,
						CountTotal: true,
					},
				}
			},
			true,
		},
		{
			"invalid client identifier",
			func() {
				req = &types.QueryConsensusStateHeightsRequest{}
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.QueryServer.ConsensusStateHeights(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(len(expConsensusStateHeights), len(res.ConsensusStateHeights))
				for i := range expConsensusStateHeights {
					s.Require().NotNil(res.ConsensusStateHeights[i])
					s.Require().Equal(expConsensusStateHeights[i], res.ConsensusStateHeights[i])
				}
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryClientStatus() {
	var req *types.QueryClientStatusRequest

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expStatus string
	}{
		{
			"req is nil",
			func() {
				req = nil
			},
			false, "",
		},
		{
			"invalid clientID",
			func() {
				req = &types.QueryClientStatusRequest{}
			},
			false, "",
		},
		{
			"client not found",
			func() {
				req = &types.QueryClientStatusRequest{
					ClientId: ibctesting.InvalidID,
				}
			},
			false, "",
		},
		{
			"Active client status",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(path)
				req = &types.QueryClientStatusRequest{
					ClientId: path.EndpointA.ClientID,
				}
			},
			true, exported.Active.String(),
		},
		{
			"Unknown client status",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(path)
				clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)

				// increment latest height so no consensus state is stored
				clientState.LatestHeight = clientState.LatestHeight.Increment().(types.Height)
				path.EndpointA.SetClientState(clientState)

				req = &types.QueryClientStatusRequest{
					ClientId: path.EndpointA.ClientID,
				}
			},
			true, exported.Expired.String(),
		},
		{
			"Frozen client status",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				s.coordinator.SetupClients(path)
				clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)

				clientState.FrozenHeight = types.NewHeight(0, 1)
				path.EndpointA.SetClientState(clientState)

				req = &types.QueryClientStatusRequest{
					ClientId: path.EndpointA.ClientID,
				}
			},
			true, exported.Frozen.String(),
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.QueryServer.ClientStatus(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(tc.expStatus, res.Status)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryUpgradedConsensusStates() {
	var (
		req               *types.QueryUpgradedConsensusStateRequest
		expConsensusState *codectypes.Any
		height            int64
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"req is nil",
			func() {
				req = nil
			},
			false,
		},
		{
			"no plan",
			func() {
				req = &types.QueryUpgradedConsensusStateRequest{}
			},
			false,
		},
		{
			"valid consensus state",
			func() {
				req = &types.QueryUpgradedConsensusStateRequest{}
				lastHeight := types.NewHeight(0, uint64(s.ctx.BlockHeight()))
				height = int64(lastHeight.GetRevisionHeight())
				s.ctx = s.ctx.WithBlockHeight(height)

				expConsensusState = types.MustPackConsensusState(s.consensusState)
				bz := types.MustMarshalConsensusState(s.cdc, s.consensusState)
				err := s.keeper.SetUpgradedConsensusState(s.ctx, height, bz)
				s.Require().NoError(err)
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()

			res, err := s.keeper.UpgradedConsensusState(s.ctx, req)
			if tc.expPass {
				s.Require().NoError(err)
				s.Require().True(expConsensusState.Equal(res.UpgradedConsensusState))
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryClientParams() {
	ctx := sdk.WrapSDKContext(s.chainA.GetContext())
	expParams := types.DefaultParams()
	res, _ := s.chainA.QueryServer.ClientParams(ctx, &types.QueryClientParamsRequest{})
	s.Require().Equal(&expParams, res.Params)
}
