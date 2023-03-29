package keeper_test

import (
	"fmt"

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
				suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.SetUpgradeSequence(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 5)
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

			tc.malleate()

			sequence, version, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				chanCap, channelUpgrade, path.EndpointB.Chain.GetTimeoutHeight(), 0,
			)

			if tc.expPass {
				suite.Require().NoError(err)

				expSequence, found := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeSequence(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(expSequence, sequence)

				expVersion, found := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetAppVersion(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found)
				suite.Require().Equal(expVersion, version)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestRestoreChannel() {
	var (
		path           *ibctesting.Path
		channelUpgrade types.Channel
	)

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

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			originalChannel := path.EndpointA.GetChannel()

			chanCap, _ := suite.chainA.GetSimApp().GetScopedIBCKeeper().GetCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			channelUpgrade = types.NewChannel(types.INITUPGRADE, types.UNORDERED, types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, fmt.Sprintf("%s-v2", mock.Version))

			sequence, _, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
				chanCap, channelUpgrade, path.EndpointB.Chain.GetTimeoutHeight(), 0,
			)

			suite.Require().NoError(err)

			// update the channel to have some other state.
			modifiedChannel := originalChannel
			modifiedChannel.Version = "different-version"
			modifiedChannel.ConnectionHops = []string{"different-connection-id"}
			path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, modifiedChannel)

			tc.malleate()

			err = suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.RestoreChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sequence)

			actualChannel, ok := path.EndpointA.Chain.GetSimApp().IBCKeeper.ChannelKeeper.GetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			errReceipt, errReceiptPresent := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().True(ok)
				suite.Require().Equal(originalChannel, actualChannel)
				suite.Require().True(errReceiptPresent)
				suite.Require().Equal(sequence, errReceipt.Sequence)
			} else {
				suite.Require().Error(err)
				suite.Require().True(ok)
				suite.Require().Equal(modifiedChannel, actualChannel)
				suite.Require().True(errReceiptPresent)
				suite.Require().Equal(sequence, errReceipt.Sequence)
			}
		})
	}
}
