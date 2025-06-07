package keeper_test

import (
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/keeper"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
)

func (s *KeeperTestSuite) TestMigrateWasmStore() {
	testCases := []struct {
		name      string
		checksums [][]byte
	}{
		{
			"success: empty checksums",
			[][]byte{},
		},
		{
			"success: multiple checksums",
			[][]byte{[]byte("hash1"), []byte("hash2"), []byte("hash3")},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			s.storeChecksums(tc.checksums)

			// run the migration
			wasmClientKeeper := GetSimApp(s.chainA).WasmClientKeeper
			m := keeper.NewMigrator(wasmClientKeeper)

			err := m.MigrateChecksums(s.chainA.GetContext())
			s.Require().NoError(err)

			// check that they were stored in KeySet
			for _, hash := range tc.checksums {
				s.Require().True(wasmClientKeeper.GetChecksums().Has(s.chainA.GetContext(), hash))
			}

			// check that the data under the old key was deleted
			store := s.chainA.GetContext().KVStore(GetSimApp(s.chainA).GetKey(types.StoreKey))
			s.Require().Nil(store.Get([]byte(types.KeyChecksums)))
		})
	}
}

// storeChecksums stores the given checksums under the KeyChecksums key, it runs
// each time on an empty store so we don't need to read the previous checksums.
func (s *KeeperTestSuite) storeChecksums(checksums [][]byte) {
	ctx := s.chainA.GetContext()

	store := ctx.KVStore(GetSimApp(s.chainA).GetKey(types.StoreKey))
	checksum := types.Checksums{Checksums: checksums}
	bz, err := GetSimApp(s.chainA).AppCodec().Marshal(&checksum)
	s.Require().NoError(err)

	store.Set([]byte(types.KeyChecksums), bz)
}
