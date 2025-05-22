package keeper_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	hosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// open and close channel is a helper function for TestOnChanOpenTry for reopening accounts
func (suite *KeeperTestSuite) openAndCloseChannel(path *ibctesting.Path) {
	err := path.EndpointB.ChanOpenTry()
	suite.Require().NoError(err)

	err = path.EndpointA.ChanOpenAck()
	suite.Require().NoError(err)

	err = path.EndpointB.ChanOpenConfirm()
	suite.Require().NoError(err)

	path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
	path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })

	path.EndpointA.ChannelID = ""
	err = RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
	suite.Require().NoError(err)

	// bump channel sequence as these test mock core IBC behaviour on ChanOpenTry
	channelSequence := path.EndpointB.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(path.EndpointB.Chain.GetContext())
	path.EndpointB.ChannelID = channeltypes.FormatChannelIdentifier(channelSequence)
}

func (suite *KeeperTestSuite) TestOnChanOpenTry() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		var (
			channel  *channeltypes.Channel
			path     *ibctesting.Path
			metadata icatypes.Metadata
		)

		testCases := []struct {
			name     string
			malleate func()
			expErr   error
		}{
			{
				"success",
				func() {},
				nil,
			},
			{
				"success - reopening closed active channel",
				func() {
					// create interchain account
					// undo setup
					path.EndpointB.ChannelID = ""

					suite.openAndCloseChannel(path)
				},
				nil,
			},
			{
				"success - reopening account with new address",
				func() {
					// create interchain account
					// undo setup
					path.EndpointB.ChannelID = ""

					suite.openAndCloseChannel(path)

					// delete interchain account address
					store := suite.chainB.GetContext().KVStore(suite.chainB.GetSimApp().GetKey(hosttypes.SubModuleName))
					store.Delete(icatypes.KeyOwnerAccount(path.EndpointA.ChannelConfig.PortID, path.EndpointB.ConnectionID))

					// assert interchain account address mapping was deleted
					_, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
					suite.Require().False(found)
				},
				nil,
			},
			{
				"success - empty host connection ID",
				func() {
					metadata.HostConnectionId = ""

					versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
					suite.Require().NoError(err)

					path.EndpointA.ChannelConfig.Version = string(versionBytes)
				},
				nil,
			},
			{
				"success - previous metadata is different",
				func() {
					// set the active channelID in state
					suite.chainB.GetSimApp().ICAHostKeeper.SetActiveChannelID(suite.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID, path.EndpointB.ChannelID)

					// set the previous encoding to be proto3json.
					// the new encoding is set to be protobuf in the test below.
					metadata.Encoding = icatypes.EncodingProto3JSON

					versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
					suite.Require().NoError(err)

					channel.State = channeltypes.CLOSED
					channel.Version = string(versionBytes)

					path.EndpointB.SetChannel(*channel)
				}, nil,
			},
			{
				"invalid metadata bytestring",
				func() {
					// the try step will propose a new valid version
					path.EndpointA.ChannelConfig.Version = "invalid-metadata-bytestring"
				},
				nil,
			},
			{
				"reopening account fails - no existing account",
				func() {
					// create interchain account
					// undo setup
					path.EndpointB.ChannelID = ""

					suite.openAndCloseChannel(path)

					// delete existing account
					addr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
					suite.Require().True(found)

					acc := suite.chainB.GetSimApp().AccountKeeper.GetAccount(suite.chainB.GetContext(), sdk.MustAccAddressFromBech32(addr))
					suite.chainB.GetSimApp().AccountKeeper.RemoveAccount(suite.chainB.GetContext(), acc)
				},
				icatypes.ErrInvalidAccountReopening,
			},
			{
				"reopening account fails - existing account is not interchain account type",
				func() {
					// create interchain account
					// undo setup
					path.EndpointB.ChannelID = ""

					suite.openAndCloseChannel(path)

					addr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
					suite.Require().True(found)

					accAddress := sdk.MustAccAddressFromBech32(addr)
					acc := suite.chainB.GetSimApp().AccountKeeper.GetAccount(suite.chainB.GetContext(), accAddress)

					icaAcc, ok := acc.(*icatypes.InterchainAccount)
					suite.Require().True(ok)

					// overwrite existing account with only base account type, not intercahin account type
					suite.chainB.GetSimApp().AccountKeeper.SetAccount(suite.chainB.GetContext(), icaAcc.BaseAccount)
				},
				icatypes.ErrInvalidAccountReopening,
			},
			{
				"account already exists",
				func() {
					interchainAccAddr := icatypes.GenerateAddress(suite.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
					err := suite.chainB.GetSimApp().BankKeeper.SendCoins(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), interchainAccAddr, sdk.Coins{sdk.NewCoin("stake", sdkmath.NewInt(1))})
					suite.Require().NoError(err)
					suite.Require().True(suite.chainB.GetSimApp().AccountKeeper.HasAccount(suite.chainB.GetContext(), interchainAccAddr))
				},
				icatypes.ErrAccountAlreadyExist,
			},
			{
				"invalid port ID",
				func() {
					path.EndpointB.ChannelConfig.PortID = "invalid-port-id" //nolint:goconst
				},
				icatypes.ErrInvalidHostPort,
			},
			{
				"connection not found",
				func() {
					channel.ConnectionHops = []string{ibctesting.InvalidID}
					path.EndpointB.SetChannel(*channel)
				},
				connectiontypes.ErrConnectionNotFound,
			},
			{
				"unsupported encoding format",
				func() {
					metadata.Encoding = "invalid-encoding-format"

					versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
					suite.Require().NoError(err)

					path.EndpointA.ChannelConfig.Version = string(versionBytes)
				},
				icatypes.ErrInvalidCodec,
			},
			{
				"unsupported transaction type",
				func() {
					metadata.TxType = "invalid-tx-types"

					versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
					suite.Require().NoError(err)

					path.EndpointA.ChannelConfig.Version = string(versionBytes)
				},
				icatypes.ErrUnknownDataType,
			},
			{
				"invalid controller connection ID",
				func() {
					metadata.ControllerConnectionId = ibctesting.InvalidID

					versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
					suite.Require().NoError(err)

					path.EndpointA.ChannelConfig.Version = string(versionBytes)
				},
				connectiontypes.ErrInvalidConnection,
			},
			{
				"invalid counterparty version",
				func() {
					metadata.Version = "invalid-version"

					versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
					suite.Require().NoError(err)

					path.EndpointA.ChannelConfig.Version = string(versionBytes)
				},
				icatypes.ErrInvalidVersion,
			},
			{
				"active channel already set (OPEN state)",
				func() {
					// create a new channel and set it in state
					ch := channeltypes.NewChannel(channeltypes.OPEN, ordering, channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointA.ConnectionID}, ibctesting.DefaultChannelVersion)
					suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, ch)

					// set the active channelID in state
					suite.chainB.GetSimApp().ICAHostKeeper.SetActiveChannelID(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, path.EndpointB.ChannelID)
				},
				icatypes.ErrActiveChannelAlreadySet,
			},
		}

		for _, tc := range testCases {
			suite.Run(tc.name, func() {
				suite.SetupTest() // reset

				path = NewICAPath(suite.chainA, suite.chainB, icatypes.EncodingProtobuf, ordering)
				path.SetupConnections()

				err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
				suite.Require().NoError(err)

				// set the channel id on host
				channelSequence := path.EndpointB.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(path.EndpointB.Chain.GetContext())
				path.EndpointB.ChannelID = channeltypes.FormatChannelIdentifier(channelSequence)

				// default values
				metadata = icatypes.NewMetadata(icatypes.Version, ibctesting.FirstConnectionID, ibctesting.FirstConnectionID, "", icatypes.EncodingProtobuf, icatypes.TxTypeSDKMultiMsg)
				versionBytes, err := icatypes.ModuleCdc.MarshalJSON(&metadata)
				suite.Require().NoError(err)

				expectedMetadata := metadata

				counterparty := channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				channel = &channeltypes.Channel{
					State:          channeltypes.TRYOPEN,
					Ordering:       ordering,
					Counterparty:   counterparty,
					ConnectionHops: []string{path.EndpointB.ConnectionID},
					Version:        string(versionBytes),
				}

				tc.malleate() // malleate mutates test data

				version, err := suite.chainB.GetSimApp().ICAHostKeeper.OnChanOpenTry(suite.chainB.GetContext(), channel.Ordering, channel.ConnectionHops,
					path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, channel.Counterparty, path.EndpointA.ChannelConfig.Version,
				)

				if tc.expErr == nil {
					suite.Require().NoError(err)

					storedAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
					suite.Require().True(found)

					interchainAccAddr, err := sdk.AccAddressFromBech32(storedAddr)
					suite.Require().NoError(err)

					// Check if account is created
					interchainAccount := suite.chainB.GetSimApp().AccountKeeper.GetAccount(suite.chainB.GetContext(), interchainAccAddr)
					suite.Require().Equal(interchainAccount.GetAddress().String(), storedAddr)

					expectedMetadata.Address = storedAddr
					expectedVersionBytes, err := icatypes.ModuleCdc.MarshalJSON(&expectedMetadata)
					suite.Require().NoError(err)

					suite.Require().Equal(string(expectedVersionBytes), version)
				} else {
					suite.Require().ErrorIs(err, tc.expErr)
					suite.Require().Equal("", version)
				}
			})
		}
	}
}

