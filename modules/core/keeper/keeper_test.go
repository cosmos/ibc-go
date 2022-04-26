package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	"github.com/stretchr/testify/suite"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	ibchost "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v3/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
	"github.com/cosmos/ibc-go/v3/testing/simapp"
)

type KeeperTestSuite struct {
	suite.Suite

	app           *simapp.SimApp
	stakingKeeper clienttypes.StakingKeeper
	upgradeKeeper clienttypes.UpgradeKeeper
	scopedKeeper  capabilitykeeper.ScopedKeeper
}

func (suite *KeeperTestSuite) SetupTest() {
	coordinator := ibctesting.NewCoordinator(suite.T(), 1)
	chainA := coordinator.GetChain(ibctesting.GetChainID(1))

	suite.app = chainA.App.(*simapp.SimApp)

	suite.stakingKeeper = suite.app.StakingKeeper
	suite.upgradeKeeper = suite.app.UpgradeKeeper
	suite.scopedKeeper = suite.app.ScopedIBCKeeper
}

func (suite *KeeperTestSuite) NewIBCKeeper() {
	ibckeeper.NewKeeper(
		suite.app.AppCodec(),
		suite.app.GetKey(ibchost.StoreKey),
		suite.app.GetSubspace(ibchost.ModuleName),
		suite.stakingKeeper,
		suite.upgradeKeeper,
		suite.scopedKeeper,
	)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// MockStakingKeeper implements clienttypes.StakingKeeper used in ibckeeper.NewKeeper
type MockStakingKeeper struct {
	mockField string
}

func (d MockStakingKeeper) GetHistoricalInfo(ctx sdk.Context, height int64) (stakingtypes.HistoricalInfo, bool) {
	return stakingtypes.HistoricalInfo{}, true
}

func (d MockStakingKeeper) UnbondingTime(ctx sdk.Context) time.Duration {
	return 0
}

// Test ibckeeper.NewKeeper used to initialize IBCKeeper when creating an app instance.
// It verifies if ibckeeper.NewKeeper panic when any of the keepers passed in is empty.
func (suite *KeeperTestSuite) TestNewKeeper() {

	testCases := []struct {
		name     string
		malleate func()
	}{
		{"failure: empty staking keeper", func() {
			emptyStakingKeeper := stakingkeeper.Keeper{}

			suite.stakingKeeper = emptyStakingKeeper

			suite.Require().Panics(suite.NewIBCKeeper)
		}},
		{"failure: empty dummy staking keeper", func() {
			// use a different implementation of clienttypes.StakingKeeper
			emptyMockStakingKeeper := MockStakingKeeper{}

			suite.stakingKeeper = emptyMockStakingKeeper

			suite.Require().Panics(suite.NewIBCKeeper)
		}},
		{"failure: empty upgrade keeper", func() {
			emptyUpgradeKeeper := upgradekeeper.Keeper{}

			suite.upgradeKeeper = emptyUpgradeKeeper

			suite.Require().Panics(suite.NewIBCKeeper)
		}},
		{"failure: empty scoped keeper", func() {
			emptyScopedKeeper := capabilitykeeper.ScopedKeeper{}

			suite.scopedKeeper = emptyScopedKeeper

			suite.Require().Panics(suite.NewIBCKeeper)
		}},
		{"success: replace stakingKeeper with non-empty MockStakingKeeper", func() {
			// use a different implementation of clienttypes.StakingKeeper
			mockStakingKeeper := MockStakingKeeper{"not empty"}

			suite.stakingKeeper = mockStakingKeeper

			suite.Require().NotPanics(suite.NewIBCKeeper)
		}},
	}

	for _, tc := range testCases {
		tc := tc
		suite.SetupTest()

		suite.Run(tc.name, func() {
			tc.malleate()
		})
	}
}
