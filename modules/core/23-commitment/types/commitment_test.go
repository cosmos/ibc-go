package types_test

import (
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	"cosmossdk.io/store/iavl"
	"cosmossdk.io/store/metrics"
	"cosmossdk.io/store/rootmulti"
	storetypes "cosmossdk.io/store/types"
)

type MerkleTestSuite struct {
	testifysuite.Suite

	store     *rootmulti.Store
	storeKey  *storetypes.KVStoreKey
	iavlStore *iavl.Store
}

func (s *MerkleTestSuite) SetupTest() {
	db := dbm.NewMemDB()
	s.store = rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())

	s.storeKey = storetypes.NewKVStoreKey("iavlStoreKey")

	s.store.MountStoreWithDB(s.storeKey, storetypes.StoreTypeIAVL, nil)
	err := s.store.LoadVersion(0)
	s.Require().NoError(err)

	var ok bool
	s.iavlStore, ok = s.store.GetCommitStore(s.storeKey).(*iavl.Store)
	s.Require().True(ok)
}

func TestMerkleTestSuite(t *testing.T) {
	testifysuite.Run(t, new(MerkleTestSuite))
}
