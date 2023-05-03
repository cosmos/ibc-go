package types_test

import (
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
		path           *ibctesting.Path
		upgradeTimeout types.UpgradeTimeout
		passInfo       string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"timeout has not passed",
			func() {},
			false,
		},
		{
			"timeout height has passed",
			func() {
				upgradeTimeout.Height = clienttypes.NewHeight(
					clienttypes.ParseChainID(path.EndpointA.Chain.GetContext().ChainID()),
					uint64(suite.chainA.GetContext().BlockHeight())-1,
				)
				passInfo = "upgrade timeout has passed at block height 1-17, timeout height 1-16"
			},
			true,
		},
		{
			"timeout timestamp has passed",
			func() {
				upgradeTimeout.Height = clienttypes.ZeroHeight()
				upgradeTimeout.Timestamp = uint64(suite.chainA.GetContext().BlockTime().UnixNano() - 1)
				passInfo = "upgrade timeout has passed at block timestamp 1577923350000000000, timeout timestamp 1577923349999999999"
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTestCoordinator()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			upgradeTimeout = types.NewUpgradeTimeout(path.EndpointA.Chain.GetTimeoutHeight(), 0)

			tc.malleate()

			passed, info := upgradeTimeout.HasPassed(suite.chainA.GetContext())

			if tc.expPass {
				suite.Require().True(passed)
				suite.Require().Equal(passInfo, info)

			} else {
				suite.Require().False(passed)
				suite.Require().Equal("upgrade timeout has not passed", info)
			}
		})
	}
}
