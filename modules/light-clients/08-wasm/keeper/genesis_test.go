package keeper_test

import (
	"encoding/hex"
	"os"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

func (suite *KeeperTestSuite) TestInitGenesis() {
	var (
		genesisState types.GenesisState
		expCodeIds   []string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				codeID := "c64f75091a6195b036f472cd8c9f19a56780b9eac3c3de7ced0ec2e29e985b64"
				codeIDBytes, err := hex.DecodeString(codeID)
				suite.Require().NoError(err)
				contractCode, err := os.ReadFile("../test_data/ics07_tendermint_cw.wasm.gz")
				suite.Require().NoError(err)

				genesisState = *types.NewGenesisState(
					[]types.GenesisContract{
						{
							CodeIdKey:    types.CodeIDKey(codeIDBytes),
							ContractCode: contractCode,
						},
					},
				)

				expCodeIds = []string{codeID}
			},
			true,
		},
		{
			"success with empty genesis contract",
			func() {
				genesisState = *types.NewGenesisState([]types.GenesisContract{})
				expCodeIds = []string{}
			},
			true,
		},
		{
			"failure with genesis contract with code ID that does not match hash of contract code",
			func() {
				codeID := "wrong-code-id"
				contractCode, err := os.ReadFile("../test_data/ics07_tendermint_cw.wasm.gz")
				suite.Require().NoError(err)

				genesisState = *types.NewGenesisState(
					[]types.GenesisContract{
						{
							CodeIdKey:    types.CodeIDKey([]byte(codeID)),
							ContractCode: contractCode,
						},
					},
				)
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := suite.chainA.GetContext()
			tc.malleate()

			err := suite.chainA.GetSimApp().WasmClientKeeper.InitGenesis(ctx, genesisState)

			if tc.expPass {
				suite.Require().NoError(err)

				req := &types.QueryCodeIdsRequest{}
				res, err := suite.chainA.GetSimApp().WasmClientKeeper.CodeIds(ctx, req)
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

func (suite *KeeperTestSuite) TestExportGenesis() {
	suite.SetupTest()
	ctx := suite.chainA.GetContext()

	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	contractCode, err := os.ReadFile("../test_data/ics07_tendermint_cw.wasm.gz")
	suite.Require().NoError(err)
	msg := types.NewMsgStoreCode(signer, contractCode)
	res, err := suite.chainA.GetSimApp().WasmClientKeeper.StoreCode(ctx, msg)
	suite.Require().NoError(err)
	codeIDKey := types.CodeIDKey(res.CodeId)

	genesisState := suite.chainA.GetSimApp().WasmClientKeeper.ExportGenesis(ctx)
	suite.Require().Len(genesisState.Contracts, 1)

	suite.Require().Equal(codeIDKey, genesisState.Contracts[0].CodeIdKey)
	suite.Require().NotEmpty(genesisState.Contracts[0].ContractCode)
}
