package keeper_test

import (
	"fmt"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
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
				counterpartyChannel := path.EndpointA.GetChannel()
				counterpartyChannel.UpgradeSequence = 4
				path.EndpointA.SetChannel(counterpartyChannel)

				channel := path.EndpointB.GetChannel()
				channel.UpgradeSequence = 3
				path.EndpointB.SetChannel(channel)

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
			"stricter proposed channel upgrade ordering",
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

			_, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgrade.Fields, upgrade.Timeout,
			)

			if tc.expPass {
				channel := path.EndpointA.GetChannel()
				suite.Require().NoError(err)
				suite.Require().Equal(expSequence, channel.UpgradeSequence)
				suite.Require().Equal(expVersion, channel.Version)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestChanUpgradeTry() {
	var (
		path                       *ibctesting.Path
		expSequence                uint64
		proposedUpgrade            *types.Upgrade
		counterpartyUpgradeTimeout types.UpgradeTimeout
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
			"errors with smaller counterparty upgrade sequence",
			func() {
				counterpartyChannel := path.EndpointA.GetChannel()
				counterpartyChannel.UpgradeSequence = 2
				path.EndpointA.SetChannel(counterpartyChannel)

				channel := path.EndpointB.GetChannel()
				channel.UpgradeSequence = 3
				path.EndpointB.SetChannel(channel)
			},
			false,
		},
		{
			"identical upgrade channel end",
			func() {
				channel := path.EndpointB.GetChannel()
				proposedUpgrade = types.NewUpgrade(
					types.NewUpgradeFields(
						channel.Ordering, channel.ConnectionHops, channel.Version,
					),
					types.NewUpgradeTimeout(path.EndpointA.Chain.GetTimeoutHeight(), 0),
					0,
				)
			},
			false,
		},
		{
			"channel not found",
			func() {
				path.EndpointB.ChannelID = "invalid-channel"
			},
			false,
		},
		{
			"channel state is not in OPEN state",
			func() {
				suite.Require().NoError(path.EndpointB.SetChannelState(types.CLOSED))
			},
			false,
		},
		{
			"timeout has passed",
			func() {
				counterpartyUpgradeTimeout = types.NewUpgradeTimeout(clienttypes.NewHeight(0, 1), 0)
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

			counterpartyUpgradeFields := types.NewUpgradeFields(
				types.UNORDERED,
				[]string{path.EndpointA.ConnectionID},
				fmt.Sprintf("%s-v2", mock.Version),
			)

			counterpartyUpgradeTimeout = types.NewUpgradeTimeout(path.EndpointB.Chain.GetTimeoutHeight(), 0)

			proposedUpgrade = types.NewUpgrade(
				types.NewUpgradeFields(
					types.UNORDERED, []string{path.EndpointB.ConnectionID}, fmt.Sprintf("%s-v2", mock.Version),
				),
				types.NewUpgradeTimeout(path.EndpointA.Chain.GetTimeoutHeight(), 0),
				0,
			)

			channel := path.EndpointB.GetChannel()

			tc.malleate()

			counterpartyUpgrade, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, counterpartyUpgradeFields,
				counterpartyUpgradeTimeout,
			)
			suite.Require().NoError(err)

			// we need to write the upgradeInit so that the correct channel state is returned for chain A
			suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeInitChannel(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				path.EndpointA.GetChannel(), counterpartyUpgrade,
			)

			// commit a block to update chain A for correct proof querying
			path.EndpointA.Chain.Coordinator.CommitBlock(path.EndpointA.Chain)
			// update chainB's client of chain A to account for ChanUpgradeInit
			path.EndpointB.UpdateClient()

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofCounterpartyChannel, proofHeight := suite.chainA.QueryProof(channelKey)

			upgradeKey := host.ChannelUpgradeKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofUpgrade, _ := suite.chainA.QueryProof(upgradeKey)

			counterpartyUpgradeSequence := path.EndpointA.GetChannel().UpgradeSequence

			sequence, err := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeTry(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, *proposedUpgrade,
				counterpartyUpgrade, counterpartyUpgradeSequence, proofCounterpartyChannel, proofUpgrade, proofHeight)

			// we need to write the upgradeTry so that the correct channel state is returned for chain B
			suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeTryChannel(
				suite.chainB.GetContext(),
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				// use the channel we saved before malleate so that we can trigger "channel not found" test
				channel,
				*proposedUpgrade,
			)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expSequence, sequence)
				suite.Require().Equal(mock.Version, path.EndpointB.GetChannel().Version)
				suite.Require().Equal(path.EndpointB.GetChannel().State, types.TRYUPGRADE)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestChanUpgradeTry_CrossingHellos() {
	var (
		path                       *ibctesting.Path
		expSequence                uint64
		proposedUpgrade            *types.Upgrade
		counterpartyUpgradeTimeout types.UpgradeTimeout
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				expSequence = 1
			},
			true,
		},
		{
			"success: counterparty sequence > channel.UpgradeSequence",
			func() {
				counterpartyChannel := path.EndpointA.GetChannel()
				counterpartyChannel.UpgradeSequence = 5
				path.EndpointA.SetChannel(counterpartyChannel)

				channel := path.EndpointB.GetChannel()
				channel.UpgradeSequence = 2
				path.EndpointB.SetChannel(channel)

				expSequence = 5
			},
			true,
		},
		{
			"fail: upgrade fields have changed",
			func() {
				proposedUpgrade = types.NewUpgrade(
					types.NewUpgradeFields(
						types.UNORDERED, []string{path.EndpointB.ConnectionID}, fmt.Sprintf("%s-v3", mock.Version),
					),
					types.NewUpgradeTimeout(path.EndpointA.Chain.GetTimeoutHeight(), 0),
					0,
				)
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

			counterpartyUpgradeFields := types.NewUpgradeFields(
				types.UNORDERED,
				[]string{path.EndpointA.ConnectionID},
				fmt.Sprintf("%s-v2", mock.Version),
			)

			counterpartyUpgradeTimeout = types.NewUpgradeTimeout(path.EndpointB.Chain.GetTimeoutHeight(), 0)

			proposedUpgrade = types.NewUpgrade(
				types.NewUpgradeFields(
					types.UNORDERED, []string{path.EndpointB.ConnectionID}, fmt.Sprintf("%s-v2", mock.Version),
				),
				types.NewUpgradeTimeout(path.EndpointA.Chain.GetTimeoutHeight(), 0),
				0,
			)

			counterpartyUpgrade, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, counterpartyUpgradeFields,
				counterpartyUpgradeTimeout,
			)
			suite.Require().NoError(err)

			// we need to write the upgradeInit so that the correct channel state is returned for chain A
			suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeInitChannel(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				path.EndpointA.GetChannel(), counterpartyUpgrade,
			)

			// commit a block to update chain A for correct proof querying
			path.EndpointA.Chain.Coordinator.CommitBlock(path.EndpointA.Chain)
			// update chainB's client of chain A to account for ChanUpgradeInit
			path.EndpointB.UpdateClient()

			// we also UpgradeInit to simulate crossing hellos situation
			suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainB.GetContext(),
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				proposedUpgrade.Fields,
				proposedUpgrade.Timeout,
			)

			// we need to write the upgradeInit so that the correct channel state is returned for chain B
			suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeInitChannel(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
				path.EndpointB.GetChannel(), *proposedUpgrade,
			)

			// commit a block to update chain B for correct proof querying
			path.EndpointB.Chain.Coordinator.CommitBlock(path.EndpointB.Chain)
			// update chainA's client of chain B to account for ChanUpgradeInit
			path.EndpointA.UpdateClient()

			tc.malleate()

			// we need to update the clients again because malleation has changed the channel state
			path.EndpointA.UpdateClient()
			path.EndpointB.UpdateClient()

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofCounterpartyChannel, proofHeight := suite.chainA.QueryProof(channelKey)

			upgradeKey := host.ChannelUpgradeKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofUpgrade, _ := suite.chainA.QueryProof(upgradeKey)

			sequence, err := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeTry(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, *proposedUpgrade,
				counterpartyUpgrade, path.EndpointA.GetChannel().UpgradeSequence, proofCounterpartyChannel, proofUpgrade, proofHeight)

			// we need to write the upgradeTry so that the correct channel state is returned for chain B
			suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeTryChannel(
				suite.chainB.GetContext(),
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				path.EndpointB.GetChannel(),
				*proposedUpgrade,
			)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expSequence, sequence)
				suite.Require().Equal(mock.Version, path.EndpointB.GetChannel().Version)
				suite.Require().Equal(path.EndpointB.GetChannel().State, types.TRYUPGRADE)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
