package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	upgradekeeper "cosmossdk.io/x/upgrade/keeper"

	"github.com/cosmos/cosmos-sdk/runtime"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v9/modules/core/keeper"
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

// Test ibckeeper.NewKeeper used to initialize IBCKeeper when creating an app instance.
// It verifies if ibckeeper.NewKeeper panic when any of the keepers passed in is empty.
func (suite *KeeperTestSuite) TestNewKeeper() {
	var (
		upgradeKeeper  clienttypes.UpgradeKeeper
		newIBCKeeperFn func()
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{"failure: empty upgrade keeper value", func() {
			emptyUpgradeKeeperValue := upgradekeeper.Keeper{}

			upgradeKeeper = emptyUpgradeKeeperValue
		}, false},
		{"failure: empty upgrade keeper pointer", func() {
			emptyUpgradeKeeperPointer := &upgradekeeper.Keeper{}

			upgradeKeeper = emptyUpgradeKeeperPointer
		}, false},
		{"failure: empty authority", func() {
			newIBCKeeperFn = func() {
				ibckeeper.NewKeeper(
					suite.chainA.GetSimApp().AppCodec(),
					runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey)),
					suite.chainA.GetSimApp().GetSubspace(ibcexported.ModuleName),
					upgradeKeeper,
					"", // authority
				)
			}
		}, false},
	}

	for _, tc := range testCases {
		tc := tc
		suite.SetupTest()

		suite.Run(tc.name, func() {
			// set default behaviour
			newIBCKeeperFn = func() {
				ibckeeper.NewKeeper(
					suite.chainA.GetSimApp().AppCodec(),
					runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey)),
					suite.chainA.GetSimApp().GetSubspace(ibcexported.ModuleName),
					upgradeKeeper,
					suite.chainA.App.GetIBCKeeper().GetAuthority(),
				)
			}

			upgradeKeeper = suite.chainA.GetSimApp().UpgradeKeeper

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
