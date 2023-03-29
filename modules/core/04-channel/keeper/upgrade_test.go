package keeper_test

import (
	"fmt"
	"time"

	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/mock"
)

func (suite *KeeperTestSuite) TestChanUpgradeInit() {
	var (
		path           *ibctesting.Path
		chanCap        *capabilitytypes.Capability
		channelUpgrade types.Channel
		expSequence    uint64
		expVersion     string
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
				// set the initial sequence and expected sequence (initial sequence + 1)
				suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.SetUpgradeSequence(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 4)
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
			"invalid capability",
			func() {
				chanCap = capabilitytypes.NewCapability(42)
			},
			false,
		},
		{
			"identical upgrade channel end",
			func() {
				channelUpgrade = types.NewChannel(types.INITUPGRADE, types.UNORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, mock.Version)
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
			"invalid proposed channel connection",
			func() {
				channelUpgrade.ConnectionHops = []string{"connection-100"}
			},
			false,
		},
		{
			"invalid proposed channel counterparty",
			func() {
				channelUpgrade.Counterparty = types.NewCounterparty(mock.PortID, "channel-100")
			},
			false,
		},
		{
			"invalid proposed channel upgrade ordering",
			func() {
				channelUpgrade.Ordering = types.ORDERED
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

			chanCap, _ = suite.chainA.GetSimApp().GetScopedIBCKeeper().GetCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			channelUpgrade = types.NewChannel(types.INITUPGRADE, types.UNORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, fmt.Sprintf("%s-v2", mock.Version))

			expSequence = 1
			expVersion = mock.Version

			tc.malleate()

			sequence, previousVersion, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				chanCap, channelUpgrade, path.EndpointB.Chain.GetTimeoutHeight(), 0,
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
		path            *ibctesting.Path
		chanCap         *capabilitytypes.Capability
		channelUpgrade  types.Channel
		upgradeSequence uint64
		upgradeTimeout  types.UpgradeTimeout
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
			"invalid capability",
			func() {
				chanCap = capabilitytypes.NewCapability(42)
			},
			false,
		},
		{
			"channel not found",
			func() {
				path.EndpointB.ChannelID = "invalid-channel"
				path.EndpointB.ChannelConfig.PortID = "invalid-port"
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
			"invalid proposed channel counterparty",
			func() {
				channelUpgrade.Counterparty = types.NewCounterparty(mock.PortID, "channel-100")
			},
			false,
		},
		{
			"invalid proposed channel upgrade ordering",
			func() {
				channelUpgrade.Ordering = types.ORDERED
			},
			false,
		},
		{
			"counterparty channel order mismatch",
			func() {
				channelEnd := path.EndpointA.GetChannel()
				channelEnd.Ordering = types.ORDERED

				path.EndpointA.SetChannel(channelEnd)
			},
			false,
		},
		{
			name: "crossing hellos",
			malleate: func() {
				// modify the version on chain B so the channel and upgrade channel are not identical.
				path.EndpointB.ChannelConfig.Version = fmt.Sprintf("%s-v2", mock.Version)
				channel := path.EndpointB.GetChannel()
				// call ChanUpgradeInit on chain B to ensure there is restore channel
				err := path.EndpointB.ChanUpgradeInit(path.EndpointA.Chain.GetTimeoutHeight(), 0)
				suite.Require().NoError(err)
				channel.State = types.INITUPGRADE
				path.EndpointB.SetChannel(channel)
			},
			expPass: true,
		},

		// TODO: add test cases
		// error receipt if counterpartyUpgradeSequence > upgradeSequence
		// connection.GetCounterparty().GetConnectionID() != counterpartyChannel.ConnectionHops[0]
		// connection, err := k.GetConnection(ctx, proposedUpgradeChannel.ConnectionHops[0])
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			path.EndpointA.ChannelConfig.Version = fmt.Sprintf("%s-v2", mock.Version)

			upgradeSequence = 1
			upgradeTimeout = types.UpgradeTimeout{TimeoutHeight: path.EndpointB.Chain.GetTimeoutHeight(), TimeoutTimestamp: uint64(suite.coordinator.CurrentTime.Add(time.Hour).UnixNano())}
			err := path.EndpointA.ChanUpgradeInit(upgradeTimeout.TimeoutHeight, upgradeTimeout.TimeoutTimestamp)
			suite.Require().NoError(err)

			chanCap, _ = suite.chainB.GetSimApp().GetScopedIBCKeeper().GetCapability(suite.chainB.GetContext(), host.ChannelCapabilityPath(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
			channelUpgrade = types.NewChannel(types.TRYUPGRADE, types.UNORDERED, types.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, fmt.Sprintf("%s-v2", mock.Version))

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofChannel, proofHeight := suite.chainA.QueryProof(channelKey)

			upgradeSequenceKey := host.ChannelUpgradeSequenceKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofUpgradeSequence, _ := suite.chainA.QueryProof(upgradeSequenceKey)

			upgradeTimeoutKey := host.ChannelUpgradeTimeoutKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofUpgradeTimeout, _ := suite.chainA.QueryProof(upgradeTimeoutKey)

			tc.malleate()

			_, _, err = suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeTry(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
				chanCap, path.EndpointA.GetChannel(), upgradeSequence, channelUpgrade, upgradeTimeout.TimeoutHeight, upgradeTimeout.TimeoutTimestamp,
				proofChannel, proofUpgradeTimeout, proofUpgradeSequence, proofHeight,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestRestoreChannel() {
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"succeeds when restore channel is set",
			func() {},
			true,
		},
		{
			name: "fails when no restore channel is present",
			malleate: func() {
				// remove the restore channel
				path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeper.DeleteUpgradeRestoreChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			upgradeSequence := uint64(1)
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			path.EndpointA.ChannelConfig.Version = fmt.Sprintf("%s-v2", mock.Version)

			originalChannel := path.EndpointA.GetChannel()

			err := path.EndpointA.ChanUpgradeInit(path.EndpointB.Chain.GetTimeoutHeight(), 0)
			suite.Require().NoError(err)

			tc.malleate()

			err = suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.RestoreChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgradeSequence, types.ErrInvalidChannel)

			actualChannel, ok := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeper.GetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			errReceipt, errReceiptPresent := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().True(ok)
				suite.Require().Equal(originalChannel, actualChannel)
				suite.Require().True(errReceiptPresent)
				suite.Require().Equal(upgradeSequence, errReceipt.Sequence)
			} else {
				// channel should still be in INITUPGRADE if restore did not happen.
				expectedChannel := originalChannel
				expectedChannel.State = types.INITUPGRADE

				suite.Require().Error(err)
				suite.Require().True(ok)
				suite.Require().Equal(expectedChannel, actualChannel)
				suite.Require().True(errReceiptPresent)
				suite.Require().Equal(upgradeSequence, errReceipt.Sequence)
			}
		})
	}
}
