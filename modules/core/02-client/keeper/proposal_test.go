package keeper_test

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (suite *KeeperTestSuite) TestHandleUpgradeProposal() {
	var (
		upgradedClientState *ibctm.ClientState
		oldPlan, plan       upgradetypes.Plan
		content             govtypes.Content
		err                 error
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid upgrade proposal", func() {
				content, err = types.NewUpgradeProposal(ibctesting.Title, ibctesting.Description, plan, upgradedClientState)
				suite.Require().NoError(err)
			}, true,
		},
		{
			"valid upgrade proposal with previous IBC state", func() {
				oldPlan = upgradetypes.Plan{
					Name:   "upgrade IBC clients",
					Height: 100,
				}

				content, err = types.NewUpgradeProposal(ibctesting.Title, ibctesting.Description, plan, upgradedClientState)
				suite.Require().NoError(err)
			}, true,
		},
		{
			"cannot unpack client state", func() {
				protoAny, err := types.PackConsensusState(&ibctm.ConsensusState{})
				suite.Require().NoError(err)
				content = &types.UpgradeProposal{
					Title:               ibctesting.Title,
					Description:         ibctesting.Description,
					Plan:                plan,
					UpgradedClientState: protoAny,
				}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()  // reset
			oldPlan.Height = 0 // reset

			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupClients(path)
			upgradedClientState = suite.chainA.GetClientState(path.EndpointA.ClientID).ZeroCustomFields().(*ibctm.ClientState)

			// use height 1000 to distinguish from old plan
			plan = upgradetypes.Plan{
				Name:   "upgrade IBC clients",
				Height: 1000,
			}

			tc.malleate()

			// set the old plan if it is not empty
			if oldPlan.Height != 0 {
				// set upgrade plan in the upgrade store
				store := suite.chainA.GetContext().KVStore(suite.chainA.GetSimApp().GetKey(upgradetypes.StoreKey))
				bz := suite.chainA.App.AppCodec().MustMarshal(&oldPlan)
				store.Set(upgradetypes.PlanKey(), bz)

				bz, err := types.MarshalClientState(suite.chainA.App.AppCodec(), upgradedClientState)
				suite.Require().NoError(err)

				suite.chainA.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainA.GetContext(), oldPlan.Height, bz) //nolint:errcheck
			}

			upgradeProp, ok := content.(*types.UpgradeProposal)
			suite.Require().True(ok)
			err = suite.chainA.App.GetIBCKeeper().ClientKeeper.HandleUpgradeProposal(suite.chainA.GetContext(), upgradeProp)

			if tc.expPass {
				suite.Require().NoError(err)

				// check that the correct plan is returned
				storedPlan, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(suite.chainA.GetContext())
				suite.Require().NoError(err)
				suite.Require().Equal(plan, storedPlan)

				// check that old upgraded client state is cleared
				_, err = suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(suite.chainA.GetContext(), oldPlan.Height)
				suite.Require().ErrorIs(err, upgradetypes.ErrNoUpgradedClientFound)

				// check that client state was set
				storedClientState, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(suite.chainA.GetContext(), plan.Height)
				suite.Require().NoError(err)
				clientState, err := types.UnmarshalClientState(suite.chainA.App.AppCodec(), storedClientState)
				suite.Require().NoError(err)
				suite.Require().Equal(upgradedClientState, clientState)
			} else {
				suite.Require().Error(err)

				// check that the new plan wasn't stored
				storedPlan, err := suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradePlan(suite.chainA.GetContext())
				if oldPlan.Height != 0 {
					// NOTE: this is only true if the ScheduleUpgrade function
					// returns an error before clearing the old plan
					suite.Require().NoError(err)
					suite.Require().Equal(oldPlan, storedPlan)
				} else {
					suite.Require().ErrorIs(err, upgradetypes.ErrNoUpgradePlanFound)
					suite.Require().Empty(storedPlan)
				}

				// check that client state was not set
				_, err = suite.chainA.GetSimApp().UpgradeKeeper.GetUpgradedClient(suite.chainA.GetContext(), plan.Height)
				suite.Require().ErrorIs(err, upgradetypes.ErrNoUpgradedClientFound)
			}
		})
	}
}
