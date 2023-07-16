package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/cosmos/ibc-go/modules/capability"
	"github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/cosmos/ibc-go/modules/capability/types"
)

var (
	stakingModuleName string = "staking"
	bankModuleName    string = "bank"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx    sdk.Context
	keeper *keeper.Keeper
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	s.ctx = testCtx.Ctx
	encCfg := moduletestutil.MakeTestEncodingConfig(capability.AppModuleBasic{})
	s.keeper = keeper.NewKeeper(encCfg.Codec, key, key)
}

func (s *KeeperTestSuite) TestSeal() {
	sk := s.keeper.ScopeToModule(bankModuleName)
	s.Require().Panics(func() {
		s.keeper.ScopeToModule("  ")
	})

	caps := make([]*types.Capability, 5)
	// Get Latest Index before creating new ones to sychronize indices correctly
	prevIndex := s.keeper.GetLatestIndex(s.ctx)

	for i := range caps {
		cap, err := sk.NewCapability(s.ctx, fmt.Sprintf("transfer-%d", i))
		s.Require().NoError(err)
		s.Require().NotNil(cap)
		s.Require().Equal(uint64(i)+prevIndex, cap.GetIndex())

		caps[i] = cap
	}

	s.Require().NotPanics(func() {
		s.keeper.Seal()
	})

	for i, cap := range caps {
		got, ok := sk.GetCapability(s.ctx, fmt.Sprintf("transfer-%d", i))
		s.Require().True(ok)
		s.Require().Equal(cap, got)
		s.Require().Equal(uint64(i)+prevIndex, got.GetIndex())
	}

	s.Require().Panics(func() {
		s.keeper.Seal()
	})

	s.Require().Panics(func() {
		_ = s.keeper.ScopeToModule(stakingModuleName)
	})
}

func (s *KeeperTestSuite) TestNewCapability() {
	sk := s.keeper.ScopeToModule(bankModuleName)

	got, ok := sk.GetCapability(s.ctx, "transfer")
	s.Require().False(ok)
	s.Require().Nil(got)

	cap, err := sk.NewCapability(s.ctx, "transfer")
	s.Require().NoError(err)
	s.Require().NotNil(cap)

	got, ok = sk.GetCapability(s.ctx, "transfer")
	s.Require().True(ok)
	s.Require().Equal(cap, got)
	s.Require().True(cap == got, "expected memory addresses to be equal")

	got, ok = sk.GetCapability(s.ctx, "invalid")
	s.Require().False(ok)
	s.Require().Nil(got)

	got, ok = sk.GetCapability(s.ctx, "transfer")
	s.Require().True(ok)
	s.Require().Equal(cap, got)
	s.Require().True(cap == got, "expected memory addresses to be equal")

	cap2, err := sk.NewCapability(s.ctx, "transfer")
	s.Require().Error(err)
	s.Require().Nil(cap2)

	got, ok = sk.GetCapability(s.ctx, "transfer")
	s.Require().True(ok)
	s.Require().Equal(cap, got)
	s.Require().True(cap == got, "expected memory addresses to be equal")

	cap, err = sk.NewCapability(s.ctx, "   ")
	s.Require().Error(err)
	s.Require().Nil(cap)
}