func (suite *KeeperTestSuite) TestOnChanOpenConfirm() {
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success", func() {}, nil,
		},
		{
			"channel not found",
			func() {
				path.EndpointB.ChannelID = "invalid-channel-id"
				path.EndpointB.ChannelConfig.PortID = "invalid-port-id" //nolint:goconst
			},
			channeltypes.ErrChannelNotFound,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			suite.Run(tc.name, func() {
				suite.SetupTest() // reset

				path = NewICAPath(suite.chainA, suite.chainB, icatypes.EncodingProtobuf, ordering)
				path.SetupConnections()

				err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
				suite.Require().NoError(err)

				err = path.EndpointB.ChanOpenTry()
				suite.Require().NoError(err)

				err = path.EndpointA.ChanOpenAck()
				suite.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				err = suite.chainB.GetSimApp().ICAHostKeeper.OnChanOpenConfirm(suite.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().ErrorIs(err, tc.expErr)
				}
			})
		}
	}
}

func (suite *KeeperTestSuite) TestOnChanCloseConfirm() {
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
			suite.Run(tc.name, func() {
				suite.SetupTest() // reset

				path = NewICAPath(suite.chainA, suite.chainB, icatypes.EncodingProtobuf, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				suite.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				err = suite.chainB.GetSimApp().ICAHostKeeper.OnChanCloseConfirm(suite.chainB.GetContext(),
					path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().ErrorIs(err, tc.expErr)
				}
			})
		}
	}
}
