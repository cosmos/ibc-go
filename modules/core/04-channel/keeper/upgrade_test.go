package keeper_test

import (
	"errors"
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

func (suite *KeeperTestSuite) TestChanUpgradeTry() {
	var (
		path                        *ibctesting.Path
		expSequence                 uint64
		counterpartyUpgrade         types.Upgrade
		counterpartyUpgradeSequence uint64
		proposedConnectionHops      []string
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
			"success: counterparty upgrade sequence",
			func() {
				counterpartyUpgradeSequence = 5
				expSequence = 5
			},
			true,
		},
		{
			"error receipt set with smaller counterparty upgrade sequence",
			func() {
				counterpartyUpgradeSequence = 2

				channel := path.EndpointB.GetChannel()
				channel.UpgradeSequence = 4
				path.EndpointB.SetChannel(channel)
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
			"channel state is not in OPEN or INITUPGRADE state",
			func() {
				suite.Require().NoError(path.EndpointB.SetChannelState(types.CLOSED))
			},
			false,
		},
		{
			"timeout has passed",
			func() {
				counterpartyUpgrade.Timeout = types.NewUpgradeTimeout(clienttypes.NewHeight(0, 1), 0)
			},
			false,
		},
		{
			"invalid connection state",
			func() {
				connectionEnd := path.EndpointB.GetConnection()
				connectionEnd.State = connectiontypes.UNINITIALIZED
				suite.chainB.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chainB.GetContext(), proposedConnectionHops[0], connectionEnd)
			},
			false,
		},
		{
			"mismatched connection hops",
			func() {
				counterpartyUpgrade.Fields = types.NewUpgradeFields(
					types.UNORDERED,
					[]string{"connection-100"},
					mock.Version,
				)
			},
			false,
		},
		{
			"upgrade field validation failed",
			func() {
				counterpartyUpgrade.Fields = types.NewUpgradeFields(
					types.UNORDERED,
					proposedConnectionHops,
					mock.Version,
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

			counterpartyUpgradeTimeout := types.NewUpgradeTimeout(path.EndpointB.Chain.GetTimeoutHeight(), 0)
			proposedConnectionHops = []string{path.EndpointB.ConnectionID}

			var err error
			counterpartyUpgrade, err = suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
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
			suite.Require().NoError(path.EndpointB.UpdateClient())

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofCounterpartyChannel, proofHeight := suite.chainA.QueryProof(channelKey)

			upgradeKey := host.ChannelUpgradeKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofUpgrade, _ := suite.chainA.QueryProof(upgradeKey)

			counterpartyUpgradeSequence = path.EndpointA.GetChannel().UpgradeSequence
			expSequence = 1

			tc.malleate()

			proposedUpgrade, err := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeTry(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, proposedConnectionHops, types.NewUpgradeTimeout(path.EndpointA.Chain.GetTimeoutHeight(), 0),
				counterpartyUpgrade, counterpartyUpgradeSequence, proofCounterpartyChannel, proofUpgrade, proofHeight)

			errorReceipt, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			if err == nil {
				// we need to write the upgradeTry so that the correct channel state is returned for chain B
				suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeTryChannel(
					suite.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID,
					path.EndpointB.ChannelID,
					proposedUpgrade,
				)
			}

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().False(found)
				suite.Require().Equal(expSequence, path.EndpointB.GetChannel().UpgradeSequence)
				suite.Require().Equal(mock.Version, path.EndpointB.GetChannel().Version)
				suite.Require().Equal(path.EndpointB.GetChannel().State, types.TRYUPGRADE)
			} else {
				if found {
					err = errors.New(errorReceipt.Error)
				}
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestChanUpgradeTry_CrossingHellos() {
	var (
		path                        *ibctesting.Path
		expSequence                 uint64
		upgrade                     *types.Upgrade
		counterpartyUpgradeTimeout  types.UpgradeTimeout
		counterpartyUpgrade         types.Upgrade
		counterpartyUpgradeSequence uint64
		err                         error
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
				counterpartyUpgradeSequence = 5
				expSequence = 5
			},
			true,
		},
		{
			"fail: upgrade fields have changed",
			func() {
				counterpartyUpgrade.Fields.Ordering = types.ORDERED
				counterpartyUpgrade.Fields.Version = fmt.Sprintf("%s-v3", mock.Version)
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

			proposedConnectionHops := []string{path.EndpointB.ConnectionID}

			counterpartyUpgradeTimeout = types.NewUpgradeTimeout(path.EndpointB.Chain.GetTimeoutHeight(), 0)

			upgrade = types.NewUpgrade(
				types.NewUpgradeFields(
					types.UNORDERED, proposedConnectionHops, fmt.Sprintf("%s-v2", mock.Version),
				),
				types.NewUpgradeTimeout(path.EndpointA.Chain.GetTimeoutHeight(), 0),
				0,
			)

			counterpartyUpgrade, err = suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
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
			suite.Require().NoError(path.EndpointB.UpdateClient())

			// we also UpgradeInit to simulate crossing hellos situation
			_, err = suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainB.GetContext(),
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				upgrade.Fields,
				upgrade.Timeout,
			)
			suite.Require().NoError(err)

			// we need to write the upgradeInit so that the correct channel state is returned for chain B
			suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeInitChannel(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
				path.EndpointB.GetChannel(), *upgrade,
			)

			// commit a block to update chain B for correct proof querying
			path.EndpointB.Chain.Coordinator.CommitBlock(path.EndpointB.Chain)
			// update chainA's client of chain B to account for ChanUpgradeInit
			suite.Require().NoError(path.EndpointA.UpdateClient())

			counterpartyUpgradeSequence = path.EndpointA.GetChannel().UpgradeSequence

			tc.malleate()

			// we need to update the clients again because malleation has changed the channel state
			suite.Require().NoError(path.EndpointA.UpdateClient())
			suite.Require().NoError(path.EndpointB.UpdateClient())

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofCounterpartyChannel, proofHeight := suite.chainA.QueryProof(channelKey)

			upgradeKey := host.ChannelUpgradeKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofUpgrade, _ := suite.chainA.QueryProof(upgradeKey)

			proposedUpgrade, err := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeTry(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, proposedConnectionHops, upgrade.Timeout,
				counterpartyUpgrade, counterpartyUpgradeSequence, proofCounterpartyChannel, proofUpgrade, proofHeight)

			if err == nil {
				// we need to write the upgradeTry so that the correct channel state is returned for chain B
				suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeTryChannel(
					suite.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID,
					path.EndpointB.ChannelID,
					proposedUpgrade,
				)
			}

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expSequence, path.EndpointB.GetChannel().UpgradeSequence)
				suite.Require().Equal(mock.Version, path.EndpointB.GetChannel().Version)
				suite.Require().Equal(path.EndpointB.GetChannel().State, types.TRYUPGRADE)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