func (s *KeeperTestSuite) TestAuthenticateCapability() {
	sk1 := s.keeper.ScopeToModule(bankModuleName)
	sk2 := s.keeper.ScopeToModule(stakingModuleName)

	cap1, err := sk1.NewCapability(s.ctx, "transfer")
	s.Require().NoError(err)
	s.Require().NotNil(cap1)

	forgedCap := types.NewCapability(cap1.Index) // index should be the same index as the first capability
	s.Require().False(sk1.AuthenticateCapability(s.ctx, forgedCap, "transfer"))
	s.Require().False(sk2.AuthenticateCapability(s.ctx, forgedCap, "transfer"))

	cap2, err := sk2.NewCapability(s.ctx, "bond")
	s.Require().NoError(err)
	s.Require().NotNil(cap2)

	got, ok := sk1.GetCapability(s.ctx, "transfer")
	s.Require().True(ok)

	s.Require().True(sk1.AuthenticateCapability(s.ctx, cap1, "transfer"))
	s.Require().True(sk1.AuthenticateCapability(s.ctx, got, "transfer"))
	s.Require().False(sk1.AuthenticateCapability(s.ctx, cap1, "invalid"))
	s.Require().False(sk1.AuthenticateCapability(s.ctx, cap2, "transfer"))

	s.Require().True(sk2.AuthenticateCapability(s.ctx, cap2, "bond"))
	s.Require().False(sk2.AuthenticateCapability(s.ctx, cap2, "invalid"))
	s.Require().False(sk2.AuthenticateCapability(s.ctx, cap1, "bond"))

	err = sk2.ReleaseCapability(s.ctx, cap2)
	s.Require().NoError(err)
	s.Require().False(sk2.AuthenticateCapability(s.ctx, cap2, "bond"))

	badCap := types.NewCapability(100)
	s.Require().False(sk1.AuthenticateCapability(s.ctx, badCap, "transfer"))
	s.Require().False(sk2.AuthenticateCapability(s.ctx, badCap, "bond"))

	s.Require().False(sk1.AuthenticateCapability(s.ctx, cap1, "  "))
	s.Require().False(sk1.AuthenticateCapability(s.ctx, nil, "transfer"))
}

func (s *KeeperTestSuite) TestClaimCapability() {
	sk1 := s.keeper.ScopeToModule(bankModuleName)
	sk2 := s.keeper.ScopeToModule(stakingModuleName)
	sk3 := s.keeper.ScopeToModule("foo")

	cap, err := sk1.NewCapability(s.ctx, "transfer")
	s.Require().NoError(err)
	s.Require().NotNil(cap)

	s.Require().Error(sk1.ClaimCapability(s.ctx, cap, "transfer"))
	s.Require().NoError(sk2.ClaimCapability(s.ctx, cap, "transfer"))

	got, ok := sk1.GetCapability(s.ctx, "transfer")
	s.Require().True(ok)
	s.Require().Equal(cap, got)

	got, ok = sk2.GetCapability(s.ctx, "transfer")
	s.Require().True(ok)
	s.Require().Equal(cap, got)

	s.Require().Error(sk3.ClaimCapability(s.ctx, cap, "  "))
	s.Require().Error(sk3.ClaimCapability(s.ctx, nil, "transfer"))
}

