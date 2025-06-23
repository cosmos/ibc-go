package keeper_test

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/mock"
)

func (s *KeeperTestSuite) TestQueryClientState() {
	var (
		req            *types.QueryClientStateRequest
		expClientState *codectypes.Any
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"req is nil",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid clientID",
			func() {
				req = &types.QueryClientStateRequest{}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
		{
			"client not found",
			func() {
				req = &types.QueryClientStateRequest{
					ClientId: testClientID,
				}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrap(types.ErrClientNotFound, "tendermint-0").Error(),
			),
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupClients()

				var err error
				expClientState, err = types.PackClientState(path.EndpointA.GetClientState())
				s.Require().NoError(err)

				req = &types.QueryClientStateRequest{
					ClientId: path.EndpointA.ClientID,
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

			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ClientState(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expClientState, res.ClientState)

				// ensure UnpackInterfaces is defined
				cachedValue := res.ClientState.GetCachedValue()
				s.Require().NotNil(cachedValue)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
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
		expErr   error
	}{
		{
			"req is nil",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"empty pagination",
			func() {
				expClientStates = nil
				req = &types.QueryClientStatesRequest{}
			},
			nil,
		},
		{
			"success",
			func() {
				path1 := ibctesting.NewPath(s.chainA, s.chainB)
				path1.SetupClients()

				path2 := ibctesting.NewPath(s.chainA, s.chainB)
				path2.SetupClients()

				clientStateA1 := path1.EndpointA.GetClientState()
				clientStateA2 := path2.EndpointA.GetClientState()

				idcs := types.NewIdentifiedClientState(path1.EndpointA.ClientID, clientStateA1)
				idcs2 := types.NewIdentifiedClientState(path2.EndpointA.ClientID, clientStateA2)

				// order is sorted by client id
				expClientStates = types.IdentifiedClientStates{idcs, idcs2}.Sort()
				req = &types.QueryClientStatesRequest{
					Pagination: &query.PageRequest{
						Limit:      20,
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
			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ClientStates(ctx, req)
			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expClientStates.Sort(), res.ClientStates)
				s.Require().Equal(len(expClientStates), int(res.Pagination.Total))
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
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
		expErr   error
	}{
		{
			"req is nil",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid clientID",
			func() {
				req = &types.QueryConsensusStateRequest{}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
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
			status.Error(codes.InvalidArgument, "consensus state height cannot be 0"),
		},
		{
			"consensus state not found",
			func() {
				req = &types.QueryConsensusStateRequest{
					ClientId:     ibctesting.FirstClientID,
					LatestHeight: true,
				}
			},
			status.Error(codes.NotFound, "client-id: 07-tendermint-0, height: 0-0: consensus state not found"),
		},
		{
			"success latest height",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupClients()
				cs := path.EndpointA.GetConsensusState(path.EndpointA.GetClientLatestHeight())

				var err error
				expConsensusState, err = types.PackConsensusState(cs)
				s.Require().NoError(err)

				req = &types.QueryConsensusStateRequest{
					ClientId:     path.EndpointA.ClientID,
					LatestHeight: true,
				}
			},
			nil,
		},
		{
			"success with height",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupClients()
				height := path.EndpointA.GetClientLatestHeight()
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
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()
			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ConsensusState(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expConsensusState, res.ConsensusState)

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

func (s *KeeperTestSuite) TestQueryConsensusStates() {
	var (
		req                *types.QueryConsensusStatesRequest
		expConsensusStates []types.ConsensusStateWithHeight
	)

	testCases := []struct {
		msg      string
		malleate func()
		expErr   error
	}{
		{
			"success: without pagination",
			func() {
				req = &types.QueryConsensusStatesRequest{
					ClientId: testClientID,
				}
			},
			nil,
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
			nil,
		},
		{
			"success",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupClients()

				height1, ok := path.EndpointA.GetClientLatestHeight().(types.Height)
				s.Require().True(ok)
				expConsensusStates = append(
					expConsensusStates,
					types.NewConsensusStateWithHeight(
						height1,
						path.EndpointA.GetConsensusState(height1),
					))

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)
				height2, ok := path.EndpointA.GetClientLatestHeight().(types.Height)
				s.Require().True(ok)
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
			nil,
		},
		{
			"invalid client identifier",
			func() {
				req = &types.QueryConsensusStatesRequest{}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()
			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ConsensusStates(ctx, req)

			if tc.expErr == nil {
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
				s.Require().ErrorIs(err, tc.expErr)
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
		expErr   error
	}{
		{
			"success: without pagination",
			func() {
				req = &types.QueryConsensusStateHeightsRequest{
					ClientId: testClientID,
				}
			},
			nil,
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
			nil,
		},
		{
			"success: returns consensus heights",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupClients()

				expConsensusStateHeights = append(expConsensusStateHeights, path.EndpointA.GetClientLatestHeight().(types.Height))

				err := path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				expConsensusStateHeights = append(expConsensusStateHeights, path.EndpointA.GetClientLatestHeight().(types.Height))

				req = &types.QueryConsensusStateHeightsRequest{
					ClientId: path.EndpointA.ClientID,
					Pagination: &query.PageRequest{
						Limit:      3,
						CountTotal: true,
					},
				}
			},
			nil,
		},
		{
			"invalid client identifier",
			func() {
				req = &types.QueryConsensusStateHeightsRequest{}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()
			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ConsensusStateHeights(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(len(expConsensusStateHeights), len(res.ConsensusStateHeights))
				for i := range expConsensusStateHeights {
					s.Require().NotNil(res.ConsensusStateHeights[i])
					s.Require().Equal(expConsensusStateHeights[i], res.ConsensusStateHeights[i])
				}
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryClientStatus() {
	var req *types.QueryClientStatusRequest

	testCases := []struct {
		msg       string
		malleate  func()
		expErr    error
		expStatus string
	}{
		{
			"req is nil",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"), "",
		},
		{
			"invalid clientID",
			func() {
				req = &types.QueryClientStatusRequest{}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"), "",
		},
		{
			"client not found",
			func() {
				req = &types.QueryClientStatusRequest{
					ClientId: ibctesting.InvalidID,
				}
			},
			nil, exported.Unauthorized.String(),
		},
		{
			"Active client status",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupClients()
				req = &types.QueryClientStatusRequest{
					ClientId: path.EndpointA.ClientID,
				}
			},
			nil, exported.Active.String(),
		},
		{
			"Unknown client status",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupClients()
				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)

				// increment latest height so no consensus state is stored
				clientState.LatestHeight, ok = clientState.LatestHeight.Increment().(types.Height)
				s.Require().True(ok)
				path.EndpointA.SetClientState(clientState)

				req = &types.QueryClientStatusRequest{
					ClientId: path.EndpointA.ClientID,
				}
			},
			nil, exported.Expired.String(),
		},
		{
			"Frozen client status",
			func() {
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.SetupClients()
				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)

				clientState.FrozenHeight = types.NewHeight(0, 1)
				path.EndpointA.SetClientState(clientState)

				req = &types.QueryClientStatusRequest{
					ClientId: path.EndpointA.ClientID,
				}
			},
			nil, exported.Frozen.String(),
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()
			ctx := s.chainA.GetContext()
			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ClientStatus(ctx, req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(tc.expStatus, res.Status)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryUpgradedClientState() {
	var (
		req            *types.QueryUpgradedClientStateRequest
		path           *ibctesting.Path
		expClientState *ibctm.ClientState
	)

	upgradePlan := upgradetypes.Plan{
		Name:   "upgrade IBC clients",
		Height: 1000,
	}

	testCases := []struct {
		msg      string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				validAuthority := s.chainA.App.GetIBCKeeper().GetAuthority()

				// update trusting period
				clientState := path.EndpointA.GetClientState()
				clientState.(*ibctm.ClientState).TrustingPeriod += 100

				msg, err := types.NewMsgIBCSoftwareUpgrade(
					validAuthority,
					upgradePlan,
					clientState,
				)
				s.Require().NoError(err)

				resp, err := s.chainA.App.GetIBCKeeper().IBCSoftwareUpgrade(s.chainA.GetContext(), msg)
				s.Require().NoError(err)
				s.Require().NotNil(resp)

				var ok bool
				expClientState, ok = clientState.(*ibctm.ClientState)
				s.Require().True(ok)
			},
			nil,
		},
		{
			"req is nil",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"no plan",
			func() {
				req = &types.QueryUpgradedClientStateRequest{}
			},
			status.Error(codes.NotFound, "upgrade plan not found"),
		},
		{
			"no upgraded client set in store",
			func() {
				err := s.chainA.GetSimApp().UpgradeKeeper.ScheduleUpgrade(s.chainA.GetContext(), upgradePlan)
				s.Require().NoError(err)
			},
			status.Error(codes.NotFound, "upgraded client not found"),
		},
		{
			"invalid upgraded client state",
			func() {
				err := s.chainA.GetSimApp().UpgradeKeeper.ScheduleUpgrade(s.chainA.GetContext(), upgradePlan)
				s.Require().NoError(err)

				bz := []byte{1, 2, 3}
				err = s.chainA.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainA.GetContext(), upgradePlan.Height, bz)
				s.Require().NoError(err)
			},
			status.Error(codes.Internal, "proto: Any: illegal tag 0 (wire type 1)"),
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			req = &types.QueryUpgradedClientStateRequest{}

			tc.malleate()

			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.UpgradedClientState(s.chainA.GetContext(), req)

			if tc.expError == nil {
				s.Require().NoError(err)

				upgradedClientState, err := types.UnpackClientState(res.UpgradedClientState)
				s.Require().NoError(err)
				upgradedClientStateCmt, ok := upgradedClientState.(*ibctm.ClientState)
				s.Require().True(ok)

				s.Require().Equal(expClientState.ZeroCustomFields(), upgradedClientStateCmt)
			} else {
				s.Require().ErrorIs(err, tc.expError)
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
		expErr   error
	}{
		{
			"req is nil",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"no plan",
			func() {
				req = &types.QueryUpgradedConsensusStateRequest{}
			},
			status.Error(codes.NotFound, "upgraded consensus state not found, height 2"),
		},
		{
			"valid consensus state",
			func() {
				req = &types.QueryUpgradedConsensusStateRequest{}

				ctx := s.chainA.GetContext()
				lastHeight := types.NewHeight(0, uint64(ctx.BlockHeight()))
				height = int64(lastHeight.GetRevisionHeight())
				ctx = ctx.WithBlockHeight(height)

				expConsensusState = types.MustPackConsensusState(s.consensusState)
				bz := types.MustMarshalConsensusState(s.cdc, s.consensusState)
				err := s.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.SetUpgradedConsensusState(ctx, height, bz)
				s.Require().NoError(err)
			},
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate()

			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.UpgradedConsensusState(s.chainA.GetContext(), req)
			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().True(expConsensusState.Equal(res.UpgradedConsensusState))
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryCreator() {
	var (
		req    *types.QueryClientCreatorRequest
		expRes *types.QueryClientCreatorResponse
		path   *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"req is nil",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"invalid clientID",
			func() {
				req = &types.QueryClientCreatorRequest{}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
		{
			"client not found",
			func() {
				req = &types.QueryClientCreatorRequest{
					ClientId: ibctesting.FirstClientID,
				}
				expRes = &types.QueryClientCreatorResponse{
					Creator: "",
				}
			},
			nil,
		},
		{
			"success",
			func() {
				path.SetupClients()
				req = &types.QueryClientCreatorRequest{
					ClientId: path.EndpointA.ClientID,
				}
				expRes = &types.QueryClientCreatorResponse{
					Creator: s.chainA.SenderAccount.GetAddress().String(),
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.name), func() {
			s.SetupTest() // reset
			path = ibctesting.NewPath(s.chainA, s.chainB)

			tc.malleate()
			ctx := s.chainA.GetContext()
			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ClientCreator(ctx, req)

			if tc.expError == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expRes, res)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryClientParams() {
	ctx := s.chainA.GetContext()
	expParams := types.DefaultParams()
	queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
	res, _ := queryServer.ClientParams(ctx, &types.QueryClientParamsRequest{})
	s.Require().Equal(&expParams, res.Params)
}

func (s *KeeperTestSuite) TestQueryVerifyMembershipProof() {
	const wasmClientID = "08-wasm-0"

	var (
		path *ibctesting.Path
		req  *types.QueryVerifyMembershipRequest
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				channel := path.EndpointB.GetChannel()
				bz, err := s.chainB.Codec.Marshal(&channel)
				s.Require().NoError(err)

				channelProof, proofHeight := path.EndpointB.QueryProof(host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))

				merklePath := commitmenttypes.NewMerklePath(host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
				merklePath, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				req = &types.QueryVerifyMembershipRequest{
					ClientId:    path.EndpointA.ClientID,
					Proof:       channelProof,
					ProofHeight: proofHeight,
					MerklePath:  merklePath,
					Value:       bz,
				}
			},
			nil,
		},
		{
			"req is nil",
			func() {
				req = nil
			},
			errors.New("empty request"),
		},
		{
			"invalid client ID",
			func() {
				req = &types.QueryVerifyMembershipRequest{
					ClientId: "//invalid_id",
				}
			},
			host.ErrInvalidID,
		},
		{
			"localhost client ID is denied",
			func() {
				req = &types.QueryVerifyMembershipRequest{
					ClientId: exported.LocalhostClientID,
				}
			},
			types.ErrInvalidClientType,
		},
		{
			"solomachine client ID is denied",
			func() {
				req = &types.QueryVerifyMembershipRequest{
					ClientId: types.FormatClientIdentifier(exported.Solomachine, 1),
				}
			},
			types.ErrInvalidClientType,
		},
		{
			"empty proof",
			func() {
				req = &types.QueryVerifyMembershipRequest{
					ClientId: ibctesting.FirstClientID,
					Proof:    []byte{},
				}
			},
			errors.New("empty proof"),
		},
		{
			"invalid proof height",
			func() {
				req = &types.QueryVerifyMembershipRequest{
					ClientId:    ibctesting.FirstClientID,
					Proof:       []byte{0x01},
					ProofHeight: types.ZeroHeight(),
				}
			},
			errors.New("proof height must be non-zero"),
		},
		{
			"empty merkle path",
			func() {
				req = &types.QueryVerifyMembershipRequest{
					ClientId:    ibctesting.FirstClientID,
					Proof:       []byte{0x01},
					ProofHeight: types.NewHeight(1, 100),
				}
			},
			errors.New("empty merkle path"),
		},
		{
			"empty value",
			func() {
				req = &types.QueryVerifyMembershipRequest{
					ClientId:    ibctesting.FirstClientID,
					Proof:       []byte{0x01},
					ProofHeight: types.NewHeight(1, 100),
					MerklePath:  commitmenttypes.NewMerklePath([]byte("/ibc"), host.ChannelKey(mock.PortID, ibctesting.FirstChannelID)),
				}
			},
			errors.New("empty value"),
		},
		{
			"light client module not found",
			func() {
				req = &types.QueryVerifyMembershipRequest{
					ClientId:    wasmClientID, // use a client type that is not registered
					Proof:       []byte{0x01},
					ProofHeight: types.NewHeight(1, 100),
					MerklePath:  commitmenttypes.NewMerklePath([]byte("/ibc"), host.ChannelKey(mock.PortID, ibctesting.FirstChannelID)),
					Value:       []byte{0x01},
				}
			},
			errors.New(wasmClientID),
		},
		{
			"client type not allowed",
			func() {
				params := types.NewParams("") // disable all clients
				s.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.SetParams(s.chainA.GetContext(), params)

				req = &types.QueryVerifyMembershipRequest{
					ClientId:    path.EndpointA.ClientID,
					Proof:       []byte{0x01},
					ProofHeight: types.NewHeight(1, 100),
					MerklePath:  commitmenttypes.NewMerklePath([]byte("/ibc"), host.ChannelKey(mock.PortID, ibctesting.FirstChannelID)),
					Value:       []byte{0x01},
				}
			},
			types.ErrInvalidClientType,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.Setup()

			tc.malleate()

			ctx := s.chainA.GetContext()
			initialGas := ctx.GasMeter().GasConsumed()
			queryServer := keeper.NewQueryServer(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.VerifyMembership(ctx, req)

			if tc.expError == nil {
				s.Require().NoError(err)
				s.Require().True(res.Success, "failed to verify membership proof")

				gasConsumed := ctx.GasMeter().GasConsumed()
				s.Require().Greater(gasConsumed, initialGas, "gas consumed should be greater than initial gas")
			} else {
				s.Require().ErrorContains(err, tc.expError.Error())

				gasConsumed := ctx.GasMeter().GasConsumed()
				s.Require().GreaterOrEqual(gasConsumed, initialGas, "gas consumed should be greater than or equal to initial gas")
			}
		})
	}
}
