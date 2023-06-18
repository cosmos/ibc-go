package keeper_test

import (
	"fmt"

	icacontrollerkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestAssertChannelCapabilityMigrations() {
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
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), portIDWithOutPrefix, ibctesting.FirstChannelID, channeltypes.Channel{
					ConnectionHops: []string{ibctesting.FirstConnectionID},
				})
			},
			true,
		},
		{
			"capability not found",
			func() {
				portIDWithPrefix := fmt.Sprintf("%s%s", icatypes.ControllerPortPrefix, "port-without-capability")
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), portIDWithPrefix, ibctesting.FirstChannelID, channeltypes.Channel{
					ConnectionHops: []string{ibctesting.FirstConnectionID},
				})
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path := NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := SetupICAPath(path, ibctesting.TestAccAddress)
			s.Require().NoError(err)

			tc.malleate()

			migrator := icacontrollerkeeper.NewMigrator(&s.chainA.GetSimApp().ICAControllerKeeper)
			err = migrator.AssertChannelCapabilityMigrations(s.chainA.GetContext())

			if tc.expPass {
				s.Require().NoError(err)

				isMiddlewareEnabled := s.chainA.GetSimApp().ICAControllerKeeper.IsMiddlewareEnabled(
					s.chainA.GetContext(),
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ConnectionID,
				)

				s.Require().True(isMiddlewareEnabled)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMigratorMigrateParams() {
	testCases := []struct {
		msg            string
		malleate       func()
		expectedParams icacontrollertypes.Params
	}{
		{
			"success: default params",
			func() {
				params := icacontrollertypes.DefaultParams()
				subspace := s.chainA.GetSimApp().GetSubspace(icacontrollertypes.SubModuleName) // get subspace
				subspace.SetParamSet(s.chainA.GetContext(), &params)                           // set params
			},
			icacontrollertypes.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate() // explicitly set params

			migrator := icacontrollerkeeper.NewMigrator(&s.chainA.GetSimApp().ICAControllerKeeper)
			err := migrator.MigrateParams(s.chainA.GetContext())
			s.Require().NoError(err)

			params := s.chainA.GetSimApp().ICAControllerKeeper.GetParams(s.chainA.GetContext())
			s.Require().Equal(tc.expectedParams, params)
		})
	}
}