func (s *KeeperTestSuite) TestGetOwners() {
	sk1 := s.keeper.ScopeToModule(bankModuleName)
	sk2 := s.keeper.ScopeToModule(stakingModuleName)
	sk3 := s.keeper.ScopeToModule("foo")

	sks := []keeper.ScopedKeeper{sk1, sk2, sk3}

	cap, err := sk1.NewCapability(s.ctx, "transfer")
	s.Require().NoError(err)
	s.Require().NotNil(cap)

	s.Require().NoError(sk2.ClaimCapability(s.ctx, cap, "transfer"))
	s.Require().NoError(sk3.ClaimCapability(s.ctx, cap, "transfer"))

	expectedOrder := []string{bankModuleName, "foo", stakingModuleName}
	// Ensure all scoped keepers can get owners
	for _, sk := range sks {
		owners, ok := sk.GetOwners(s.ctx, "transfer")
		mods, gotCap, err := sk.LookupModules(s.ctx, "transfer")

		s.Require().True(ok, "could not retrieve owners")
		s.Require().NotNil(owners, "owners is nil")

		s.Require().NoError(err, "could not retrieve modules")
		s.Require().NotNil(gotCap, "capability is nil")
		s.Require().NotNil(mods, "modules is nil")
		s.Require().Equal(cap, gotCap, "caps not equal")

		s.Require().Equal(len(expectedOrder), len(owners.Owners), "length of owners is unexpected")
		for i, o := range owners.Owners {
			// Require owner is in expected position
			s.Require().Equal(expectedOrder[i], o.Module, "module is unexpected")
			s.Require().Equal(expectedOrder[i], mods[i], "module in lookup is unexpected")
		}
	}

	// foo module releases capability
	err = sk3.ReleaseCapability(s.ctx, cap)
	s.Require().Nil(err, "could not release capability")

	// new expected order and scoped capabilities
	expectedOrder = []string{bankModuleName, stakingModuleName}
	sks = []keeper.ScopedKeeper{sk1, sk2}

	// Ensure all scoped keepers can get owners
	for _, sk := range sks {
		owners, ok := sk.GetOwners(s.ctx, "transfer")
		mods, cap, err := sk.LookupModules(s.ctx, "transfer")

		s.Require().True(ok, "could not retrieve owners")
		s.Require().NotNil(owners, "owners is nil")

		s.Require().NoError(err, "could not retrieve modules")
		s.Require().NotNil(cap, "capability is nil")
		s.Require().NotNil(mods, "modules is nil")

		s.Require().Equal(len(expectedOrder), len(owners.Owners), "length of owners is unexpected")
		for i, o := range owners.Owners {
			// Require owner is in expected position
			s.Require().Equal(expectedOrder[i], o.Module, "module is unexpected")
			s.Require().Equal(expectedOrder[i], mods[i], "module in lookup is unexpected")
		}
	}

	_, ok := sk1.GetOwners(s.ctx, "  ")
	s.Require().False(ok, "got owners from empty capability name")
}

func (s *KeeperTestSuite) TestReleaseCapability() {
	sk1 := s.keeper.ScopeToModule(bankModuleName)
	sk2 := s.keeper.ScopeToModule(stakingModuleName)

	cap1, err := sk1.NewCapability(s.ctx, "transfer")
	s.Require().NoError(err)
	s.Require().NotNil(cap1)

	s.Require().NoError(sk2.ClaimCapability(s.ctx, cap1, "transfer"))

	cap2, err := sk2.NewCapability(s.ctx, "bond")
	s.Require().NoError(err)
	s.Require().NotNil(cap2)

	s.Require().Error(sk1.ReleaseCapability(s.ctx, cap2))

	s.Require().NoError(sk2.ReleaseCapability(s.ctx, cap1))
	got, ok := sk2.GetCapability(s.ctx, "transfer")
	s.Require().False(ok)
	s.Require().Nil(got)

	s.Require().NoError(sk1.ReleaseCapability(s.ctx, cap1))
	got, ok = sk1.GetCapability(s.ctx, "transfer")
	s.Require().False(ok)
	s.Require().Nil(got)

	s.Require().Error(sk1.ReleaseCapability(s.ctx, nil))
}

func (s *KeeperTestSuite) TestRevertCapability() {
	sk := s.keeper.ScopeToModule(bankModuleName)

	ms := s.ctx.MultiStore()

	msCache := ms.CacheMultiStore()
	cacheCtx := s.ctx.WithMultiStore(msCache)

	capName := "revert"
	// Create capability on cached context
	cap, err := sk.NewCapability(cacheCtx, capName)
	s.Require().NoError(err, "could not create capability")

	// Check that capability written in cached context
	gotCache, ok := sk.GetCapability(cacheCtx, capName)
	s.Require().True(ok, "could not retrieve capability from cached context")
	s.Require().Equal(cap, gotCache, "did not get correct capability from cached context")

	// Check that capability is NOT written to original context
	got, ok := sk.GetCapability(s.ctx, capName)
	s.Require().False(ok, "retrieved capability from original context before write")
	s.Require().Nil(got, "capability not nil in original store")

	// Write to underlying memKVStore
	msCache.Write()

	got, ok = sk.GetCapability(s.ctx, capName)
	s.Require().True(ok, "could not retrieve capability from context")
	s.Require().Equal(cap, got, "did not get correct capability from context")
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
