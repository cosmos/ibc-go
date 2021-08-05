package keeper_test

import (
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func (suite *KeeperTestSuite) TestOnChanOpenInit() {
	var (
		channel *channeltypes.Channel
		path    *ibctesting.Path
		chanCap *capabilitytypes.Capability
		err     error
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
			"invalid order - UNORDERED", func() {
				channel.Ordering = channeltypes.UNORDERED
			}, false,
		},
		{
			"invalid counterparty port ID", func() {
				channel.Counterparty.PortId = ibctesting.MockPort
			}, false,
		},
		{
			"invalid version", func() {
				channel.Version = "version"
			}, false,
		},
		{
			"channel is already active", func() {
				suite.chainA.GetSimApp().ICAKeeper.SetActiveChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			}, false,
		},
		{
			"capability already claimed", func() {
				err := suite.chainA.GetSimApp().ScopedICAKeeper.ClaimCapability(suite.chainA.GetContext(), chanCap, host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				suite.Require().NoError(err)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			// mock init interchain account
			portID := suite.chainA.GetSimApp().ICAKeeper.GeneratePortId("owner", path.EndpointA.ConnectionID)
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
				Version:        types.Version,
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

// ChainA is controller, ChainB is host chain
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
			"success", func() {}, true,
		},
		{
			"invalid order - UNORDERED", func() {
				channel.Ordering = channeltypes.UNORDERED
			}, false,
		},
		{
			"invalid version", func() {
				channel.Version = "version"
			}, false,
		},
		{
			"invalid counterparty version", func() {
				counterpartyVersion = "version"
			}, false,
		},
		{
			"capability already claimed", func() {
				err := suite.chainB.GetSimApp().ScopedICAKeeper.ClaimCapability(suite.chainB.GetContext(), chanCap, host.ChannelCapabilityPath(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
				suite.Require().NoError(err)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			owner := "owner"
			counterpartyVersion = types.Version
			suite.coordinator.SetupConnections(path)

			err := InitInterchainAccount(path.EndpointA, owner)
			suite.Require().NoError(err)

			// default values
			counterparty := channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channel = &channeltypes.Channel{
				State:          channeltypes.TRYOPEN,
				Ordering:       channeltypes.ORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{path.EndpointB.ConnectionID},
				Version:        types.Version,
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
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			owner := "owner"
			counterpartyVersion = types.Version
			suite.coordinator.SetupConnections(path)

			err := InitInterchainAccount(path.EndpointA, owner)
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			tc.malleate() // explicitly change fields in channel and testChannel

			err = suite.chainA.GetSimApp().ICAKeeper.OnChanOpenAck(suite.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, counterpartyVersion,
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
			owner := "owner"
			suite.coordinator.SetupConnections(path)

			err := InitInterchainAccount(path.EndpointA, owner)
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			suite.Require().NoError(err)

			tc.malleate() // explicitly change fields in channel and testChannel

			err = suite.chainB.GetSimApp().ICAKeeper.OnChanOpenConfirm(suite.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

		})
	}
}
