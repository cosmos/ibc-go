package types_test

import (
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/log/v2"
	"github.com/cosmos/cosmos-sdk/store/v2/legacy/rootmulti"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
)

type MerkleTestSuite struct {
	testifysuite.Suite

	store    *rootmulti.Store
	storeKey *storetypes.KVStoreKey
	kvStore  storetypes.KVStore
}

func (s *MerkleTestSuite) SetupTest() {
	db := dbm.NewMemDB()
	s.store = rootmulti.NewStore(db, log.NewNopLogger())

	s.storeKey = storetypes.NewKVStoreKey("iavlStoreKey")

	s.store.MountStoreWithDB(s.storeKey, storetypes.StoreTypeIAVL, nil)
	err := s.store.LoadVersion(0)
	s.Require().NoError(err)

	s.kvStore = s.store.GetKVStore(s.storeKey)
}

func TestMerkleTestSuite(t *testing.T) {
	testifysuite.Run(t, new(MerkleTestSuite))
}
