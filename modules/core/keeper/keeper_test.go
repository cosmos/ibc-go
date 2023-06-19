package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))

	// TODO: remove
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	s.coordinator.CommitNBlocks(s.chainA, 2)
	s.coordinator.CommitNBlocks(s.chainB, 2)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// MockStakingKeeper implements clienttypes.StakingKeeper used in ibckeeper.NewKeeper
type MockStakingKeeper struct {
	mockField string
}

func (MockStakingKeeper) GetHistoricalInfo(ctx sdk.Context, height int64) (stakingtypes.HistoricalInfo, bool) {
	return stakingtypes.HistoricalInfo{}, true
}

func (MockStakingKeeper) UnbondingTime(ctx sdk.Context) time.Duration {
	return 0
}

// Test ibckeeper.NewKeeper used to initialize IBCKeeper when creating an app instance.
// It verifies if ibckeeper.NewKeeper panic when any of the keepers passed in is empty.
func (s *KeeperTestSuite) TestNewKeeper() {
	var (
		stakingKeeper  clienttypes.StakingKeeper
		upgradeKeeper  clienttypes.UpgradeKeeper
		scopedKeeper   capabilitykeeper.ScopedKeeper
		newIBCKeeperFn = func() {
			ibckeeper.NewKeeper(
				s.chainA.GetSimApp().AppCodec(),
				s.chainA.GetSimApp().GetKey(ibcexported.StoreKey),
				s.chainA.GetSimApp().GetSubspace(ibcexported.ModuleName),
				stakingKeeper,
				upgradeKeeper,
				scopedKeeper,
				s.chainA.App.GetIBCKeeper().GetAuthority(),
			)
		}
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{"failure: empty staking keeper value", func() {
			emptyStakingKeeperValue := stakingkeeper.Keeper{}

			stakingKeeper = emptyStakingKeeperValue
		}, false},
		{"failure: empty staking keeper pointer", func() {
			emptyStakingKeeperPointer := &stakingkeeper.Keeper{}

			stakingKeeper = emptyStakingKeeperPointer
		}, false},
		{"failure: empty mock staking keeper", func() {
			// use a different implementation of clienttypes.StakingKeeper
			emptyMockStakingKeeper := MockStakingKeeper{}

			stakingKeeper = emptyMockStakingKeeper
		}, false},
		{"failure: empty upgrade keeper value", func() {
			emptyUpgradeKeeperValue := upgradekeeper.Keeper{}

			upgradeKeeper = emptyUpgradeKeeperValue
		}, false},
		{"failure: empty upgrade keeper pointer", func() {
			emptyUpgradeKeeperPointer := &upgradekeeper.Keeper{}

			upgradeKeeper = emptyUpgradeKeeperPointer
		}, false},
		{"failure: empty scoped keeper", func() {
			emptyScopedKeeper := capabilitykeeper.ScopedKeeper{}

			scopedKeeper = emptyScopedKeeper
		}, false},
		{"success: replace stakingKeeper with non-empty MockStakingKeeper", func() {
			// use a different implementation of clienttypes.StakingKeeper
			mockStakingKeeper := MockStakingKeeper{"not empty"}

			stakingKeeper = mockStakingKeeper
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		s.SetupTest()

		s.Run(tc.name, func() {
			stakingKeeper = s.chainA.GetSimApp().StakingKeeper
			upgradeKeeper = s.chainA.GetSimApp().UpgradeKeeper
			scopedKeeper = s.chainA.GetSimApp().ScopedIBCKeeper

			tc.malleate()

			if tc.expPass {
				s.Require().NotPanics(
					newIBCKeeperFn,
				)
			} else {
				s.Require().Panics(
					newIBCKeeperFn,
				)
			}
		})
	}
}
