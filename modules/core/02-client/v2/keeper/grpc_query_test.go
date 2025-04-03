package keeper_test

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (suite *KeeperTestSuite) TestQueryCounterPartyInfo() {
	var (
		req     *types.QueryCounterpartyInfoRequest
		expInfo = types.CounterpartyInfo{}
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
			"req has no ID",
			func() {
				req = &types.QueryCounterpartyInfoRequest{}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
		{
			"counterparty not found",
			func() {
				path1 := ibctesting.NewPath(suite.chainA, suite.chainB)
				path1.SetupClients()
				// counter party not set up

				expInfo = types.NewCounterpartyInfo(path1.EndpointA.Counterparty.MerklePathPrefix.KeyPath, path1.EndpointA.ClientID)
				req = &types.QueryCounterpartyInfoRequest{
					ClientId: path1.EndpointA.ClientID,
				}
			},
			status.Error(codes.NotFound, "client 07-tendermint-0 counterparty not found"),
		},
		{
			"success",
			func() {
				path1 := ibctesting.NewPath(suite.chainA, suite.chainB)
				path1.SetupClients()
				path1.SetupCounterparties()

				expInfo = types.NewCounterpartyInfo(path1.EndpointA.Counterparty.MerklePathPrefix.KeyPath, path1.EndpointA.ClientID)
				req = &types.QueryCounterpartyInfoRequest{
					ClientId: path1.EndpointA.ClientID,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			tc.malleate()

			ctx := suite.chainA.GetContext()
			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientV2Keeper)
			res, err := queryServer.CounterpartyInfo(ctx, req)
			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expInfo, *res.CounterpartyInfo)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryConfig() {
	var (
		req       *types.QueryConfigRequest
		expConfig = types.Config{}
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
			"req has no ID",
			func() {
				req = &types.QueryConfigRequest{}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
		{
			"success with default config",
			func() {
				path1 := ibctesting.NewPath(suite.chainA, suite.chainB)
				path1.SetupClients()

				expConfig = types.DefaultConfig()
				req = &types.QueryConfigRequest{
					ClientId: path1.EndpointA.ClientID,
				}
			},
			nil,
		},
		{
			"success with custom config",
			func() {
				path1 := ibctesting.NewPath(suite.chainA, suite.chainB)
				path1.SetupClients()

				expConfig = types.NewConfig(ibctesting.TestAccAddress)
				suite.chainA.App.GetIBCKeeper().ClientV2Keeper.SetConfig(suite.chainA.GetContext(), path1.EndpointA.ClientID, expConfig)
				req = &types.QueryConfigRequest{
					ClientId: path1.EndpointA.ClientID,
				}
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			tc.malleate()

			ctx := suite.chainA.GetContext()
			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.ClientV2Keeper)
			res, err := queryServer.Config(ctx, req)
			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expConfig, *res.Config)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
