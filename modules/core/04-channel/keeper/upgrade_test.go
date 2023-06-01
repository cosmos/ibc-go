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
		upgrade     types.Upgrade
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
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			expSequence = 1

			upgrade = types.NewUpgrade(
				types.NewUpgradeFields(
					types.UNORDERED, []string{path.EndpointA.ConnectionID}, fmt.Sprintf("%s-v2", mock.Version),
				),
				types.NewUpgradeTimeout(path.EndpointB.Chain.GetTimeoutHeight(), 0),
				0,
			)

			tc.malleate()

			proposedUpgrade, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgrade.Fields, upgrade.Timeout,
			)

			if tc.expPass {
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeInitChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointA.GetChannel(), proposedUpgrade)
				channel := path.EndpointA.GetChannel()

				suite.Require().NoError(err)
				suite.Require().Equal(expSequence, channel.UpgradeSequence)
				suite.Require().Equal(mock.Version, channel.Version)
				suite.Require().Equal(types.INITUPGRADE, channel.State)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestValidateProposedUpgradeFields() {
	var (
		proposedUpgrade *types.UpgradeFields
		path            *ibctesting.Path
	)

	tests := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "change channel version",
			malleate: func() {
				proposedUpgrade.Version = "1.0.0"
			},
			expPass: true,
		},
		{
			name: "change connection hops",
			malleate: func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				suite.coordinator.Setup(path)
				proposedUpgrade.ConnectionHops = []string{path.EndpointA.ConnectionID}
			},
			expPass: true,
		},
		{
			name:     "fails with unmodified fields",
			malleate: func() {},
			expPass:  false,
		},
		{
			name: "fails when connection is not set",
			malleate: func() {
				storeKey := suite.chainA.GetSimApp().GetKey(exported.StoreKey)
				kvStore := suite.chainA.GetContext().KVStore(storeKey)
				kvStore.Delete(host.ConnectionKey(ibctesting.FirstConnectionID))
			},
			expPass: false,
		},
		{
			name: "fails when connection is not open",
			malleate: func() {
				connection := path.EndpointA.GetConnection()
				connection.State = connectiontypes.UNINITIALIZED
				path.EndpointA.SetConnection(connection)
			},
			expPass: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			existingChannel := path.EndpointA.GetChannel()
			proposedUpgrade = &types.UpgradeFields{
				Ordering:       existingChannel.Ordering,
				ConnectionHops: existingChannel.ConnectionHops,
				Version:        existingChannel.Version,
			}

			tc.malleate()

			err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ValidateUpgradeFields(suite.chainA.GetContext(), *proposedUpgrade, existingChannel)
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

