package keeper_test

import (
	"context"
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	upgradekeeper "cosmossdk.io/x/upgrade/keeper"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v9/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	// TODO: remove
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

// MockStakingKeeper implements clienttypes.StakingKeeper used in ibckeeper.NewKeeper
type MockStakingKeeper struct {
	mockField string
}

func (MockStakingKeeper) GetHistoricalInfo(_ context.Context, _ int64) (stakingtypes.HistoricalInfo, error) {
	return stakingtypes.HistoricalInfo{}, nil
}

func (MockStakingKeeper) UnbondingTime(_ context.Context) (time.Duration, error) {
	return 0, nil
}

// Test ibckeeper.NewKeeper used to initialize IBCKeeper when creating an app instance.
// It verifies if ibckeeper.NewKeeper panic when any of the keepers passed in is empty.
func (suite *KeeperTestSuite) TestNewKeeper() {
	var (
		consensusHost  clienttypes.ConsensusHost
		upgradeKeeper  clienttypes.UpgradeKeeper
		scopedKeeper   capabilitykeeper.ScopedKeeper
		newIBCKeeperFn func()
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{"failure: empty consensus host value", func() {
			consensusHost = &ibctm.ConsensusHost{}
		}, false},
		{"failure: nil consensus host value", func() {
			consensusHost = nil
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
		{"failure: empty authority", func() {
			newIBCKeeperFn = func() {
				ibckeeper.NewKeeper(
					suite.chainA.GetSimApp().AppCodec(),
					suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey),
					suite.chainA.GetSimApp().GetSubspace(ibcexported.ModuleName),
					consensusHost,
					upgradeKeeper,
					scopedKeeper,
					"", // authority
				)
			}
		}, false},
		{"success: replace stakingKeeper with non-empty MockStakingKeeper", func() {
			// use a different implementation of clienttypes.StakingKeeper
			mockStakingKeeper := MockStakingKeeper{"not empty"}
			consensusHost = ibctm.NewConsensusHost(mockStakingKeeper)
		}, true},
	}

	for _, tc := range testCases {
		tc := tc
		suite.SetupTest()

		suite.Run(tc.name, func() {
			// set default behaviour
			newIBCKeeperFn = func() {
				ibckeeper.NewKeeper(
					suite.chainA.GetSimApp().AppCodec(),
					suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey),
					suite.chainA.GetSimApp().GetSubspace(ibcexported.ModuleName),
					consensusHost,
					upgradeKeeper,
					scopedKeeper,
					suite.chainA.App.GetIBCKeeper().GetAuthority(),
				)
			}

			consensusHost = ibctm.NewConsensusHost(suite.chainA.GetSimApp().StakingKeeper)
			upgradeKeeper = suite.chainA.GetSimApp().UpgradeKeeper
			scopedKeeper = suite.chainA.GetSimApp().ScopedIBCKeeper

			tc.malleate()

			if tc.expPass {
				suite.Require().NotPanics(
					newIBCKeeperFn,
				)
			} else {
				suite.Require().Panics(
					newIBCKeeperFn,
				)
			}
		})
	}
}
