package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	upgradekeeper "cosmossdk.io/x/upgrade/keeper"

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
	testifysuite.Run(t, new(KeeperTestSuite))
}

// Test ibckeeper.NewKeeper used to initialize IBCKeeper when creating an app instance.
// It verifies if ibckeeper.NewKeeper panic when any of the keepers passed in is empty.
func (s *KeeperTestSuite) TestNewKeeper() {
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
						s.chainA.GetSimApp().AppCodec(),
						runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ibcexported.StoreKey)),
						upgradeKeeper,
						"", // authority
					)
				}
			},
			expPanic: "authority cannot be empty",
		},
	}

	for _, tc := range testCases {
		s.SetupTest()

		s.Run(tc.name, func() {
			// set default behaviour
			newIBCKeeperFn = func() {
				ibckeeper.NewKeeper(
					s.chainA.GetSimApp().AppCodec(),
					runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ibcexported.StoreKey)),
					upgradeKeeper,
					s.chainA.App.GetIBCKeeper().GetAuthority(),
				)
			}

			upgradeKeeper = s.chainA.GetSimApp().UpgradeKeeper

			tc.malleate()

			if tc.expPanic != "" {
				s.Require().Panics(func() {
					newIBCKeeperFn()
				}, "expected panic but no panic occurred")

				defer func() {
					if r := recover(); r != nil {
						s.Require().Contains(r.(error).Error(), tc.expPanic, "unexpected panic message")
					}
				}()
			} else {
				s.Require().NotPanics(newIBCKeeperFn, "unexpected panic occurred")
			}
		})
	}
}
