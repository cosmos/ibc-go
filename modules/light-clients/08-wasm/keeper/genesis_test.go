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
				codeHash := "9b18dc4aa6a4dc6183f148bdcadbf7d3de2fdc7aac59394f1589b81e77de5e3c" //nolint:gosec // these are not hard-coded credentials
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

			err := GetSimApp(suite.chainA).WasmClientKeeper.InitGenesis(ctx, genesisState)
			suite.Require().NoError(err)

			var storedHashes []string
			codeHashes, err := types.GetAllCodeHashes(suite.chainA.GetContext())
			suite.Require().NoError(err)

			for _, hash := range codeHashes {
				storedHashes = append(storedHashes, hex.EncodeToString(hash))
			}

			suite.Require().Equal(len(expCodeHashes), len(storedHashes))
			suite.Require().ElementsMatch(expCodeHashes, storedHashes)
		})
	}
}

func (suite *KeeperTestSuite) TestExportGenesis() {
	suite.SetupTest()
	ctx := suite.chainA.GetContext()

	expCodeHash := "9b18dc4aa6a4dc6183f148bdcadbf7d3de2fdc7aac59394f1589b81e77de5e3c" //nolint:gosec // these are not hard-coded credentials

	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	contractCode, err := os.ReadFile("../test_data/ics07_tendermint_cw.wasm.gz")
	suite.Require().NoError(err)

	msg := types.NewMsgStoreCode(signer, contractCode)
	res, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(ctx, msg)
	suite.Require().NoError(err)
	suite.Require().Equal(expCodeHash, hex.EncodeToString(res.Checksum))

	genesisState := GetSimApp(suite.chainA).WasmClientKeeper.ExportGenesis(ctx)
	suite.Require().Len(genesisState.Contracts, 1)
	suite.Require().NotEmpty(genesisState.Contracts[0].CodeBytes)
}
