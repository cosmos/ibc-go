package keeper_test

import (
	"fmt"

	"github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/keeper"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
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
			"channel with different port is filtered out",
			func() {
				portIDWithOutPrefix := ibctesting.MockPort
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(suite.chainA.GetContext(), portIDWithOutPrefix, ibctesting.FirstChannelID, channeltypes.Channel{
					ConnectionHops: []string{ibctesting.FirstConnectionID},
				})
			},
			true,
		},
		{
			"capability not found",
			func() {
				portIDWithPrefix := fmt.Sprintf("%s%s", icatypes.ControllerPortPrefix, "port-without-capability")
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(suite.chainA.GetContext(), portIDWithPrefix, ibctesting.FirstChannelID, channeltypes.Channel{
					ConnectionHops: []string{ibctesting.FirstConnectionID},
				})
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
