package capability_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/ibc-go/modules/capability"
	"github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/cosmos/ibc-go/modules/capability/types"
)

const mockMemStoreKey = "memory:mock"

type CapabilityTestSuite struct {
	suite.Suite

	cdc codec.Codec
	ctx sdk.Context

	keeper *keeper.Keeper

	storeKey        *storetypes.KVStoreKey
	memStoreKey     *storetypes.MemoryStoreKey
	mockMemStoreKey *storetypes.MemoryStoreKey
}

func (s *CapabilityTestSuite) SetupTest() {
	encodingCfg := moduletestutil.MakeTestEncodingConfig(capability.AppModuleBasic{})
	s.cdc = encodingCfg.Codec

	s.storeKey = storetypes.NewKVStoreKey(types.StoreKey)
	s.memStoreKey = storetypes.NewMemoryStoreKey(types.MemStoreKey)
	s.mockMemStoreKey = storetypes.NewMemoryStoreKey(mockMemStoreKey)

	s.ctx = s.NewTestContext()
	s.keeper = keeper.NewKeeper(s.cdc, s.storeKey, s.memStoreKey)
}

func (suite *CapabilityTestSuite) NewTestContext() sdk.Context {
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(suite.storeKey, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(suite.memStoreKey, storetypes.StoreTypeMemory, db)
	cms.MountStoreWithDB(suite.mockMemStoreKey, storetypes.StoreTypeMemory, db)

	err := cms.LoadLatestVersion()
	suite.Require().NoError(err)

	return sdk.NewContext(cms, tmproto.Header{}, false, log.NewNopLogger())
}

// The following test case mocks a specific bug discovered in https://github.com/cosmos/cosmos-sdk/issues/9800
// and ensures that the current code successfully fixes the issue.
// This test emulates statesync by firstly populating persisted state by creating a new scoped keeper and capability.
// In-memory storage is then discarded by creating a new capability keeper and app module using a mock memstore key.
// BeginBlock is then called to populate the new in-memory store using the persisted state.
func (s *CapabilityTestSuite) TestInitializeMemStore() {
	// create a scoped keeper and instantiate a new capability to populate state
	scopedKeeper := s.keeper.ScopeToModule(banktypes.ModuleName)

	cap1, err := scopedKeeper.NewCapability(s.ctx, "transfer")
	s.Require().NoError(err)
	s.Require().NotNil(cap1)

	// mock statesync by creating a new keeper and module that shares persisted state
	// but discards in-memory map by using a mock memstore key
	newKeeper := keeper.NewKeeper(s.cdc, s.storeKey, s.mockMemStoreKey)
	newModule := capability.NewAppModule(s.cdc, *newKeeper, true)

	// reassign the scoped keeper, this will inherit the the mock memstore key used above
	scopedKeeper = newKeeper.ScopeToModule(banktypes.ModuleName)

	// seal the new keeper and ensure the in-memory store is not initialized
	newKeeper.Seal()
	s.Require().False(newKeeper.IsInitialized(s.ctx), "memstore initialized flag set before BeginBlock")

	cap1, ok := scopedKeeper.GetCapability(s.ctx, "transfer")
	s.Require().False(ok)
	s.Require().Nil(cap1)

	// add a new block gas meter to the context
	ctx := s.ctx.WithBlockGasMeter(storetypes.NewGasMeter(50))

	prevGas := ctx.GasMeter().GasConsumed()
	prevBlockGas := ctx.BlockGasMeter().GasConsumed()

	// call app module BeginBlock and ensure that no gas has been consumed
	newModule.BeginBlock(ctx)

	gasUsed := ctx.GasMeter().GasConsumed()
	blockGasUsed := ctx.BlockGasMeter().GasConsumed()

	s.Require().Equal(prevBlockGas, blockGasUsed, "ensure beginblocker consumed no block gas during execution")
	s.Require().Equal(prevGas, gasUsed, "ensure beginblocker consumed no gas during execution")

	// assert that the in-memory store is now initialized
	s.Require().True(newKeeper.IsInitialized(ctx), "memstore initialized flag not set")

	// ensure that BeginBlock has populated the new in-memory store (using the mock memstore key) and initialized capabilities
	cap1, ok = scopedKeeper.GetCapability(ctx, "transfer")
	s.Require().True(ok)
	s.Require().NotNil(cap1)

	// ensure capabilities do not get reinitialized on next BeginBlock by comparing capability pointers
	// and assert that the in-memory store is still initialized
	newModule.BeginBlock(ctx)
	refreshedCap, ok := scopedKeeper.GetCapability(ctx, "transfer")
	s.Require().True(ok)
	s.Require().Equal(cap1, refreshedCap, "capabilities got reinitialized after second BeginBlock")
	s.Require().True(newKeeper.IsInitialized(ctx), "memstore initialized flag not set")
}

func TestCapabilityTestSuite(t *testing.T) {
	suite.Run(t, new(CapabilityTestSuite))
}
