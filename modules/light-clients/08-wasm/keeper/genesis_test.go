package keeper_test

import (
	"encoding/hex"
	"os"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

func (suite *KeeperTestSuite) TestInitGenesis() {
	var (
		genesisState  types.GenesisState
		expCodeHashes []string
	)

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"success",
			func() {
				codeHash := "561715ea6ee1dce8f78499914e6c7853dc315a5e6ecf01da09a9054a160e5d1d"
				contractCode, err := os.ReadFile("../test_data/ics07_tendermint_cw.wasm.gz")
				suite.Require().NoError(err)

				genesisState = *types.NewGenesisState(
					[]types.Contract{
						{
							CodeBytes: contractCode,
						},
					},
				)

				expCodeHashes = []string{codeHash}
			},
		},
		{
			"success with empty genesis contract",
			func() {
				genesisState = *types.NewGenesisState([]types.Contract{})
				expCodeHashes = []string{}
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			ctx := suite.chainA.GetContext()
			tc.malleate()

			err := suite.chainA.GetSimApp().WasmClientKeeper.InitGenesis(ctx, genesisState)
			suite.Require().NoError(err)

			res := types.GetCodeHashes(suite.chainA.GetContext(), suite.chainA.GetSimApp().AppCodec())
			for idx, codeHash := range res {
				res[idx] = hex.EncodeToString([]byte(codeHash))
			}

			suite.Require().NoError(err)
			suite.Require().NotNil(res)
			suite.Require().Equal(len(expCodeHashes), len(res))
			suite.Require().ElementsMatch(expCodeHashes, res)
		})
	}
}

func (suite *KeeperTestSuite) TestExportGenesis() {
	suite.SetupTest()
	ctx := suite.chainA.GetContext()

	expCodeHash := "561715ea6ee1dce8f78499914e6c7853dc315a5e6ecf01da09a9054a160e5d1d"

	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	contractCode, err := os.ReadFile("../test_data/ics07_tendermint_cw.wasm.gz")
	suite.Require().NoError(err)

	msg := types.NewMsgStoreCode(signer, contractCode)
	res, err := suite.chainA.GetSimApp().WasmClientKeeper.StoreCode(ctx, msg)
	suite.Require().NoError(err)
	suite.Require().Equal(expCodeHash, hex.EncodeToString(res.Checksum))

	genesisState := suite.chainA.GetSimApp().WasmClientKeeper.ExportGenesis(ctx)
	suite.Require().Len(genesisState.Contracts, 1)
	suite.Require().NotEmpty(genesisState.Contracts[0].CodeBytes)
}
