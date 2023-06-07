package keeper_test

import (
	"fmt"

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
					types.NewTimeout(path.EndpointB.Chain.GetTimeoutHeight(), 0),
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
				types.NewTimeout(path.EndpointB.Chain.GetTimeoutHeight(), 0),
				0,
			)

			tc.malleate()

			proposedUpgrade, err := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.ChanUpgradeInit(
				suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgrade.Fields, upgrade.Timeout,
			)

			if tc.expPass {
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.WriteUpgradeInitChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, proposedUpgrade)
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

// TestAbortHandshake tests that when the channel handshake is aborted, the channel state
// is restored the previous state and that an error receipt is written, and upgrade state which
// is no longer required is deleted.
func (suite *KeeperTestSuite) TestAbortHandshake() {
	var (
		path *ibctesting.Path
	)

	tests := []struct {
		name           string
		malleate       func()
		expPass        bool
		channelPresent bool
	}{
		{
			name:           "success",
			malleate:       func() {},
			expPass:        true,
			channelPresent: true,
		},
		{
			name: "upgrade does not exist",
			malleate: func() {
				suite.chainA.DeleteKey(host.ChannelUpgradeKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			},
			expPass:        false,
			channelPresent: true,
		},
		{
			name: "channel does not exist",
			malleate: func() {
				suite.chainA.DeleteKey(host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			},
			expPass:        false,
			channelPresent: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			channelKeeper := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper

			path.EndpointA.ChannelConfig.Version = mock.Version + "-v2"

			suite.Require().NoError(path.EndpointA.ChanUpgradeInit())

			path.EndpointA.Chain.Coordinator.CommitBlock(path.EndpointA.Chain)

			suite.Require().NoError(path.EndpointA.UpdateClient())
			suite.Require().NoError(path.EndpointB.UpdateClient())

			upgradeError := types.NewUpgradeError(1, types.ErrInvalidChannel)

			tc.malleate()

			err := channelKeeper.AbortHandshake(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, upgradeError)

			if tc.expPass {
				suite.Require().NoError(err)

				channel, found := channelKeeper.GetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().True(found, "channel should be found")

				suite.Require().Equal(types.OPEN, channel.State, "channel state should be OPEN")
				suite.Require().Equal(types.NOTINFLUSH, channel.FlushStatus, "channel flush status should be NOTINFLUSH")

				_, found = channelKeeper.GetUpgrade(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().False(found, "upgrade info should be deleted")

				errorReceipt, found := channelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				if upgradeError != nil {
					suite.Require().True(found, "error receipt should be found")
					suite.Require().Equal(upgradeError.GetErrorReceipt(), errorReceipt, "error receipt should be stored")
				}

				_, found = channelKeeper.GetCounterpartyLastPacketSequence(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().False(found, "counterparty last packet sequence should be found")

			} else {
				suite.Require().Error(err)
				channel, found := channelKeeper.GetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				if tc.channelPresent {
					suite.Require().True(found, "channel should be found")
					suite.Require().Equal(types.INITUPGRADE, channel.State, "channel state should not be restored to OPEN")
				}

				_, found = channelKeeper.GetUpgradeErrorReceipt(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				suite.Require().False(found, "error receipt should not be written")

				// TODO: assertion that GetCounterpartyLastPacketSequence is present and correct
			}
		})
	}
}
