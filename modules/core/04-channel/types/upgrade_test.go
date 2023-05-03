package types_test

import (
	"fmt"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/mock"
)

func (suite *TypesTestSuite) TestUpgradeTimeout() {
	var upgrade *types.Upgrade

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
			},
			true,
		},
		{
			"invalid ordering",
			func() {
				upgrade.Fields.Ordering = types.NONE
			},
			false,
		},
		{
			"more than one connection hop",
			func() {
				upgrade.Fields.ConnectionHops = []string{"connection-0", "connection-1"}
			},
			false,
		},
		{
			"empty version",
			func() {
				upgrade.Fields.Version = ""
			},
			false,
		},
		{
			"invalid timeout",
			func() {
				upgrade.Timeout.Height = clienttypes.ZeroHeight()
				upgrade.Timeout.Timestamp = 0
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			upgrade = types.NewUpgrade(
				types.NewUpgradeFields(
					types.ORDERED,
					[]string{"connection-0"},
					mock.Version,
				),
				types.NewUpgradeTimeout(
					clienttypes.NewHeight(0, 100),
					0,
				),
				0,
			)

			tc.malleate()

			err := upgrade.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestHasPassed() {
	var (
		path                   *ibctesting.Path
		upgrade                *types.Upgrade
		proposedConnectionHops []string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: timeout has not passed",
			func() {},
			true,
		},
		{
			"fail: timeout height has passed",
			func() {
				upgrade.Timeout.Height = clienttypes.NewHeight(
					clienttypes.ParseChainID(path.EndpointA.Chain.GetContext().ChainID()),
					uint64(suite.chainA.GetContext().BlockHeight())-1,
				)
			},
			false,
		},
		{
			"fail: timeout timestamp has passed",
			func() {
				upgrade.Timeout.Height = clienttypes.ZeroHeight()
				upgrade.Timeout.Timestamp = uint64(suite.chainA.GetContext().BlockTime().UnixNano() - 1)
			},
			false,
		},
		{
			"fail: invalid upgrade timeout",
			func() {
				upgrade.Timeout.Height = clienttypes.ZeroHeight()
				upgrade.Timeout.Timestamp = 0
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTestCoordinator()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			proposedConnectionHops = []string{path.EndpointB.ConnectionID}

			upgrade = types.NewUpgrade(
				types.NewUpgradeFields(
					types.UNORDERED, proposedConnectionHops, fmt.Sprintf("%s-v2", mock.Version),
				),
				types.NewUpgradeTimeout(path.EndpointA.Chain.GetTimeoutHeight(), 0),
				0,
			)

			tc.malleate()

			passed, err := upgrade.Timeout.HasPassed(suite.chainA.GetContext())

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().False(passed)
			} else {
				suite.Require().Error(err)
				suite.Require().True(passed)
			}
		})
	}
}
