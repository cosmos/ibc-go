package keeper_test

import (
	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/keeper"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
)

func (suite *KeeperTestSuite) TestAssertChannelCapabilityMigrations() {
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"capability not found",
			func() {
				suite.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(
					suite.chainA.GetContext(),
					ibctesting.FirstConnectionID,
					ibctesting.MockPort,
					ibctesting.FirstChannelID,
				)
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path := NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICAPath(path, ibctesting.TestAccAddress)
			suite.Require().NoError(err)

			tc.malleate()

			migrator := keeper.NewMigrator(&suite.chainA.GetSimApp().ICAControllerKeeper)
			err = migrator.AssertChannelCapabilityMigrations(suite.chainA.GetContext())

			if tc.expPass {
				suite.Require().NoError(err)

				isMiddlewareEnabled := suite.chainA.GetSimApp().ICAControllerKeeper.IsMiddlewareEnabled(
					suite.chainA.GetContext(),
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ConnectionID,
				)

				suite.Require().True(isMiddlewareEnabled)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
