package keeper_test

import (
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestRegisterInterchainAccount() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		var (
			owner string
			path  *ibctesting.Path
			err   error
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
				"fails to generate port-id",
				func() {
					owner = ""
				},
				icatypes.ErrInvalidAccountAddress,
			},
			{
				"MsgChanOpenInit fails - channel is already active & in state OPEN",
				func() {
					portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
					s.Require().NoError(err)

					channelID := channeltypes.FormatChannelIdentifier(s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetNextChannelSequence(s.chainA.GetContext()))
					path.EndpointA.ChannelID = channelID

					s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, portID, path.EndpointA.ChannelID)

					counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
					channel := channeltypes.Channel{
						State:          channeltypes.OPEN,
						Ordering:       ordering,
						Counterparty:   counterparty,
						ConnectionHops: []string{path.EndpointA.ConnectionID},
						Version:        TestVersion,
					}
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), portID, path.EndpointA.ChannelID, channel)
				},
				icatypes.ErrActiveChannelAlreadySet,
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest()

				owner = TestOwnerAddress // must be explicitly changed

				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				tc.malleate() // malleate mutates test data

				err = s.chainA.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(s.chainA.GetContext(), path.EndpointA.ConnectionID, owner, TestVersion, ordering)

				if tc.expErr == nil {
					s.Require().NoError(err)
				} else {
					s.Require().ErrorIs(err, tc.expErr)
				}
			})
		}
	}
}

func (s *KeeperTestSuite) TestRegisterSameOwnerMultipleConnections() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest()

		owner := TestOwnerAddress

		pathAToB := NewICAPath(s.chainA, s.chainB, ordering)
		pathAToB.SetupConnections()

		pathAToC := NewICAPath(s.chainA, s.chainC, ordering)
		pathAToC.SetupConnections()

		// build ICS27 metadata with connection identifiers for path A->B
		metadata := &icatypes.Metadata{
			Version:                icatypes.Version,
			ControllerConnectionId: pathAToB.EndpointA.ConnectionID,
			HostConnectionId:       pathAToB.EndpointB.ConnectionID,
			Encoding:               icatypes.EncodingProtobuf,
			TxType:                 icatypes.TxTypeSDKMultiMsg,
		}

		err := s.chainA.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(s.chainA.GetContext(), pathAToB.EndpointA.ConnectionID, owner, string(icatypes.ModuleCdc.MustMarshalJSON(metadata)), ordering)
		s.Require().NoError(err)

		// build ICS27 metadata with connection identifiers for path A->C
		metadata = &icatypes.Metadata{
			Version:                icatypes.Version,
			ControllerConnectionId: pathAToC.EndpointA.ConnectionID,
			HostConnectionId:       pathAToC.EndpointB.ConnectionID,
			Encoding:               icatypes.EncodingProtobuf,
			TxType:                 icatypes.TxTypeSDKMultiMsg,
		}

		err = s.chainA.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(s.chainA.GetContext(), pathAToC.EndpointA.ConnectionID, owner, string(icatypes.ModuleCdc.MustMarshalJSON(metadata)), ordering)
		s.Require().NoError(err)
	}
}
