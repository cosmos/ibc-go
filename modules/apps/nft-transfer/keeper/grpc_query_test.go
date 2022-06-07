package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
)

func (suite *KeeperTestSuite) TestQueryClassTrace() {
	var (
		req      *types.QueryClassTraceRequest
		expTrace types.ClassTrace
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"invalid hex hash",
			func() {
				req = &types.QueryClassTraceRequest{Hash: "!@#!@#!"}
			},
			false,
		},
		{
			"not found class trace",
			func() {
				expTrace.Path = "nft-transfer/channelToA/nft-transfer/channelToB"
				expTrace.BaseClassId = "kitty"
				req = &types.QueryClassTraceRequest{
					Hash: expTrace.Hash().String(),
				}
			},
			false,
		},
		{
			"success",
			func() {
				expTrace.Path = "nft-transfer/channelToA/nft-transfer/channelToB"
				expTrace.BaseClassId = "kitty"
				suite.chainA.GetSimApp().NFTTransferKeeper.SetClassTrace(suite.chainA.GetContext(), expTrace)

				req = &types.QueryClassTraceRequest{
					Hash: expTrace.Hash().String(),
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.ClassTrace(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(&expTrace, res.ClassTrace)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryClassTraces() {
	var (
		req       *types.QueryClassTracesRequest
		expTraces = types.Traces(nil)
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty pagination",
			func() {
				req = &types.QueryClassTracesRequest{}
			},
			true,
		},
		{
			"success",
			func() {
				expTraces = append(expTraces, types.ClassTrace{Path: "", BaseClassId: "kitty"})
				expTraces = append(expTraces, types.ClassTrace{Path: "transfer/channelToB", BaseClassId: "kitty"})
				expTraces = append(expTraces, types.ClassTrace{Path: "transfer/channelToA/transfer/channelToB", BaseClassId: "kitty"})

				for _, trace := range expTraces {
					suite.chainA.GetSimApp().NFTTransferKeeper.SetClassTrace(suite.chainA.GetContext(), trace)
				}

				req = &types.QueryClassTracesRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())

			res, err := suite.queryClient.ClassTraces(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expTraces.Sort(), res.ClassTraces)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
