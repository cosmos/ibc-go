package keeper_test

import (
	"encoding/hex"
	"os"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

func (suite *KeeperTestSuite) TestQueryCode() {
	var (
		req *types.QueryCodeRequest
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
				code, err := os.ReadFile("../test_data/ics10_grandpa_cw.wasm.gz")
				suite.Require().NoError(err)
				msg := types.NewMsgStoreCode(signer, code)

				res, err := suite.chainA.GetSimApp().WasmClientKeeper.StoreCode(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				req = &types.QueryCodeRequest{CodeId: hex.EncodeToString(res.CodeId)}
			},
			true,
		},
		{
			"fails with empty request",
			func() {
				req = &types.QueryCodeRequest{}
			},
			false,
		},
		{
			"fails with non-existent code ID",
			func() {
				req = &types.QueryCodeRequest{CodeId: "test"}
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			res, err := suite.chainA.GetSimApp().WasmClientKeeper.Code(suite.chainA.GetContext(), req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().NotEmpty(res.Code)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryCodeIDs() {
	var expCodeIds []string

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success with no code IDs",
			func() {
				expCodeIds = []string{}
			},
			true,
		},
		{
			"success with one code ID",
			func() {
				signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
				code, err := os.ReadFile("../test_data/ics10_grandpa_cw.wasm.gz")
				suite.Require().NoError(err)
				msg := types.NewMsgStoreCode(signer, code)

				res, err := suite.chainA.GetSimApp().WasmClientKeeper.StoreCode(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				expCodeIds = append(expCodeIds, hex.EncodeToString(res.CodeId))
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.malleate()

			req := &types.QueryCodeIdsRequest{}
			res, err := suite.chainA.GetSimApp().WasmClientKeeper.CodeIds(suite.chainA.GetContext(), req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(len(expCodeIds), len(res.CodeIds))
				suite.Require().ElementsMatch(expCodeIds, res.CodeIds)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
