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

func (suite *KeeperTestSuite) TestQueryClientState() {
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
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupClients()

				var err error
				expClientState, err = types.PackClientState(path.EndpointA.GetClientState())
				suite.Require().NoError(err)

				req = &types.QueryClientStateRequest{
					ClientId: path.EndpointA.ClientID,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()

			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ClientState(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expClientState, res.ClientState)

				// ensure UnpackInterfaces is defined
				cachedValue := res.ClientState.GetCachedValue()
				suite.Require().NotNil(cachedValue)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryClientStates() {
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
				path1 := ibctesting.NewPath(suite.chainA, suite.chainB)
				path1.SetupClients()

				path2 := ibctesting.NewPath(suite.chainA, suite.chainB)
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
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			tc.malleate()

			ctx := suite.chainA.GetContext()
			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ClientStates(ctx, req)
			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expClientStates.Sort(), res.ClientStates)
				suite.Require().Equal(len(expClientStates), int(res.Pagination.Total))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryConsensusState() {
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
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupClients()
				cs := path.EndpointA.GetConsensusState(path.EndpointA.GetClientLatestHeight())

				var err error
				expConsensusState, err = types.PackConsensusState(cs)
				suite.Require().NoError(err)

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
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupClients()
				height := path.EndpointA.GetClientLatestHeight()
				cs := path.EndpointA.GetConsensusState(height)

				var err error
				expConsensusState, err = types.PackConsensusState(cs)
				suite.Require().NoError(err)

				// update client to new height
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

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
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()
			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ConsensusState(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expConsensusState, res.ConsensusState)

				// ensure UnpackInterfaces is defined
				cachedValue := res.ConsensusState.GetCachedValue()
				suite.Require().NotNil(cachedValue)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryConsensusStates() {
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
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupClients()

				height1, ok := path.EndpointA.GetClientLatestHeight().(types.Height)
				suite.Require().True(ok)
				expConsensusStates = append(
					expConsensusStates,
					types.NewConsensusStateWithHeight(
						height1,
						path.EndpointA.GetConsensusState(height1),
					))

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
				height2, ok := path.EndpointA.GetClientLatestHeight().(types.Height)
				suite.Require().True(ok)
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
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()
			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ConsensusStates(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(len(expConsensusStates), len(res.ConsensusStates))
				for i := range expConsensusStates {
					suite.Require().NotNil(res.ConsensusStates[i])
					suite.Require().Equal(expConsensusStates[i], res.ConsensusStates[i])
					// ensure UnpackInterfaces is defined
					cachedValue := res.ConsensusStates[i].ConsensusState.GetCachedValue()
					suite.Require().NotNil(cachedValue)
				}
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryConsensusStateHeights() {
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
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupClients()

				expConsensusStateHeights = append(expConsensusStateHeights, path.EndpointA.GetClientLatestHeight().(types.Height))

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

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
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()
			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ConsensusStateHeights(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(len(expConsensusStateHeights), len(res.ConsensusStateHeights))
				for i := range expConsensusStateHeights {
					suite.Require().NotNil(res.ConsensusStateHeights[i])
					suite.Require().Equal(expConsensusStateHeights[i], res.ConsensusStateHeights[i])
				}
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryClientStatus() {
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
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
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
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupClients()
				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)

				// increment latest height so no consensus state is stored
				clientState.LatestHeight, ok = clientState.LatestHeight.Increment().(types.Height)
				suite.Require().True(ok)
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
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.SetupClients()
				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)

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
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := suite.chainA.GetContext()
			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ClientStatus(ctx, req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(tc.expStatus, res.Status)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryUpgradedClientState() {
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
				validAuthority := suite.chainA.App.GetIBCKeeper().GetAuthority()

				// update trusting period
				clientState := path.EndpointA.GetClientState()
				clientState.(*ibctm.ClientState).TrustingPeriod += 100

				msg, err := types.NewMsgIBCSoftwareUpgrade(
					validAuthority,
					upgradePlan,
					clientState,
				)
				suite.Require().NoError(err)

				resp, err := suite.chainA.App.GetIBCKeeper().IBCSoftwareUpgrade(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)
				suite.Require().NotNil(resp)

				var ok bool
				expClientState, ok = clientState.(*ibctm.ClientState)
				suite.Require().True(ok)
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
				err := suite.chainA.GetSimApp().UpgradeKeeper.ScheduleUpgrade(suite.chainA.GetContext(), upgradePlan)
				suite.Require().NoError(err)
			},
			status.Error(codes.NotFound, "upgraded client not found"),
		},
		{
			"invalid upgraded client state",
			func() {
				err := suite.chainA.GetSimApp().UpgradeKeeper.ScheduleUpgrade(suite.chainA.GetContext(), upgradePlan)
				suite.Require().NoError(err)

				bz := []byte{1, 2, 3}
				err = suite.chainA.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainA.GetContext(), upgradePlan.Height, bz)
				suite.Require().NoError(err)
			},
			status.Error(codes.Internal, "proto: Any: illegal tag 0 (wire type 1)"),
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			req = &types.QueryUpgradedClientStateRequest{}

			tc.malleate()

			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.UpgradedClientState(suite.chainA.GetContext(), req)

			if tc.expError == nil {
				suite.Require().NoError(err)

				upgradedClientState, err := types.UnpackClientState(res.UpgradedClientState)
				suite.Require().NoError(err)
				upgradedClientStateCmt, ok := upgradedClientState.(*ibctm.ClientState)
				suite.Require().True(ok)

				suite.Require().Equal(expClientState.ZeroCustomFields(), upgradedClientStateCmt)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryUpgradedConsensusStates() {
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

				ctx := suite.chainA.GetContext()
				lastHeight := types.NewHeight(0, uint64(ctx.BlockHeight()))
				height = int64(lastHeight.GetRevisionHeight())
				ctx = ctx.WithBlockHeight(height)

				expConsensusState = types.MustPackConsensusState(suite.consensusState)
				bz := types.MustMarshalConsensusState(suite.cdc, suite.consensusState)
				err := suite.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.SetUpgradedConsensusState(ctx, height, bz)
				suite.Require().NoError(err)
			},
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()

			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.UpgradedConsensusState(suite.chainA.GetContext(), req)
			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().True(expConsensusState.Equal(res.UpgradedConsensusState))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryCreator() {
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
					Creator: suite.chainA.SenderAccount.GetAddress().String(),
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			tc.malleate()
			ctx := suite.chainA.GetContext()
			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.ClientCreator(ctx, req)

			if tc.expError == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expRes, res)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryClientParams() {
	ctx := suite.chainA.GetContext()
	expParams := types.DefaultParams()
	queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
	res, _ := queryServer.ClientParams(ctx, &types.QueryClientParamsRequest{})
	suite.Require().Equal(&expParams, res.Params)
}

func (suite *KeeperTestSuite) TestQueryVerifyMembershipProof() {
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
				bz, err := suite.chainB.Codec.Marshal(&channel)
				suite.Require().NoError(err)

				channelProof, proofHeight := path.EndpointB.QueryProof(host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))

				merklePath := commitmenttypes.NewMerklePath(host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
				merklePath, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

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
				suite.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.SetParams(suite.chainA.GetContext(), params)

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
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			tc.malleate()

			ctx := suite.chainA.GetContext()
			initialGas := ctx.GasMeter().GasConsumed()
			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			res, err := queryServer.VerifyMembership(ctx, req)

			if tc.expError == nil {
				suite.Require().NoError(err)
				suite.Require().True(res.Success, "failed to verify membership proof")

				gasConsumed := ctx.GasMeter().GasConsumed()
				suite.Require().Greater(gasConsumed, initialGas, "gas consumed should be greater than initial gas")
			} else {
				suite.Require().ErrorContains(err, tc.expError.Error())

				gasConsumed := ctx.GasMeter().GasConsumed()
				suite.Require().GreaterOrEqual(gasConsumed, initialGas, "gas consumed should be greater than or equal to initial gas")
			}
		})
	}
}
