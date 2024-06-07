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

func (suite *MerkleTestSuite) SetupTest() {
	db := dbm.NewMemDB()
	suite.store = rootmulti.NewStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())

	suite.storeKey = storetypes.NewKVStoreKey("iavlStoreKey")

	suite.store.MountStoreWithDB(suite.storeKey, storetypes.StoreTypeIAVL, nil)
	err := suite.store.LoadVersion(0)
	suite.Require().NoError(err)

	var ok bool
	suite.iavlStore, ok = suite.store.GetCommitStore(suite.storeKey).(*iavl.Store)
	suite.Require().True(ok)
}

func TestMerkleTestSuite(t *testing.T) {
	testifysuite.Run(t, new(MerkleTestSuite))
}
