package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"
)

func (suite *KeeperTestSuite) TestSubmitCrossChainQueryResult() {
	var (
		msg    *types.MsgSubmitCrossChainQueryResult
		result types.QueryResult
		data   []byte
	)

	// checkList
	// 1. retrieve the query from privateStore
	// 2. remover query from privateStore
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

		msg = types.NewMsgSubmitCrossChainQueryResult("queryId", result, data)

		res, err := suite.chainA.GetSimApp().IBCQueryKeeper.SubmitCrossChainQueryResult(sdk.WrapSDKContext(suite.chainA.GetContext()), msg)

		if tc.expPass {
			suite.Require().NoError(err)
			suite.Require().NotNil(res)
			queryResult, found := suite.chainA.GetSimApp().IBCQueryKeeper.GetCrossChainQueryResult(suite.chainA.GetContext(), "queryId")

			suite.Require().True(found)
			suite.Require().Equal(result, queryResult)
		} else {
			suite.Require().Error(err)
		}
	}
}
