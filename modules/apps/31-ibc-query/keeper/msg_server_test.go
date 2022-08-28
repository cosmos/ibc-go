package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
)


var (
	timeoutHeight        = clienttypes.NewHeight(0, 100)
	timeoutTimestamp     = uint64(0)
)

func (suite *KeeperTestSuite) TestSubmitCrossChainQuery() {
	var (
		msg    *types.MsgSubmitCrossChainQuery
	)

	testCases := []struct {
		name     string
		expPass  bool
		malleate func()
	}{
		{
			"success",
			true,
			func() {
				msg = types.NewMsgSubmitCrossChainQuery("query-1", "test/query_path", timeoutHeight.RevisionHeight, timeoutTimestamp, 12, "client-1", "cosmos1234565")
			},
		},
	}

	for _, tc := range testCases {
		suite.SetupTest()

		tc.malleate()
		res, err := suite.chainA.GetSimApp().IBCQueryKeeper.SubmitCrossChainQuery(sdk.WrapSDKContext(suite.chainA.GetContext()), msg)

		if tc.expPass {
			suite.Require().NoError(err)
			suite.Require().NotNil(res)
			queryResult, found := suite.chainA.GetSimApp().IBCQueryKeeper.GetSubmitCrossChainQuery(suite.chainA.GetContext(), "query-1")

			suite.Require().True(found)
			suite.Require().Equal("query-1", queryResult.Id)
			suite.Require().Equal("test/query_path", queryResult.Path)
			suite.Require().Equal(timeoutHeight.RevisionHeight, queryResult.LocalTimeoutHeight)
			suite.Require().Equal(timeoutTimestamp, queryResult.LocalTimeoutStamp)
			suite.Require().Equal(uint64(0xc), queryResult.QueryHeight)
			suite.Require().Equal("client-1", queryResult.ClientId)
			suite.Require().Equal("cosmos1234565", queryResult.Sender)
		} else {
			suite.Require().Error(err)
		}
	}
}



func (suite *KeeperTestSuite) TestSubmitCrossChainQueryResult() {
	var (
		msg    *types.MsgSubmitCrossChainQueryResult
		result types.QueryResult
		data   []byte
	)

	// checkList
	// 1. retrieve the query from privateStore
	// 2. remove query from privateStore
	// 3. store result in privateStore

	testCases := []struct {
		name     string
		expPass  bool
		malleate func()
	}{
		{
			"success",
			true,
			func() {
				result = types.QueryResult_QUERY_RESULT_SUCCESS
				data = []byte("query data")
			},
		},
	}

	for _, tc := range testCases {
		suite.SetupTest()

		tc.malleate()

		query := &types.CrossChainQuery{Id: "queryId"}
		suite.chainA.GetSimApp().IBCQueryKeeper.SetCrossChainQuery(suite.chainA.GetContext(), query)

		msg = types.NewMsgSubmitCrossChainQueryResult("queryId", result, data)
		res, err := suite.chainA.GetSimApp().IBCQueryKeeper.SubmitCrossChainQueryResult(sdk.WrapSDKContext(suite.chainA.GetContext()), msg)

		if tc.expPass {
			suite.Require().NoError(err)
			suite.Require().NotNil(res)
			queryResult, found := suite.chainA.GetSimApp().IBCQueryKeeper.GetCrossChainQueryResult(suite.chainA.GetContext(), "queryId")

			suite.Require().True(found)
			suite.Require().Equal("queryId", queryResult.Id)
			suite.Require().Equal(result, queryResult.Result)
			suite.Require().Equal(data, queryResult.Data)
		} else {
			suite.Require().Error(err)
		}
	}
}
