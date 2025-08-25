package keeper_test

import (
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestOnChanOpenInit() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		var (
			channel         *channeltypes.Channel
			path            *ibctesting.Path
			metadata        icatypes.Metadata
			expectedVersion string
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
				"success: previous active channel closed",
				func() {
					s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

					counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
					channel := channeltypes.Channel{
						State:          channeltypes.CLOSED,
						Ordering:       ordering,
						Counterparty:   counterparty,
						ConnectionHops: []string{path.EndpointA.ConnectionID},
						Version:        TestVersion,
					}

					path.EndpointA.SetChannel(channel)
				},
				nil,
			},
			{
				"success: empty channel version returns default metadata JSON string",
				func() {
					channel.Version = ""
					expectedVersion = icatypes.NewDefaultMetadataString(path.EndpointA.ConnectionID, path.EndpointB.ConnectionID)
				},
				nil,
			},
			{
				"success: channel reopening",
				func() {
					err := SetupICAPath(path, TestOwnerAddress)
					s.Require().NoError(err)

					path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
					path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })

					path.EndpointA.ChannelID = ""
					path.EndpointB.ChannelID = ""
				},
				nil,
			},
			{
				"failure: different ordering from previous channel",
				func() {
					differentOrdering := channeltypes.UNORDERED
					if ordering == channeltypes.UNORDERED {
						differentOrdering = channeltypes.ORDERED
					}

					s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

					counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
					channel := channeltypes.Channel{
						State:          channeltypes.CLOSED,
						Ordering:       differentOrdering,
						Counterparty:   counterparty,
						ConnectionHops: []string{path.EndpointA.ConnectionID},
						Version:        TestVersion,
					}

					path.EndpointA.SetChannel(channel)
				},
				channeltypes.ErrInvalidChannelOrdering,
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
						Ordering:       ordering,
						Counterparty:   counterparty,
						ConnectionHops: []string{path.EndpointA.ConnectionID},
						Version:        string(versionBytes),
					}
					path.EndpointA.SetChannel(closedChannel)
				},
				icatypes.ErrInvalidVersion,
			},
			{
				"invalid port ID",
				func() {
					path.EndpointA.ChannelConfig.PortID = "invalid-port-id" //nolint:goconst
				},
				icatypes.ErrInvalidControllerPort,
			},
			{
				"invalid counterparty port ID",
				func() {
					path.EndpointA.SetChannel(*channel)
					channel.Counterparty.PortId = "invalid-port-id" //nolint:goconst
				},
				icatypes.ErrInvalidHostPort,
			},
			{
				"invalid metadata bytestring",
				func() {
					path.EndpointA.SetChannel(*channel)
					channel.Version = "invalid-metadata-bytestring"
				},
				ibcerrors.ErrInvalidType,
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
				icatypes.ErrInvalidCodec,
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
				icatypes.ErrUnknownDataType,
			},
			{
				"connection not found",
				func() {
					channel.ConnectionHops = []string{ibctesting.InvalidID}
					path.EndpointA.SetChannel(*channel)
				},
				connectiontypes.ErrConnectionNotFound,
			},
			{
				"connection not found with default empty channel version",
				func() {
					channel.ConnectionHops = []string{"connection-10"}
					channel.Version = ""
				},
				connectiontypes.ErrConnectionNotFound,
			},
			{
				"invalid controller connection ID",
				func() {
					metadata.ControllerConnectionId = ibctesting.InvalidID

					versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
					s.Require().NoError(err)

					channel.Version = string(versionBytes)
					path.EndpointA.SetChannel(*channel)
				},
				connectiontypes.ErrInvalidConnection,
			},
			{
				"invalid host connection ID",
				func() {
					metadata.HostConnectionId = ibctesting.InvalidID

					versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
					s.Require().NoError(err)

					channel.Version = string(versionBytes)
					path.EndpointA.SetChannel(*channel)
				},
				connectiontypes.ErrInvalidConnection,
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
				icatypes.ErrInvalidVersion,
			},
			{
				"channel is already active (OPEN state)",
				func() {
					s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

					counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
					channel := channeltypes.Channel{
						State:          channeltypes.OPEN,
						Ordering:       ordering,
						Counterparty:   counterparty,
						ConnectionHops: []string{path.EndpointA.ConnectionID},
						Version:        TestVersion,
					}
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)
				},
				icatypes.ErrActiveChannelAlreadySet,
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest() // reset

				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				// mock init interchain account
				portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
				s.Require().NoError(err)

				path.EndpointA.ChannelConfig.PortID = portID

				// default values
				metadata = icatypes.NewMetadata(icatypes.Version, path.EndpointA.ConnectionID, path.EndpointB.ConnectionID, "", icatypes.EncodingProtobuf, icatypes.TxTypeSDKMultiMsg)
				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				expectedVersion = string(versionBytes)

				counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				channel = &channeltypes.Channel{
					State:          channeltypes.INIT,
					Ordering:       ordering,
					Counterparty:   counterparty,
					ConnectionHops: []string{path.EndpointA.ConnectionID},
					Version:        string(versionBytes),
				}

				channelID := channeltypes.FormatChannelIdentifier(s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetNextChannelSequence(s.chainA.GetContext()))
				path.EndpointA.ChannelID = channelID

				tc.malleate() // malleate mutates test data

				version, err := s.chainA.GetSimApp().ICAControllerKeeper.OnChanOpenInit(s.chainA.GetContext(), channel.Ordering, channel.ConnectionHops,
					path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel.Counterparty, channel.Version,
				)

				if tc.expError == nil {
					s.Require().NoError(err)
					s.Require().Equal(expectedVersion, version)
				} else {
					s.Require().Error(err)
					s.Require().ErrorIs(err, tc.expError)
				}
			})
		}
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
		expErr   error
	}{
		{
			"success", func() {}, nil,
		},
		{
			"invalid port ID - host chain",
			func() {
				path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
			},
			icatypes.ErrInvalidControllerPort,
		},
		{
			"invalid port ID - unexpected prefix",
			func() {
				path.EndpointA.ChannelConfig.PortID = "invalid-port-id" //nolint:goconst
			},
			icatypes.ErrInvalidControllerPort,
		},
		{
			"invalid metadata bytestring",
			func() {
				path.EndpointA.Counterparty.ChannelConfig.Version = "invalid-metadata-bytestring"
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"unsupported encoding format",
			func() {
				metadata.Encoding = "invalid-encoding-format"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				path.EndpointA.Counterparty.ChannelConfig.Version = string(versionBytes)
			},
			icatypes.ErrInvalidCodec,
		},
		{
			"unsupported transaction type",
			func() {
				metadata.TxType = "invalid-tx-types"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				path.EndpointA.Counterparty.ChannelConfig.Version = string(versionBytes)
			},
			icatypes.ErrUnknownDataType,
		},
		{
			"invalid account address",
			func() {
				metadata.Address = "invalid-account-address"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				path.EndpointA.Counterparty.ChannelConfig.Version = string(versionBytes)
			},
			icatypes.ErrInvalidAccountAddress,
		},
		{
			"empty account address",
			func() {
				metadata.Address = ""

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				path.EndpointA.Counterparty.ChannelConfig.Version = string(versionBytes)
			},
			icatypes.ErrInvalidAccountAddress,
		},
		{
			"invalid counterparty version",
			func() {
				metadata.Version = "invalid-version"

				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				s.Require().NoError(err)

				path.EndpointA.Counterparty.ChannelConfig.Version = string(versionBytes)
			},
			icatypes.ErrInvalidVersion,
		},
		{
			"active channel already set",
			func() {
				// create a new channel and set it in state
				ch := channeltypes.NewChannel(channeltypes.OPEN, channeltypes.ORDERED, channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, ibctesting.DefaultChannelVersion)
				s.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ch)

				// set the active channelID in state
				s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			}, icatypes.ErrActiveChannelAlreadySet,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest() // reset

				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

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

				if tc.expErr == nil {
					s.Require().NoError(err)

					activeChannelID, found := s.chainA.GetSimApp().ICAControllerKeeper.GetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
					s.Require().True(found)

					s.Require().Equal(path.EndpointA.ChannelID, activeChannelID)

					interchainAccAddress, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
					s.Require().True(found)

					s.Require().Equal(metadata.Address, interchainAccAddress)
				} else {
					s.Require().ErrorIs(err, tc.expErr)
				}
			})
		}
	}
}

func (s *KeeperTestSuite) TestOnChanCloseConfirm() {
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success", func() {}, nil,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest() // reset

				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				s.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				err = s.chainB.GetSimApp().ICAControllerKeeper.OnChanCloseConfirm(s.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

				activeChannelID, found := s.chainB.GetSimApp().ICAControllerKeeper.GetActiveChannelID(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointB.ChannelConfig.PortID)

				if tc.expErr == nil {
					s.Require().NoError(err)
					s.Require().False(found)
					s.Require().Empty(activeChannelID)
				} else {
					s.Require().Error(err)
				}
			})
		}
	}
}
