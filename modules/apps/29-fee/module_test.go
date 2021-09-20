package fee_test

import (
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func (suite *FeeTestSuite) TestOnChanOpenInit() {
	testCases := []struct {
		name    string
		version string
		expPass bool
	}{
		{
			"valid fee middleware and transfer version",
			"fee29-1:ics20-1",
			true,
		},
		{
			"fee version not included, only perform transfer logic",
			"ics20-1",
			true,
		},
		{
			"invalid fee middleware version",
			"otherfee28-1:ics20-1",
			false,
		},
		{
			"invalid transfer version",
			"fee29-1:wrongics20-1",
			false,
		},
		{
			"incorrect wrapping delimiter",
			"fee29-1//ics20-1",
			false,
		},
		{
			"transfer version not wrapped",
			"fee29-1",
			false,
		},
		{
			"hanging delimiter",
			"fee29-1:ics20-1:",
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupTest()
			suite.coordinator.SetupClients(suite.path)
			suite.coordinator.SetupConnections(suite.path)

			suite.path.EndpointA.ChannelID = ibctesting.FirstChannelID

			counterparty := channeltypes.NewCounterparty(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)
			channel := &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{suite.path.EndpointA.ConnectionID},
				Version:        tc.version,
			}

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			chanCap, err := suite.chainA.App.GetScopedIBCKeeper().NewCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(ibctesting.TransferPort, suite.path.EndpointA.ChannelID))
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			err = cbs.OnChanOpenInit(suite.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, chanCap, counterparty, channel.Version)

			if tc.expPass {
				suite.Require().NoError(err, "unexpected error from version: %s", tc.version)
			} else {
				suite.Require().Error(err, "error not returned for version: %s", tc.version)
			}
		})
	}
}

func (suite *FeeTestSuite) TestOnChanOpenTry() {
	testCases := []struct {
		name      string
		version   string
		cpVersion string
		capExists bool
		expPass   bool
	}{
		{
			"valid fee middleware and transfer version",
			"fee29-1:ics20-1",
			"fee29-1:ics20-1",
			false,
			true,
		},
		{
			"valid transfer version on try and counterparty",
			"ics20-1",
			"ics20-1",
			false,
			true,
		},
		{
			"valid fee middleware and transfer version, crossing hellos",
			"fee29-1:ics20-1",
			"fee29-1:ics20-1",
			true,
			true,
		},
		{
			"invalid fee middleware version",
			"otherfee28-1:ics20-1",
			"fee29-1:ics20-1",
			false,
			false,
		},
		{
			"invalid counterparty fee middleware version",
			"fee29-1:ics20-1",
			"wrongfee29-1:ics20-1",
			false,
			false,
		},
		{
			"invalid transfer version",
			"fee29-1:wrongics20-1",
			"fee29-1:ics20-1",
			false,
			false,
		},
		{
			"invalid counterparty transfer version",
			"fee29-1:ics20-1",
			"fee29-1:wrongics20-1",
			false,
			false,
		},
		{
			"transfer version not wrapped",
			"fee29-1",
			"fee29-1:ics20-1",
			false,
			false,
		},
		{
			"counterparty transfer version not wrapped",
			"fee29-1:ics20-1",
			"fee29-1",
			false,
			false,
		},
		{
			"fee version not included on try, but included in counterparty",
			"ics20-1",
			"fee29-1:ics20-1",
			false,
			false,
		},
		{
			"transfer version not included",
			"fee29-1:ics20-1",
			"ics20-1",
			false,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupTest()
			suite.coordinator.SetupClients(suite.path)
			suite.coordinator.SetupConnections(suite.path)
			suite.path.EndpointB.ChanOpenInit()

			var (
				chanCap *capabilitytypes.Capability
				ok      bool
				err     error
			)
			if tc.capExists {
				suite.path.EndpointA.ChanOpenInit()
				chanCap, ok = suite.chainA.GetSimApp().ScopedTransferKeeper.GetCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(ibctesting.TransferPort, suite.path.EndpointA.ChannelID))
				suite.Require().True(ok)
			} else {
				chanCap, err = suite.chainA.App.GetScopedIBCKeeper().NewCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(ibctesting.TransferPort, suite.path.EndpointA.ChannelID))
				suite.Require().NoError(err)
			}

			suite.path.EndpointA.ChannelID = ibctesting.FirstChannelID

			counterparty := channeltypes.NewCounterparty(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)
			channel := &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{suite.path.EndpointA.ConnectionID},
				Version:        tc.version,
			}

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			err = cbs.OnChanOpenTry(suite.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, chanCap, counterparty, tc.version, tc.cpVersion)

			if tc.expPass {
				suite.Require().NoError(err, "unexpected error from version: %s", tc.version)
			} else {
				suite.Require().Error(err, "error not returned for version: %s", tc.version)
			}
		})
	}
}

func (suite *FeeTestSuite) TestOnChanOpenAck() {
	testCases := []struct {
		name      string
		cpVersion string
		expPass   bool
	}{
		{
			"success",
			"fee29-1:ics20-1",
			true,
		},
		{
			"invalid fee version",
			"fee29-3:ics20-1",
			false,
		},
		{
			"invalid transfer version",
			"fee29-1:ics20-4",
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.SetupClients(suite.path)
			suite.coordinator.SetupConnections(suite.path)

			suite.path.EndpointA.ChanOpenInit()
			suite.path.EndpointB.ChanOpenTry()

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			err = cbs.OnChanOpenAck(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, tc.cpVersion)
			if tc.expPass {
				suite.Require().NoError(err, "unexpected error for case: %s", tc.name)
			} else {
				suite.Require().Error(err, "%s expected error but returned none", tc.name)
			}
		})
	}
}
