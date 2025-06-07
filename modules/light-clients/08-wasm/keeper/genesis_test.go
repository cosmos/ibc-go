package keeper_test

import (
	"encoding/hex"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
)

func (s *KeeperTestSuite) TestInitGenesis() {
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
				checksum := "b3a49b2914f5e6a673215e74325c1d153bb6776e079774e52c5b7e674d9ad3ab" //nolint:gosec // these are not hard-coded credentials

				genesisState = *types.NewGenesisState(
					[]types.Contract{
						{
							CodeBytes: wasmtesting.Code,
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
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()

			ctx := s.chainA.GetContext()
			tc.malleate()

			err := GetSimApp(s.chainA).WasmClientKeeper.InitGenesis(ctx, genesisState)
			s.Require().NoError(err)

			var storedHashes []string
			checksums, err := GetSimApp(s.chainA).WasmClientKeeper.GetAllChecksums(s.chainA.GetContext())
			s.Require().NoError(err)

			for _, hash := range checksums {
				storedHashes = append(storedHashes, hex.EncodeToString(hash))
			}

			s.Require().Equal(len(expChecksums), len(storedHashes))
			s.Require().ElementsMatch(expChecksums, storedHashes)
		})
	}
}

func (s *KeeperTestSuite) TestExportGenesis() {
	s.SetupWasmWithMockVM()

	ctx := s.chainA.GetContext()

	expChecksum := "b3a49b2914f5e6a673215e74325c1d153bb6776e079774e52c5b7e674d9ad3ab" //nolint:gosec // these are not hard-coded credentials

	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	msg := types.NewMsgStoreCode(signer, wasmtesting.Code)
	res, err := GetSimApp(s.chainA).WasmClientKeeper.StoreCode(ctx, msg)
	s.Require().NoError(err)
	s.Require().Equal(expChecksum, hex.EncodeToString(res.Checksum))

	genesisState := GetSimApp(s.chainA).WasmClientKeeper.ExportGenesis(ctx)
	s.Require().Len(genesisState.Contracts, 1)
	s.Require().NotEmpty(genesisState.Contracts[0].CodeBytes)
}
