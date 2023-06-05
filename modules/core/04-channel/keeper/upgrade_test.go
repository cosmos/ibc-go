package keeper_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
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

// UpgradeInit will be run on chainA and startFlushUpgradeHandshake
// will be called on chainB
func (suite *KeeperTestSuite) TestStartFlushUpgradeHandshake() {
	var (
		path                *ibctesting.Path
		upgrade             types.Upgrade
		counterpartyChannel types.Channel
		counterpartyUpgrade types.Upgrade
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"channel not found",
			func() {
				path.EndpointB.ChannelID = "invalid-channel"
				path.EndpointB.ChannelConfig.PortID = "invalid-port"
			},
			types.ErrChannelNotFound,
		},
		{
			"connection not found",
			func() {
				channel := path.EndpointB.GetChannel()
				channel.ConnectionHops[0] = ibctesting.InvalidID
				path.EndpointB.SetChannel(channel)
			},
			connectiontypes.ErrConnectionNotFound,
		},
		{
			"connection state is not in OPEN state",
			func() {
				conn := path.EndpointB.GetConnection()
				conn.State = connectiontypes.INIT
				path.EndpointB.SetConnection(conn)
			},
			connectiontypes.ErrInvalidConnectionState,
		},
		{
			"failed verification for counterparty channel state due to incorrectly constructed counterparty channel",
			func() {
				counterpartyChannel.State = types.CLOSED
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"failed verification for counterparty upgrade due to incorrectly constructed counterparty upgrade",
			func() {
				counterpartyUpgrade.LatestSequenceSend = 100
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"upgrade sequence mismatch, endpointB channel upgrade sequence is ahead",
			func() {
				channel := path.EndpointB.GetChannel()
				channel.UpgradeSequence++
				path.EndpointB.SetChannel(channel)
			},
			types.NewUpgradeError(2, types.ErrIncompatibleCounterpartyUpgrade), // max sequence will be returned
		},
		{
			"upgrade ordering is not the same on both sides",
			func() {
				upgrade.Fields.Ordering = types.ORDERED
			},
			types.NewUpgradeError(1, types.ErrIncompatibleCounterpartyUpgrade),
		},
		{
			"proposed connection is not found",
			func() {
				upgrade.Fields.ConnectionHops[0] = ibctesting.InvalidID
			},
			types.NewUpgradeError(1, connectiontypes.ErrConnectionNotFound),
		},
		{
			"proposed connection is not in OPEN state",
			func() {
				// reuse existing connection to create a new connection in a non OPEN state
				connectionEnd := path.EndpointB.GetConnection()
				connectionEnd.State = connectiontypes.UNINITIALIZED
				connectionEnd.Counterparty.ConnectionId = counterpartyUpgrade.Fields.ConnectionHops[0] // both sides must be each other's counterparty

				// set proposed connection in state
				proposedConnectionID := "connection-100"
				suite.chainB.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chainB.GetContext(), proposedConnectionID, connectionEnd)
				upgrade.Fields.ConnectionHops[0] = proposedConnectionID
			},
			types.NewUpgradeError(1, connectiontypes.ErrInvalidConnectionState),
		},
		{
			"proposed connection ends are not each other's counterparty",
			func() {
				// reuse existing connection to create a new connection in a non OPEN state
				connectionEnd := path.EndpointB.GetConnection()
				// ensure counterparty connectionID does not match connectionID set in counterparty proposed upgrade
				connectionEnd.Counterparty.ConnectionId = "connection-50"

				// set proposed connection in state
				proposedConnectionID := "connection-100"
				suite.chainB.GetSimApp().GetIBCKeeper().ConnectionKeeper.SetConnection(suite.chainB.GetContext(), proposedConnectionID, connectionEnd)
				upgrade.Fields.ConnectionHops[0] = proposedConnectionID
			},
			types.NewUpgradeError(1, types.ErrIncompatibleCounterpartyUpgrade),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			upgradeVersion := fmt.Sprintf("%s-v2", mock.Version)
			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = upgradeVersion
			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			// ensure proof verification succeeds
			err = path.EndpointB.UpdateClient()
			suite.Require().NoError(err)

			proofChannel, proofUpgrade, proofHeight := path.EndpointB.QueryChannelUpgradeProof()
			counterpartyChannel = path.EndpointA.GetChannel()

			var found bool
			counterpartyUpgrade, found = path.EndpointA.Chain.App.GetIBCKeeper().ChannelKeeper.GetUpgrade(path.EndpointA.Chain.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			suite.Require().True(found)

			// ensure that the channel has a valid upgrade sequence
			channel := path.EndpointB.GetChannel()
			channel.UpgradeSequence = 1
			path.EndpointB.SetChannel(channel)

			upgrade = types.Upgrade{
				Fields: types.UpgradeFields{
					Ordering:       types.UNORDERED,
					ConnectionHops: []string{path.EndpointB.ConnectionID},
					Version:        upgradeVersion,
				},
				Timeout:            types.NewTimeout(path.EndpointA.Chain.GetTimeoutHeight(), 0),
				LatestSequenceSend: 1,
			}

			tc.malleate()

			err = suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.StartFlushUpgradeHandshake(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, upgrade.Fields,
				counterpartyChannel, counterpartyUpgrade, proofChannel, proofUpgrade, proofHeight,
			)

			if tc.expError != nil {
				suite.Require().Error(err)

				if expUpgradeError, ok := tc.expError.(types.UpgradeError); ok {
					upgradeError, ok := err.(types.UpgradeError)
					suite.Require().True(ok)
					suite.Require().Equal(expUpgradeError.GetErrorReceipt(), upgradeError.GetErrorReceipt())
				}

				suite.Require().True(errorsmod.IsOf(err, tc.expError), err)
			} else {
				suite.Require().NoError(err)
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
