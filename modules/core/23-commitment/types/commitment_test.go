package types_test

import (
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/store/iavl"
	"github.com/cosmos/cosmos-sdk/store/rootmulti"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/stretchr/testify/suite"
)

type MerkleTestSuite struct {
	suite.Suite

	store     *rootmulti.Store
	storeKey  *storetypes.KVStoreKey
	iavlStore *iavl.Store
}

func (s *MerkleTestSuite) SetupTest() {
	db := dbm.NewMemDB()
	dblog := log.TestingLogger()
	s.store = rootmulti.NewStore(db, dblog)

	s.storeKey = storetypes.NewKVStoreKey("iavlStoreKey")

	s.store.MountStoreWithDB(s.storeKey, storetypes.StoreTypeIAVL, nil)
	err := s.store.LoadVersion(0)
	s.Require().NoError(err)

	s.iavlStore = s.store.GetCommitStore(s.storeKey).(*iavl.Store)
}

func TestMerkleTestSuite(t *testing.T) {
	suite.Run(t, new(MerkleTestSuite))
}
