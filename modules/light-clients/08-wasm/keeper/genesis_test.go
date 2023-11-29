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
		genesisState types.GenesisState
		expChecksums []string
	)

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"success",
			func() {
				checksum := "9b18dc4aa6a4dc6183f148bdcadbf7d3de2fdc7aac59394f1589b81e77de5e3c" //nolint:gosec // these are not hard-coded credentials
				contractCode, err := os.ReadFile("../test_data/ics07_tendermint_cw.wasm.gz")
				suite.Require().NoError(err)

				genesisState = *types.NewGenesisState(
					[]types.Contract{
						{
							CodeBytes: contractCode,
						},
					},
				)

				expChecksums = []string{checksum}
			},
		},
		{
			"success with empty genesis contract",
			func() {
				genesisState = *types.NewGenesisState([]types.Contract{})
				expChecksums = []string{}
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
			checksums, err := types.GetAllChecksums(suite.chainA.GetContext(), suite.chainA.App.AppCodec())
			suite.Require().NoError(err)

			for _, hash := range checksums {
				storedHashes = append(storedHashes, hex.EncodeToString(hash))
			}

			suite.Require().Equal(len(expChecksums), len(storedHashes))
			suite.Require().ElementsMatch(expChecksums, storedHashes)
		})
	}
}

func (suite *KeeperTestSuite) TestExportGenesis() {
	suite.SetupTest()
	ctx := suite.chainA.GetContext()

	expChecksum := "9b18dc4aa6a4dc6183f148bdcadbf7d3de2fdc7aac59394f1589b81e77de5e3c" //nolint:gosec // these are not hard-coded credentials

	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	contractCode, err := os.ReadFile("../test_data/ics07_tendermint_cw.wasm.gz")
	suite.Require().NoError(err)

	msg := types.NewMsgStoreCode(signer, contractCode)
	res, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(ctx, msg)
	suite.Require().NoError(err)
	suite.Require().Equal(expChecksum, hex.EncodeToString(res.Checksum))

	genesisState := GetSimApp(suite.chainA).WasmClientKeeper.ExportGenesis(ctx)
	suite.Require().Len(genesisState.Contracts, 1)
	suite.Require().NotEmpty(genesisState.Contracts[0].CodeBytes)
}
