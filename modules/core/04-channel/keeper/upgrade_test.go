package keeper_test

import (
	"fmt"

	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/mock"
)

func (suite *KeeperTestSuite) TestChanUpgradeInit() {
	var (
		path        *ibctesting.Path
		expSequence uint64
		expVersion  string
		upgrade     *types.Upgrade
	)

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
			"success with later upgrade sequence",
			func() {
				channel := path.EndpointA.GetChannel()
				channel.UpgradeSequence = 4
				path.EndpointA.SetChannel(channel)
				expSequence = 5
			},
			true,
		},
		{
			"success with alternative previous version",
			func() {
				expVersion = "mock-v1.1"
				channel := path.EndpointA.GetChannel()
				channel.Version = expVersion

				path.EndpointA.SetChannel(channel)
			},
			true,
		},
		{
			"identical upgrade channel end",
			func() {
				channel := path.EndpointA.GetChannel()
				upgrade = types.NewUpgrade(
					types.NewUpgradeFields(
						channel.Ordering, channel.ConnectionHops, channel.Version,
					),
					types.NewUpgradeTimeout(path.EndpointB.Chain.GetTimeoutHeight(), 0),
					0,
				)
			},
			false,
		},
		{
			"channel not found",
			func() {
				path.EndpointA.ChannelID = "invalid-channel"
				path.EndpointA.ChannelConfig.PortID = "invalid-port"
			},
			false,
		},
		{
			"channel state is not in OPEN state",
			func() {
				suite.Require().NoError(path.EndpointA.SetChannelState(types.CLOSED))
			},
			false,
		},
		{
			"proposed channel connection not found",
			func() {
				upgrade.Fields.ConnectionHops = []string{"connection-100"}
			},
			false,
		},
		{
			"invalid proposed channel connection state",
			func() {
				connectionEnd := path.EndpointA.GetConnection()
				connectionEnd.State = connectiontypes.UNINITIALIZED

				suite.chainA.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chainA.GetContext(), "connection-100", connectionEnd)
				upgrade.Fields.ConnectionHops = []string{"connection-100"}
			},
			false,
		},
		{
			"invalid proposed channel upgrade ordering",
			func() {
				upgrade.Fields.Ordering = types.ORDERED
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			expSequence = 1
			expVersion = mock.Version

			upgrade = types.NewUpgrade(
				types.NewUpgradeFields(
					types.UNORDERED, []string{path.EndpointA.ConnectionID}, fmt.Sprintf("%s-v2", mock.Version),
				),
				types.NewUpgradeTimeout(path.EndpointB.Chain.GetTimeoutHeight(), 0),
				0,
			)

			tc.malleate()

			sequence, previousVersion, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, *upgrade,
			)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expSequence, sequence)
				suite.Require().Equal(expVersion, previousVersion)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestChanUpgradeTry() {
	var (
		path        *ibctesting.Path
		expSequence uint64
		expVersion  string
		counterpartyUpgrade     *types.Upgrade
		proposedUpgrade     *types.Upgrade
	)

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
		// {
		// 	"success with later upgrade sequence",
		// 	func() {
		// 		channel := path.EndpointA.GetChannel()
		// 		channel.UpgradeSequence = 4
		// 		path.EndpointA.SetChannel(channel)
		// 		expSequence = 5
		// 	},
		// 	true,
		// },
		// {
		// 	"success with alternative previous version",
		// 	func() {
		// 		expVersion = "mock-v1.1"
		// 		channel := path.EndpointA.GetChannel()
		// 		channel.Version = expVersion

		// 		path.EndpointA.SetChannel(channel)
		// 	},
		// 	true,
		// },
		// {
		// 	"identical upgrade channel end",
		// 	func() {
		// 		channel := path.EndpointA.GetChannel()
		// 		upgrade = types.NewUpgrade(
		// 			types.NewModifiableUpgradeFields(
		// 				channel.Ordering, channel.ConnectionHops, channel.Version,
		// 			),
		// 			types.NewUpgradeTimeout(path.EndpointB.Chain.GetTimeoutHeight(), 0),
		// 			0,
		// 		)
		// 	},
		// 	false,
		// },
		// {
		// 	"channel not found",
		// 	func() {
		// 		path.EndpointA.ChannelID = "invalid-channel"
		// 		path.EndpointA.ChannelConfig.PortID = "invalid-port"
		// 	},
		// 	false,
		// },
		// {
		// 	"channel state is not in OPEN state",
		// 	func() {
		// 		suite.Require().NoError(path.EndpointA.SetChannelState(types.CLOSED))
		// 	},
		// 	false,
		// },
		// {
		// 	"proposed channel connection not found",
		// 	func() {
		// 		upgrade.UpgradeFields.ConnectionHops = []string{"connection-100"}
		// 	},
		// 	false,
		// },
		// {
		// 	"invalid proposed channel connection state",
		// 	func() {
		// 		connectionEnd := path.EndpointA.GetConnection()
		// 		connectionEnd.State = connectiontypes.UNINITIALIZED

		// 		suite.chainA.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chainA.GetContext(), "connection-100", connectionEnd)
		// 		upgrade.UpgradeFields.ConnectionHops = []string{"connection-100"}
		// 	},
		// 	false,
		// },
		// {
		// 	"invalid proposed channel upgrade ordering",
		// 	func() {
		// 		upgrade.UpgradeFields.Ordering = types.ORDERED
		// 	},
		// 	false,
		// },
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			expSequence = 1
			expVersion = mock.Version

			counterpartyUpgrade = types.NewUpgrade(
				types.NewUpgradeFields(
					types.UNORDERED, []string{path.EndpointA.ConnectionID}, fmt.Sprintf("%s-v2", mock.Version),
				),
				types.NewUpgradeTimeout(path.EndpointB.Chain.GetTimeoutHeight(), 0),
				0,
			)

			tc.malleate()

			counterpartySequence, previousVersion, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, *counterpartyUpgrade,
			)

			sequence, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeTry(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, *proposedUpgrade,
				*counterpartyUpgrade, counterpartySequence, proofCounterpartyChannel, proofUpgrade, proofHeight)

			

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expSequence, sequence)
				suite.Require().Equal(expVersion, previousVersion)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}