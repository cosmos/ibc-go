package keeper_test

import (
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v2/testing"
)

func (suite *KeeperTestSuite) TestOnChanOpenInit() {
	var (
		channel *channeltypes.Channel
		path    *ibctesting.Path
		chanCap *capabilitytypes.Capability
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{

		{
			"success",
			func() {
				path.EndpointA.SetChannel(*channel)
			},
			true,
		},
		{
			"invalid order - UNORDERED",
			func() {
				channel.Ordering = channeltypes.UNORDERED
			},
			false,
		},
		{
			"invalid port ID",
			func() {
				path.EndpointA.ChannelConfig.PortID = "invalid-port-id"
			},
			false,
		},
		{
			"invalid counterparty port ID",
			func() {
				path.EndpointA.SetChannel(*channel)
				channel.Counterparty.PortId = "invalid-port-id"
			},
			false,
		},
		{
			"invalid version",
			func() {
				path.EndpointA.SetChannel(*channel)
				channel.Version = "version"
			},
			false,
		},
		{
			"channel not found",
			func() {
				path.EndpointA.ChannelID = "invalid-channel-id"
			},
			false,
		},
		{
			"connection not found",
			func() {
				channel.ConnectionHops = []string{"invalid-connnection-id"}
				path.EndpointA.SetChannel(*channel)
			},
			false,
		},
		{
			"invalid connection sequence",
			func() {
				portID, err := types.GeneratePortID(TestOwnerAddress, "connection-1", "connection-0")
				suite.Require().NoError(err)

				path.EndpointA.ChannelConfig.PortID = portID
				path.EndpointA.SetChannel(*channel)
			},
			false,
		},
		{
			"invalid counterparty connection sequence",
			func() {
				portID, err := types.GeneratePortID(TestOwnerAddress, "connection-0", "connection-1")
				suite.Require().NoError(err)

				path.EndpointA.ChannelConfig.PortID = portID
				path.EndpointA.SetChannel(*channel)
			},
			false,
		},
		{
			"channel is already active",
			func() {
				suite.chainA.GetSimApp().ICAKeeper.SetActiveChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			},
			false,
		},
		{
			"capability already claimed",
			func() {
				path.EndpointA.SetChannel(*channel)
				err := suite.chainA.GetSimApp().ScopedICAKeeper.ClaimCapability(suite.chainA.GetContext(), chanCap, host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				suite.Require().NoError(err)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			// mock init interchain account
			portID, err := types.GeneratePortID(TestOwnerAddress, path.EndpointA.ConnectionID, path.EndpointB.ConnectionID)
			suite.Require().NoError(err)
			portCap := suite.chainA.GetSimApp().IBCKeeper.PortKeeper.BindPort(suite.chainA.GetContext(), portID)
			suite.chainA.GetSimApp().ICAKeeper.ClaimCapability(suite.chainA.GetContext(), portCap, host.PortPath(portID))
			path.EndpointA.ChannelConfig.PortID = portID

			// default values
			counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			channel = &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.ORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{path.EndpointA.ConnectionID},
				Version:        types.VersionPrefix,
			}

			chanCap, err = suite.chainA.App.GetScopedIBCKeeper().NewCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(portID, path.EndpointA.ChannelID))
			suite.Require().NoError(err)

			tc.malleate() // explicitly change fields in channel and testChannel

			err = suite.chainA.GetSimApp().ICAKeeper.OnChanOpenInit(suite.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, chanCap, channel.Counterparty, channel.GetVersion(),
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

		})
	}
}

func (suite *KeeperTestSuite) TestOnChanOpenTry() {
	var (
		channel             *channeltypes.Channel
		path                *ibctesting.Path
		chanCap             *capabilitytypes.Capability
		counterpartyVersion string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{

		{
			"success",
			func() {
				path.EndpointB.SetChannel(*channel)
			},
			true,
		},
		{
			"invalid order - UNORDERED",
			func() {
				channel.Ordering = channeltypes.UNORDERED
			},
			false,
		},
		{
			"invalid port",
			func() {
				path.EndpointB.ChannelConfig.PortID = "invalid-port-id"
			},
			false,
		},
		{
			"invalid counterparty port",
			func() {
				channel.Counterparty.PortId = "invalid-port-id"
			},
			false,
		},
		{
			"channel not found",
			func() {
				path.EndpointB.ChannelID = "invalid-channel-id"
			},
			false,
		},
		{
			"connection not found",
			func() {
				channel.ConnectionHops = []string{"invalid-connnection-id"}
				path.EndpointB.SetChannel(*channel)
			},
			false,
		},
		{
			"invalid connection sequence",
			func() {
				portID, err := types.GeneratePortID(TestOwnerAddress, "connection-0", "connection-1")
				suite.Require().NoError(err)

				channel.Counterparty.PortId = portID
				path.EndpointB.SetChannel(*channel)
			},
			false,
		},
		{
			"invalid counterparty connection sequence",
			func() {
				portID, err := types.GeneratePortID(TestOwnerAddress, "connection-1", "connection-0")
				suite.Require().NoError(err)

				channel.Counterparty.PortId = portID
				path.EndpointB.SetChannel(*channel)
			},
			false,
		},
		{
			"invalid version",
			func() {
				channel.Version = "version"
				path.EndpointB.SetChannel(*channel)
			},
			false,
		},
		{
			"invalid counterparty version",
			func() {
				counterpartyVersion = "version"
				path.EndpointB.SetChannel(*channel)
			},
			false,
		},
		{
			"capability already claimed",
			func() {
				path.EndpointB.SetChannel(*channel)
				err := suite.chainB.GetSimApp().ScopedICAKeeper.ClaimCapability(suite.chainB.GetContext(), chanCap, host.ChannelCapabilityPath(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
				suite.Require().NoError(err)
			},
			false,
		},
		{
			"invalid account address",
			func() {
				portID, err := types.GeneratePortID("invalid-owner-addr", "connection-0", "connection-0")
				suite.Require().NoError(err)

				channel.Counterparty.PortId = portID
				path.EndpointB.SetChannel(*channel)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			counterpartyVersion = types.VersionPrefix
			suite.coordinator.SetupConnections(path)

			err := InitInterchainAccount(path.EndpointA, TestOwnerAddress)
			suite.Require().NoError(err)

			// set the channel id on host
			channelSequence := path.EndpointB.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(path.EndpointB.Chain.GetContext())
			path.EndpointB.ChannelID = channeltypes.FormatChannelIdentifier(channelSequence)

			// default values
			counterparty := channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channel = &channeltypes.Channel{
				State:          channeltypes.TRYOPEN,
				Ordering:       channeltypes.ORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{path.EndpointB.ConnectionID},
				Version:        TestVersion,
			}

			chanCap, err = suite.chainB.App.GetScopedIBCKeeper().NewCapability(suite.chainB.GetContext(), host.ChannelCapabilityPath(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
			suite.Require().NoError(err)

			tc.malleate() // explicitly change fields in channel and testChannel

			err = suite.chainB.GetSimApp().ICAKeeper.OnChanOpenTry(suite.chainB.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, chanCap, channel.Counterparty, channel.GetVersion(),
				counterpartyVersion,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

		})
	}
}

// ChainA is controller, ChainB is host chain
func (suite *KeeperTestSuite) TestOnChanOpenAck() {
	var (
		path                *ibctesting.Path
		expectedChannel     string
		counterpartyVersion string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"invalid counterparty version", func() {
				expectedChannel = ""
				counterpartyVersion = "version"
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			counterpartyVersion = TestVersion
			suite.coordinator.SetupConnections(path)

			err := InitInterchainAccount(path.EndpointA, TestOwnerAddress)
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)
			expectedChannel = path.EndpointA.ChannelID

			tc.malleate() // explicitly change fields in channel and testChannel

			err = suite.chainA.GetSimApp().ICAKeeper.OnChanOpenAck(suite.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, counterpartyVersion,
			)

			activeChannel, _ := suite.chainA.GetSimApp().ICAKeeper.GetActiveChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)

			suite.Require().Equal(activeChannel, expectedChannel)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// ChainA is controller, ChainB is host chain
func (suite *KeeperTestSuite) TestOnChanOpenConfirm() {
	var (
		path *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{

		{
			"success", func() {}, true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := InitInterchainAccount(path.EndpointA, TestOwnerAddress)
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			suite.Require().NoError(err)

			tc.malleate() // explicitly change fields in channel and testChannel

			err = suite.chainB.GetSimApp().ICAKeeper.OnChanOpenConfirm(suite.chainB.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

		})
	}
}

func (suite *KeeperTestSuite) TestOnChanCloseConfirm() {
	var (
		path *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{

		{
			"success", func() {}, true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			tc.malleate() // explicitly change fields in channel and testChannel

			err = suite.chainB.GetSimApp().ICAKeeper.OnChanCloseConfirm(suite.chainB.GetContext(),
				path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			activeChannel, found := suite.chainB.GetSimApp().ICAKeeper.GetActiveChannel(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().False(found)
				suite.Require().Empty(activeChannel)
			} else {
				suite.Require().Error(err)
			}

		})
	}
}
