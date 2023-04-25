package keeper_test

import (
	"fmt"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
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
		path                       *ibctesting.Path
		expSequence                uint64
		upgrade                    *types.Upgrade
		counterpartyUpgradeTimeout types.UpgradeTimeout
		proposedConnectionHops     []string
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
				counterpartyChannel.UpgradeSequence = 1
				path.EndpointA.SetChannel(counterpartyChannel)

				channel := path.EndpointB.GetChannel()
				channel.UpgradeSequence = 3
				path.EndpointB.SetChannel(channel)
			},
			false,
		},
		{
			"proposed connection hops do not match the counterparty proposed connection hops",
			func() {
				proposedConnectionHops = []string{"connection-100"}
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
			proposedConnectionHops = []string{path.EndpointB.ConnectionID}

			upgrade = types.NewUpgrade(
				types.NewUpgradeFields(
					types.UNORDERED, proposedConnectionHops, fmt.Sprintf("%s-v2", mock.Version),
				),
				types.NewUpgradeTimeout(path.EndpointA.Chain.GetTimeoutHeight(), 0),
				0,
			)

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
			suite.Require().NoError(path.EndpointB.UpdateClient())

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofCounterpartyChannel, proofHeight := suite.chainA.QueryProof(channelKey)

			upgradeKey := host.ChannelUpgradeKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofUpgrade, _ := suite.chainA.QueryProof(upgradeKey)

			counterpartyUpgradeSequence := path.EndpointA.GetChannel().UpgradeSequence

			proposedUpgrade, _, err := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeTry(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, proposedConnectionHops, upgrade.Timeout,
				counterpartyUpgrade, counterpartyUpgradeSequence, proofCounterpartyChannel, proofUpgrade, proofHeight)

			if err == nil {
				// we need to write the upgradeTry so that the correct channel state is returned for chain B
				suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeTryChannel(
					suite.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID,
					path.EndpointB.ChannelID,
					path.EndpointB.GetChannel(),
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

func (suite *KeeperTestSuite) TestChanUpgradeTry_CrossingHellos() {
	var (
		path                       *ibctesting.Path
		expSequence                uint64
		upgrade                    *types.Upgrade
		counterpartyUpgradeTimeout types.UpgradeTimeout
		counterpartyUpgrade        types.Upgrade
		err                        error
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
			"fail: upgrade fields have changed. NOTE that this actually fails on verifying the channel proof",
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

			tc.malleate()

			// we need to update the clients again because malleation has changed the channel state
			suite.Require().NoError(path.EndpointA.UpdateClient())
			suite.Require().NoError(path.EndpointB.UpdateClient())

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofCounterpartyChannel, proofHeight := suite.chainA.QueryProof(channelKey)

			upgradeKey := host.ChannelUpgradeKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofUpgrade, _ := suite.chainA.QueryProof(upgradeKey)

			counterpartyUpgradeSequence := path.EndpointA.GetChannel().UpgradeSequence

			proposedUpgrade, _, err := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeTry(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, proposedConnectionHops, upgrade.Timeout,
				counterpartyUpgrade, counterpartyUpgradeSequence, proofCounterpartyChannel, proofUpgrade, proofHeight)

			if err == nil {
				// we need to write the upgradeTry so that the correct channel state is returned for chain B
				suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeTryChannel(
					suite.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID,
					path.EndpointB.ChannelID,
					path.EndpointB.GetChannel(),
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

func (suite *KeeperTestSuite) TestChanUpgradeAck() {
	var (
		path                        *ibctesting.Path
		counterpartyUpgradeSequence uint64
		upgradeTimeout              types.UpgradeTimeout
	)

	testCases := []struct {
		name        string
		malleate    func()
		expPass     bool
		shouldPanic bool
	}{
		{
			"success",
			func() {},
			true,
			false,
		},
		{
			name: "panics if channel not found",
			malleate: func() {
				storeKey := suite.chainA.GetSimApp().GetKey(exported.StoreKey)
				kvStore := suite.chainA.GetContext().KVStore(storeKey)
				kvStore.Delete(host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			},
			expPass:     false,
			shouldPanic: true,
		},
		{
			"channel state is not in OPEN or INITUPGRADE state",
			func() {
				suite.Require().NoError(path.EndpointA.SetChannelState(types.CLOSED))
			},
			false,
			false,
		},
		{
			"counterparty channel state is not in TRYUPGRADE",
			func() {
				counterparty := path.EndpointB.GetChannel()
				counterparty.State = types.OPEN
				path.EndpointB.SetChannel(counterparty)
				path.EndpointB.Chain.Coordinator.CommitBlock(path.EndpointB.Chain)
				suite.Require().NoError(path.EndpointA.UpdateClient())
			},
			false,
			false,
		},
		{
			"counterparty channel ordering is not equal to current channel ordering",
			func() {
				counterparty := path.EndpointB.GetChannel()
				counterparty.Ordering = types.ORDERED
				path.EndpointB.SetChannel(counterparty)
				path.EndpointB.Chain.Coordinator.CommitBlock(path.EndpointB.Chain)
				suite.Require().NoError(path.EndpointA.UpdateClient())
			},
			false,
			false,
		},
		{
			"counterparty channel connection not found",
			func() {
				channel := path.EndpointB.GetChannel()
				channel.ConnectionHops = []string{"connection-100"}
				path.EndpointB.SetChannel(channel)
				path.EndpointB.Chain.Coordinator.CommitBlock(path.EndpointB.Chain)
				suite.Require().NoError(path.EndpointA.UpdateClient())
			},
			false,
			false,
		},
		{
			"invalid proposed upgrade channel connection state",
			func() {
				connectionEnd := path.EndpointA.GetConnection()
				connectionEnd.State = connectiontypes.UNINITIALIZED
				path.EndpointA.SetConnection(connectionEnd)
				path.EndpointB.Chain.Coordinator.CommitBlock(path.EndpointB.Chain)
				suite.Require().NoError(path.EndpointA.UpdateClient())
			},
			false,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			path.EndpointA.ChannelConfig.Version = fmt.Sprintf("%s-v2", mock.Version)
			path.EndpointB.ChannelConfig.Version = fmt.Sprintf("%s-v2", mock.Version)

			counterpartyUpgradeSequence = 1
			upgradeTimeout = types.UpgradeTimeout{Height: path.EndpointB.Chain.GetTimeoutHeight(), Timestamp: uint64(suite.coordinator.CurrentTime.Add(time.Hour).UnixNano())}
			err := path.EndpointA.ChanUpgradeInit(upgradeTimeout.Height, upgradeTimeout.Timestamp)
			suite.Require().NoError(err)

			err = path.EndpointB.ChanUpgradeTry(upgradeTimeout.Height, upgradeTimeout.Timestamp, counterpartyUpgradeSequence)
			suite.Require().NoError(err)

			tc.malleate()

			channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			counterpartyChannelProof, proofHeight := suite.chainB.QueryProof(channelKey)

			testFn := func() {
				err = suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeAck(
					suite.chainA.GetContext(),
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					counterpartyUpgradeSequence,
					counterpartyChannelProof,
					proofHeight,
				)
			}

			if tc.shouldPanic {
				suite.Require().Panics(testFn)
				return
			}

			suite.Require().NotPanicsf(testFn, "unexpected panic for test case %s", tc.name)

			suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeAckChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointA.GetChannel())

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(types.OPEN, path.EndpointA.GetChannel().State)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
