package keeper_test

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
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
		path                        *ibctesting.Path
		chanCap                     *capabilitytypes.Capability
		channelUpgrade              types.Channel
		counterpartyUpgradeSequence uint64
		upgradeTimeout              types.UpgradeTimeout
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
				// set the counterparty (chainA) upgrade sequence to 10
				counterpartyUpgradeSequence = 10
				suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.SetUpgradeSequence(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, counterpartyUpgradeSequence)

				// set the TRY handler upgrade sequence to the expected value (counterpartySequence - 1)
				suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.SetUpgradeSequence(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, counterpartyUpgradeSequence-1)

				suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
				suite.Require().NoError(path.EndpointB.UpdateClient())
			},
			true,
		},
		{
			"crossing hellos: success",
			func() {
				// modify the version on chain B so the channel and upgrade channel are not identical.
				path.EndpointB.ChannelConfig.Version = fmt.Sprintf("%s-v2", mock.Version)
				// call ChanUpgradeInit on chain B to ensure there is restore channel
				err := path.EndpointB.ChanUpgradeInit(path.EndpointA.Chain.GetTimeoutHeight(), 0)
				suite.Require().NoError(err)

				suite.Require().NoError(path.EndpointB.UpdateClient())
			},
			true,
		},
		{
			"crossing hellos: invalid proposed upgrade channel",
			func() {
				// modify the version on chain B so the channel and upgrade channel are not identical.
				path.EndpointB.ChannelConfig.Version = fmt.Sprintf("%s-v2", mock.Version)
				// call ChanUpgradeInit on chain B to ensure there is restore channel

				channelUpgrade.Version = "different-version"

				err := path.EndpointB.ChanUpgradeInit(path.EndpointA.Chain.GetTimeoutHeight(), 0)
				suite.Require().NoError(err)

				suite.Require().NoError(path.EndpointB.UpdateClient())
			},
			false,
		},

		{
			"crossing hellos: upgrade sequence not found",
			func() {
				// modify the version on chain B so the channel and upgrade channel are not identical.
				path.EndpointB.ChannelConfig.Version = fmt.Sprintf("%s-v2", mock.Version)
				// call ChanUpgradeInit on chain B to ensure there is restore channel

				err := path.EndpointB.ChanUpgradeInit(path.EndpointA.Chain.GetTimeoutHeight(), 0)
				suite.Require().NoError(err)

				// delete upgrade sequence
				store := suite.chainB.GetContext().KVStore(suite.chainB.GetSimApp().GetKey(exported.StoreKey))
				store.Delete(host.ChannelUpgradeSequenceKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))

				suite.Require().NoError(path.EndpointB.UpdateClient())
			},
			false,
		},
		{
			"crossing hellos: invalid upgrade sequence",
			func() {
				// modify the version on chain B so the channel and upgrade channel are not identical.
				path.EndpointB.ChannelConfig.Version = fmt.Sprintf("%s-v2", mock.Version)

				// call ChanUpgradeInit on chain B to ensure there is restore channel
				err := path.EndpointB.ChanUpgradeInit(path.EndpointA.Chain.GetTimeoutHeight(), 0)
				suite.Require().NoError(err)

				counterpartyUpgradeSequence = 20
				suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.SetUpgradeSequence(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, counterpartyUpgradeSequence)

				suite.coordinator.CommitBlock(suite.chainA, suite.chainB)

				suite.Require().NoError(path.EndpointB.UpdateClient())
			},
			false,
		},
		{
			"success with crossing hellos with later upgrade sequence",
			func() {
				// modify the version on chain B so the channel and upgrade channel are not identical.
				path.EndpointB.ChannelConfig.Version = fmt.Sprintf("%s-v2", mock.Version)
				// call ChanUpgradeInit on chain B to ensure there is restore channel
				err := path.EndpointB.ChanUpgradeInit(path.EndpointA.Chain.GetTimeoutHeight(), 0)
				suite.Require().NoError(err)

				counterpartyUpgradeSequence = 10
				suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.SetUpgradeSequence(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, counterpartyUpgradeSequence)
				suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.SetUpgradeSequence(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, counterpartyUpgradeSequence)

				suite.coordinator.CommitBlock(suite.chainA, suite.chainB)

				suite.Require().NoError(path.EndpointB.UpdateClient())
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
			"connection not found",
			func() {
				channelUpgrade.ConnectionHops = []string{"connection-100"}
			},
			false,
		},
		{
			"invalid connection upgrade, counterparty mismatch",
			func() {
				channel := path.EndpointA.GetChannel()
				channel.ConnectionHops = []string{"connection-100"}

				path.EndpointA.SetChannel(channel)
			},
			false,
		},
		{
			"error receipt is written for invalid upgrade sequence",
			func() {
				// set the TRY handler upgrade sequence to 10 to trigger a failure
				suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.SetUpgradeSequence(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, 10)

				suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
				suite.Require().NoError(path.EndpointB.UpdateClient())
			},
			false,
		},
		{
			"error timeout height",
			func() {
				upgradeTimeout.TimeoutHeight = clienttypes.NewHeight(1, 1)
				upgradeTimeout.TimeoutTimestamp = 0

				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetUpgradeTimeout(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgradeTimeout)

				suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
				suite.Require().NoError(path.EndpointB.UpdateClient())
			},
			false,
		},
		{
			"error timeout timestamp",
			func() {
				upgradeTimeout.TimeoutHeight = clienttypes.ZeroHeight()
				upgradeTimeout.TimeoutTimestamp = uint64(time.Second)

				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetUpgradeTimeout(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgradeTimeout)

				suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
				suite.Require().NoError(path.EndpointB.UpdateClient())
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

			path.EndpointA.ChannelConfig.Version = fmt.Sprintf("%s-v2", mock.Version)

			counterpartyUpgradeSequence = 1
			upgradeTimeout = types.UpgradeTimeout{TimeoutHeight: path.EndpointB.Chain.GetTimeoutHeight(), TimeoutTimestamp: uint64(suite.coordinator.CurrentTime.Add(time.Hour).UnixNano())}
			err := path.EndpointA.ChanUpgradeInit(upgradeTimeout.TimeoutHeight, upgradeTimeout.TimeoutTimestamp)
			suite.Require().NoError(err)

			chanCap, _ = suite.chainB.GetSimApp().GetScopedIBCKeeper().GetCapability(suite.chainB.GetContext(), host.ChannelCapabilityPath(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
			channelUpgrade = types.NewChannel(types.TRYUPGRADE, types.UNORDERED, types.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, fmt.Sprintf("%s-v2", mock.Version))

			tc.malleate()

			channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofChannel, proofHeight := suite.chainA.QueryProof(channelKey)

			upgradeSequenceKey := host.ChannelUpgradeSequenceKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofUpgradeSequence, _ := suite.chainA.QueryProof(upgradeSequenceKey)

			upgradeTimeoutKey := host.ChannelUpgradeTimeoutKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			proofUpgradeTimeout, _ := suite.chainA.QueryProof(upgradeTimeoutKey)

			upgradeSequence, previousVersion, err := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeTry(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
				chanCap, path.EndpointA.GetChannel(), counterpartyUpgradeSequence, channelUpgrade, upgradeTimeout.TimeoutHeight, upgradeTimeout.TimeoutTimestamp,
				proofChannel, proofUpgradeTimeout, proofUpgradeSequence, proofHeight,
			)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(counterpartyUpgradeSequence, upgradeSequence)
				suite.Require().Equal(mock.Version, previousVersion)
			} else {
				suite.Require().Error(err)

				if errorsmod.IsOf(err, types.ErrInvalidUpgradeSequence, types.ErrUpgradeAborted, types.ErrUpgradeTimeout) {
					errorReceipt, found := suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetUpgradeErrorReceipt(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
					suite.Require().True(found)
					suite.Require().NotNil(errorReceipt)
				}
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
				expectedChannel.Version = fmt.Sprintf("%s-v2", mock.Version)

				suite.Require().Error(err)
				suite.Require().True(ok)
				suite.Require().Equal(expectedChannel, actualChannel)
				suite.Require().True(errReceiptPresent)
				suite.Require().Equal(upgradeSequence, errReceipt.Sequence)
			}
		})
	}
}
