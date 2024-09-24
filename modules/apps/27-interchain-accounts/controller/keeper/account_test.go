package keeper_test

import (
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *KeeperTestSuite) TestRegisterInterchainAccount() {
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
					suite.Require().NoError(err)

					channelID := channeltypes.FormatChannelIdentifier(suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetNextChannelSequence(suite.chainA.GetContext()))
					path.EndpointA.ChannelID = channelID

					suite.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(suite.chainA.GetContext(), ibctesting.FirstConnectionID, portID, path.EndpointA.ChannelID)

					counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
					channel := channeltypes.Channel{
						State:          channeltypes.OPEN,
						Ordering:       ordering,
						Counterparty:   counterparty,
						ConnectionHops: []string{path.EndpointA.ConnectionID},
						Version:        TestVersion,
					}
					suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(suite.chainA.GetContext(), portID, path.EndpointA.ChannelID, channel)
				},
				icatypes.ErrActiveChannelAlreadySet,
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				suite.SetupTest()

				owner = TestOwnerAddress // must be explicitly changed

				path = NewICAPath(suite.chainA, suite.chainB, ordering)
				path.SetupConnections()

				tc.malleate() // malleate mutates test data

				err = suite.chainA.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(suite.chainA.GetContext(), path.EndpointA.ConnectionID, owner, TestVersion, ordering)

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().ErrorIs(err, tc.expErr)
				}
			})
		}
	}
}

func (suite *KeeperTestSuite) TestRegisterSameOwnerMultipleConnections() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		suite.SetupTest()

		owner := TestOwnerAddress

		pathAToB := NewICAPath(suite.chainA, suite.chainB, ordering)
		pathAToB.SetupConnections()

		pathAToC := NewICAPath(suite.chainA, suite.chainC, ordering)
		pathAToC.SetupConnections()

		// build ICS27 metadata with connection identifiers for path A->B
		metadata := &icatypes.Metadata{
			Version:                icatypes.Version,
			ControllerConnectionId: pathAToB.EndpointA.ConnectionID,
			HostConnectionId:       pathAToB.EndpointB.ConnectionID,
			Encoding:               icatypes.EncodingProtobuf,
			TxType:                 icatypes.TxTypeSDKMultiMsg,
		}

		err := suite.chainA.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(suite.chainA.GetContext(), pathAToB.EndpointA.ConnectionID, owner, string(icatypes.ModuleCdc.MustMarshalJSON(metadata)), ordering)
		suite.Require().NoError(err)

		// build ICS27 metadata with connection identifiers for path A->C
		metadata = &icatypes.Metadata{
			Version:                icatypes.Version,
			ControllerConnectionId: pathAToC.EndpointA.ConnectionID,
			HostConnectionId:       pathAToC.EndpointB.ConnectionID,
			Encoding:               icatypes.EncodingProtobuf,
			TxType:                 icatypes.TxTypeSDKMultiMsg,
		}

		err = suite.chainA.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(suite.chainA.GetContext(), pathAToC.EndpointA.ConnectionID, owner, string(icatypes.ModuleCdc.MustMarshalJSON(metadata)), ordering)
		suite.Require().NoError(err)
	}
}
