package keeper_test

import (
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestOnChanOpenInit() {
	var (
		channel  *channeltypes.Channel
		path     *ibctesting.Path
		chanCap  *capabilitytypes.Capability
		metadata icatypes.Metadata
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
			"success: previous active channel closed",
			func() {
				s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				channel := channeltypes.Channel{
					State:          channeltypes.CLOSED,
					Ordering:       channeltypes.ORDERED,
					Counterparty:   counterparty,
					ConnectionHops: []string{path.EndpointA.ConnectionID},
					Version:        TestVersion,
				}

				path.EndpointA.SetChannel(channel)
			},
			true,
		},
		{
			"success: empty channel version returns default metadata JSON string",
			func() {
				channel.Version = ""
			},
			true,
		},
		{
			"success: channel reopening",
			func() {
				err := SetupICAPath(path, TestOwnerAddress)
				s.Require().NoError(err)

				err = path.EndpointA.SetChannelState(channeltypes.CLOSED)
				s.Require().NoError(err)

				err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
				s.Require().NoError(err)

				path.EndpointA.ChannelID = ""
				path.EndpointB.ChannelID = ""
			},
			true,
		},
		{
			"invalid metadata -  previous metadata is different",
			func() {
				// set active channel to closed
				s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				// attempt to downgrade version by reinitializing channel with version 1, but setting channel to version 2
				metadata.Version = "ics27-2"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				closedChannel := channeltypes.Channel{
					State:          channeltypes.CLOSED,
					Ordering:       channeltypes.ORDERED,
					Counterparty:   counterparty,
					ConnectionHops: []string{path.EndpointA.ConnectionID},
					Version:        string(versionBytes),
				}
				path.EndpointA.SetChannel(closedChannel)
			},
			false,
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
				path.EndpointA.ChannelConfig.PortID = "invalid-port-id" //nolint:goconst
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
			"invalid metadata bytestring",
			func() {
				path.EndpointA.SetChannel(*channel)
				channel.Version = "invalid-metadata-bytestring"
			},
			false,
		},
		{
			"unsupported encoding format",
			func() {
				metadata.Encoding = "invalid-encoding-format"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				channel.Version = string(versionBytes)
				path.EndpointA.SetChannel(*channel)
			},
			false,
		},
		{
			"unsupported transaction type",
			func() {
				metadata.TxType = "invalid-tx-types"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				channel.Version = string(versionBytes)
				path.EndpointA.SetChannel(*channel)
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
			"connection not found with default empty channel version",
			func() {
				channel.ConnectionHops = []string{"connection-10"}
				channel.Version = ""
			},
			false,
		},
		{
			"invalid controller connection ID",
			func() {
				metadata.ControllerConnectionId = "invalid-connnection-id"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				channel.Version = string(versionBytes)
				path.EndpointA.SetChannel(*channel)
			},
			false,
		},
		{
			"invalid host connection ID",
			func() {
				metadata.HostConnectionId = "invalid-connnection-id"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				channel.Version = string(versionBytes)
				path.EndpointA.SetChannel(*channel)
			},
			false,
		},
		{
			"invalid version",
			func() {
				metadata.Version = "invalid-version"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				channel.Version = string(versionBytes)
				path.EndpointA.SetChannel(*channel)
			},
			false,
		},
		{
			"channel is already active",
			func() {
				s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				channel := channeltypes.Channel{
					State:          channeltypes.OPEN,
					Ordering:       channeltypes.ORDERED,
					Counterparty:   counterparty,
					ConnectionHops: []string{path.EndpointA.ConnectionID},
					Version:        TestVersion,
				}
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			// mock init interchain account
			portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
			s.Require().NoError(err)

			portCap := s.chainA.GetSimApp().IBCKeeper.PortKeeper.BindPort(s.chainA.GetContext(), portID)
			s.chainA.GetSimApp().ICAControllerKeeper.ClaimCapability(s.chainA.GetContext(), portCap, host.PortPath(portID)) //nolint:errcheck // this error check isn't needed for tests
			path.EndpointA.ChannelConfig.PortID = portID

			// default values
			metadata = icatypes.NewMetadata(icatypes.Version, path.EndpointA.ConnectionID, path.EndpointB.ConnectionID, "", icatypes.EncodingProtobuf, icatypes.TxTypeSDKMultiMsg)
			versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
			s.Require().NoError(err)

			counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			channel = &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.ORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{path.EndpointA.ConnectionID},
				Version:        string(versionBytes),
			}

			chanCap, err = s.chainA.App.GetScopedIBCKeeper().NewCapability(s.chainA.GetContext(), host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			s.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			version, err := s.chainA.GetSimApp().ICAControllerKeeper.OnChanOpenInit(s.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, chanCap, channel.Counterparty, channel.Version,
			)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(string(versionBytes), version)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestOnChanOpenAck() {
	var (
		path     *ibctesting.Path
		metadata icatypes.Metadata
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
			"invalid port ID - host chain",
			func() {
				path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
			},
			false,
		},
		{
			"invalid port ID - unexpected prefix",
			func() {
				path.EndpointA.ChannelConfig.PortID = "invalid-port-id"
			},
			false,
		},
		{
			"invalid metadata bytestring",
			func() {
				path.EndpointA.Counterparty.ChannelConfig.Version = "invalid-metadata-bytestring"
			},
			false,
		},
		{
			"unsupported encoding format",
			func() {
				metadata.Encoding = "invalid-encoding-format"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				path.EndpointA.Counterparty.ChannelConfig.Version = string(versionBytes)
			},
			false,
		},
		{
			"unsupported transaction type",
			func() {
				metadata.TxType = "invalid-tx-types"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				path.EndpointA.Counterparty.ChannelConfig.Version = string(versionBytes)
			},
			false,
		},
		{
			"invalid account address",
			func() {
				metadata.Address = "invalid-account-address"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				path.EndpointA.Counterparty.ChannelConfig.Version = string(versionBytes)
			},
			false,
		},
		{
			"empty account address",
			func() {
				metadata.Address = ""

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				path.EndpointA.Counterparty.ChannelConfig.Version = string(versionBytes)
			},
			false,
		},
		{
			"invalid counterparty version",
			func() {
				metadata.Version = "invalid-version"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				path.EndpointA.Counterparty.ChannelConfig.Version = string(versionBytes)
			},
			false,
		},
		{
			"active channel already set",
			func() {
				// create a new channel and set it in state
				ch := channeltypes.NewChannel(channeltypes.OPEN, channeltypes.ORDERED, channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, ibctesting.DefaultChannelVersion)
				s.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ch)

				// set the active channelID in state
				s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			interchainAccAddr, exists := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
			s.Require().True(exists)

			metadata = icatypes.NewMetadata(icatypes.Version, ibctesting.FirstConnectionID, ibctesting.FirstConnectionID, interchainAccAddr, icatypes.EncodingProtobuf, icatypes.TxTypeSDKMultiMsg)
			versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
			s.Require().NoError(err)

			path.EndpointB.ChannelConfig.Version = string(versionBytes)

			tc.malleate() // malleate mutates test data

			err = s.chainA.GetSimApp().ICAControllerKeeper.OnChanOpenAck(s.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointA.Counterparty.ChannelConfig.Version,
			)

			if tc.expPass {
				s.Require().NoError(err)

				activeChannelID, found := s.chainA.GetSimApp().ICAControllerKeeper.GetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				s.Require().Equal(path.EndpointA.ChannelID, activeChannelID)

				interchainAccAddress, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				s.Require().Equal(metadata.Address, interchainAccAddress)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestOnChanCloseConfirm() {
	var path *ibctesting.Path

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
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			s.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			err = s.chainB.GetSimApp().ICAControllerKeeper.OnChanCloseConfirm(s.chainB.GetContext(),
				path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			activeChannelID, found := s.chainB.GetSimApp().ICAControllerKeeper.GetActiveChannelID(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointB.ChannelConfig.PortID)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().False(found)
				s.Require().Empty(activeChannelID)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
