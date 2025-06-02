package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"

	"github.com/cosmos/cosmos-sdk/runtime"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
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
		expPanic string
	}{
		{
			name: "failure: empty upgrade keeper value",
			malleate: func() {
				emptyUpgradeKeeperValue := upgradekeeper.Keeper{}
				upgradeKeeper = emptyUpgradeKeeperValue
			},
			expPanic: "cannot initialize IBC keeper: empty upgrade keeper",
		},
		{
			name: "failure: empty upgrade keeper pointer",
			malleate: func() {
				emptyUpgradeKeeperPointer := &upgradekeeper.Keeper{}
				upgradeKeeper = emptyUpgradeKeeperPointer
			},
			expPanic: "cannot initialize IBC keeper: empty upgrade keeper",
		},
		{
			name: "failure: empty authority",
			malleate: func() {
				newIBCKeeperFn = func() {
					ibckeeper.NewKeeper(
						suite.chainA.GetSimApp().AppCodec(),
						runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(ibcexported.StoreKey)),
						suite.chainA.GetSimApp().GetSubspace(ibcexported.ModuleName),
						upgradeKeeper,
						"", // authority
					)
				}
			},
			expPanic: "authority cannot be empty",
		},
	}

	for _, tc := range testCases {

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

			if tc.expPanic != "" {
				suite.Require().Panics(func() {
					newIBCKeeperFn()
				}, "expected panic but no panic occurred")

				defer func() {
					if r := recover(); r != nil {
						suite.Require().Contains(r.(error).Error(), tc.expPanic, "unexpected panic message")
					}
				}()

			} else {
				suite.Require().NotPanics(newIBCKeeperFn, "unexpected panic occurred")
			}
		})
	}
}
