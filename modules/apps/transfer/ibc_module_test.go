package transfer_test

import (
	"encoding/json"
	"errors"
	"math"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *TransferTestSuite) TestOnChanOpenInit() {
	var (
		channel      *channeltypes.Channel
		path         *ibctesting.Path
		chanCap      *capabilitytypes.Capability
		counterparty channeltypes.Counterparty
	)

	testCases := []struct {
		name       string
		malleate   func()
		expError   error
		expVersion string
	}{
		{
			"success", func() {}, nil, types.V2,
		},
		{
			// connection hops is not used in the transfer application callback,
			// it is already validated in the core OnChanUpgradeInit.
			"success: invalid connection hops", func() {
				path.EndpointA.ConnectionID = "invalid-connection-id"
			}, nil, types.V2,
		},
		{
			"success: empty version string", func() {
				channel.Version = ""
			}, nil, types.V2,
		},
		{
			"success: ics20-1 legacy", func() {
				channel.Version = types.V1
			}, nil, types.V1,
		},
		{
			"max channels reached", func() {
				path.EndpointA.ChannelID = channeltypes.FormatChannelIdentifier(math.MaxUint32 + 1)
			}, types.ErrMaxTransferChannels, "",
		},
		{
			"invalid order - ORDERED", func() {
				channel.Ordering = channeltypes.ORDERED
			}, channeltypes.ErrInvalidChannelOrdering, "",
		},
		{
			"invalid port ID", func() {
				path.EndpointA.ChannelConfig.PortID = ibctesting.MockPort
			}, porttypes.ErrInvalidPort, "",
		},
		{
			"invalid version", func() {
				channel.Version = "version" //nolint:goconst
			}, types.ErrInvalidVersion, "",
		},
		{
			"capability already claimed", func() {
				err := suite.chainA.GetSimApp().ScopedTransferKeeper.ClaimCapability(suite.chainA.GetContext(), chanCap, host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				suite.Require().NoError(err)
			}, capabilitytypes.ErrOwnerClaimed, "",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.SetupConnections()
			path.EndpointA.ChannelID = ibctesting.FirstChannelID

			counterparty = channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			channel = &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{path.EndpointA.ConnectionID},
				Version:        types.V2,
			}

			var err error
			chanCap, err = suite.chainA.App.GetScopedIBCKeeper().NewCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(ibctesting.TransferPort, path.EndpointA.ChannelID))
			suite.Require().NoError(err)

			tc.malleate() // explicitly change fields in channel and testChannel

			transferModule := transfer.NewIBCModule(suite.chainA.GetSimApp().TransferKeeper)
			version, err := transferModule.OnChanOpenInit(suite.chainA.GetContext(), channel.Ordering, channel.ConnectionHops,
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, chanCap, counterparty, channel.Version,
			)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expVersion, version)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnChanOpenTry() {
	var (
		channel             *channeltypes.Channel
		chanCap             *capabilitytypes.Capability
		path                *ibctesting.Path
		counterparty        channeltypes.Counterparty
		counterpartyVersion string
	)

	testCases := []struct {
		name       string
		malleate   func()
		expError   error
		expVersion string
	}{
		{
			"success", func() {}, nil, types.V2,
		},
		{
			"success: counterparty version is legacy ics20-1", func() {
				counterpartyVersion = types.V1
			}, nil, types.V1,
		},
		{
			"success: invalid counterparty version, we propose new version", func() {
				// transfer module will propose the default version
				counterpartyVersion = "version"
			}, nil, types.V2,
		},
		{
			"failure: max channels reached", func() {
				path.EndpointA.ChannelID = channeltypes.FormatChannelIdentifier(math.MaxUint32 + 1)
			}, types.ErrMaxTransferChannels, "",
		},
		{
			"failure: capability already claimed", func() {
				err := suite.chainA.GetSimApp().ScopedTransferKeeper.ClaimCapability(suite.chainA.GetContext(), chanCap, host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				suite.Require().NoError(err)
			}, capabilitytypes.ErrOwnerClaimed, "",
		},
		{
			"failure: invalid order - ORDERED", func() {
				channel.Ordering = channeltypes.ORDERED
			}, channeltypes.ErrInvalidChannelOrdering, "",
		},
		{
			"failure: invalid port ID", func() {
				path.EndpointA.ChannelConfig.PortID = ibctesting.MockPort
			}, porttypes.ErrInvalidPort, "",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.SetupConnections()
			path.EndpointA.ChannelID = ibctesting.FirstChannelID

			counterparty = channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			channel = &channeltypes.Channel{
				State:          channeltypes.TRYOPEN,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{path.EndpointA.ConnectionID},
				Version:        types.V2,
			}
			counterpartyVersion = types.V2

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			chanCap, err = suite.chainA.App.GetScopedIBCKeeper().NewCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(ibctesting.TransferPort, path.EndpointA.ChannelID))
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(module)
			suite.Require().True(ok)

			tc.malleate() // explicitly change fields in channel and testChannel

			version, err := cbs.OnChanOpenTry(suite.chainA.GetContext(), channel.Ordering, channel.ConnectionHops,
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, chanCap, channel.Counterparty, counterpartyVersion,
			)
			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expVersion, version)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnChanOpenAck() {
	var counterpartyVersion string

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success", func() {}, nil,
		},
		{
			"invalid counterparty version",
			func() {
				counterpartyVersion = "version"
			},
			types.ErrInvalidVersion,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.SetupConnections()
			path.EndpointA.ChannelID = ibctesting.FirstChannelID
			counterpartyVersion = types.V2

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(module)
			suite.Require().True(ok)

			tc.malleate() // explicitly change fields in channel and testChannel

			err = cbs.OnChanOpenAck(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointA.Counterparty.ChannelID, counterpartyVersion)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnRecvPacket() {
	// This test suite mostly covers the top-level logic of the ibc module OnRecvPacket function
	// The core logic is covered in keeper OnRecvPacket
	var (
		packet             channeltypes.Packet
		expectedAttributes []sdk.Attribute
		path               *ibctesting.Path
	)
	testCases := []struct {
		name             string
		malleate         func()
		expAck           exported.Acknowledgement
		expEventErrorMsg string
	}{
		{
			"success", func() {}, channeltypes.NewResultAcknowledgement([]byte{byte(1)}), "",
		},
		{
			"success: async aknowledgment with forwarding path",
			func() {
				packetData := types.NewFungibleTokenPacketDataV2(
					[]types.Token{
						{
							Denom:  types.NewDenom(sdk.DefaultBondDenom),
							Amount: sdkmath.NewInt(100).String(),
						},
					},
					suite.chainA.SenderAccount.GetAddress().String(),
					suite.chainB.SenderAccount.GetAddress().String(),
					"", types.NewForwardingPacketData("", types.NewHop(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)),
				)
				packet.Data = packetData.GetBytes()

				forwardingHopsBz, err := json.Marshal(packetData.Forwarding.Hops)
				suite.Require().NoError(err)
				for i, attr := range expectedAttributes {
					if attr.Key == types.AttributeKeyForwardingHops {
						expectedAttributes[i].Value = string(forwardingHopsBz)
						break
					}
				}
			},
			nil,
			"",
		},
		{
			"failure: invalid packet data bytes",
			func() {
				packet.Data = []byte("invalid data")

				// Override expected attributes because this fails on unmarshaling packet data (so can't get the attributes)
				expectedAttributes = []sdk.Attribute{
					sdk.NewAttribute(types.AttributeKeySender, ""),
					sdk.NewAttribute(types.AttributeKeyReceiver, ""),
					sdk.NewAttribute(types.AttributeKeyTokens, "null"),
					sdk.NewAttribute(types.AttributeKeyMemo, ""),
					sdk.NewAttribute(types.AttributeKeyForwardingHops, "null"),
					sdk.NewAttribute(types.AttributeKeyAckSuccess, "false"),
					sdk.NewAttribute(types.AttributeKeyAckError, "cannot unmarshal ICS20-V2 transfer packet data: errUnknownField \"*types.FungibleTokenPacketDataV2\": {TagNum: 13, WireType:\"fixed64\"}: invalid type: invalid type"),
				}
			},
			channeltypes.NewErrorAcknowledgement(ibcerrors.ErrInvalidType),
			"cannot unmarshal ICS20-V2 transfer packet data: unexpected EOF: invalid type: invalid type",
		},
		{
			"failure: receive disabled",
			func() {
				suite.chainB.GetSimApp().TransferKeeper.SetParams(suite.chainB.GetContext(), types.Params{ReceiveEnabled: false})
			},
			channeltypes.NewErrorAcknowledgement(types.ErrReceiveDisabled),
			"fungible token transfers to this chain are disabled",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			packetData := types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(sdk.DefaultBondDenom),
						Amount: sdkmath.NewInt(100).String(),
					},
				},
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				"",
				ibctesting.EmptyForwardingPacketData,
			)

			tokensBz, err := json.Marshal(packetData.Tokens)
			suite.Require().NoError(err)
			forwardingHopsBz, err := json.Marshal(packetData.Forwarding.Hops)
			suite.Require().NoError(err)

			expectedAttributes = []sdk.Attribute{
				sdk.NewAttribute(types.AttributeKeySender, packetData.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, packetData.Receiver),
				sdk.NewAttribute(types.AttributeKeyTokens, string(tokensBz)),
				sdk.NewAttribute(types.AttributeKeyMemo, packetData.Memo),
				sdk.NewAttribute(types.AttributeKeyForwardingHops, string(forwardingHopsBz)),
			}
			if tc.expAck == nil || tc.expAck.Success() {
				expectedAttributes = append(expectedAttributes, sdk.NewAttribute(types.AttributeKeyAckSuccess, "true"))
			} else {
				expectedAttributes = append(expectedAttributes,
					sdk.NewAttribute(types.AttributeKeyAckSuccess, "false"),
					sdk.NewAttribute(types.AttributeKeyAckError, tc.expEventErrorMsg),
				)
			}

			seq := uint64(1)
			packet = channeltypes.NewPacket(packetData.GetBytes(), seq, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.ZeroHeight(), suite.chainA.GetTimeoutTimestamp())

			ctx := suite.chainB.GetContext()
			module, _, err := suite.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(ctx, ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainB.App.GetIBCKeeper().PortKeeper.Route(module)
			suite.Require().True(ok)

			tc.malleate() // change fields in packet

			ack := cbs.OnRecvPacket(ctx, packet, suite.chainB.SenderAccount.GetAddress())

			suite.Require().Equal(tc.expAck, ack)

			expectedEvents := sdk.Events{
				sdk.NewEvent(
					types.EventTypePacket,
					expectedAttributes...,
				),
			}.ToABCIEvents()

			expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
			ibctesting.AssertEvents(&suite.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())
		})
	}
}

func (suite *TransferTestSuite) TestOnTimeoutPacket() {
	var path *ibctesting.Path
	var packet channeltypes.Packet

	testCases := []struct {
		name           string
		coinsToSendToB sdk.Coins
		malleate       func()
		expError       error
	}{
		{
			"success",
			sdk.NewCoins(ibctesting.TestCoin),
			func() {},
			nil,
		},
		{
			"success with multiple coins",
			sdk.NewCoins(ibctesting.TestCoin, ibctesting.SecondaryTestCoin),
			func() {},
			nil,
		},
		{
			"non-existent channel",
			sdk.NewCoins(ibctesting.TestCoin),
			func() {
				packet.SourceChannel = "channel-100"
			},
			ibcerrors.ErrNotFound,
		},
		{
			"invalid packet data",
			sdk.NewCoins(ibctesting.TestCoin),
			func() {
				packet.Data = []byte("invalid data")
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"already timed-out packet",
			sdk.NewCoins(ibctesting.TestCoin),
			func() {
				module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
				suite.Require().NoError(err)

				cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(module)
				suite.Require().True(ok)

				suite.Require().NoError(cbs.OnTimeoutPacket(suite.chainA.GetContext(), packet, suite.chainA.SenderAccount.GetAddress()))
			},
			errors.New("unable to unescrow tokens"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			timeoutHeight := suite.chainA.GetTimeoutHeight()
			msg := types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				tc.coinsToSendToB,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				timeoutHeight,
				0,
				"",
				nil,
			)
			res, err := suite.chainA.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			packet, err = ibctesting.ParsePacketFromEvents(res.Events)
			suite.Require().NoError(err)

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(module)
			suite.Require().True(ok)

			tc.malleate() // change fields in packet

			err = cbs.OnTimeoutPacket(suite.chainA.GetContext(), packet, suite.chainA.SenderAccount.GetAddress())

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
				escrowBalanceAfter := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom)
				suite.Require().Equal(sdkmath.NewInt(0), escrowBalanceAfter.Amount)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnChanUpgradeInit() {
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {}, // successful happy path for a standalone transfer app is swapping out the underlying connection
			nil,
		},
		{
			"invalid upgrade connection",
			func() {
				path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.ConnectionHops = []string{"connection-100"}
				path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.ConnectionHops = []string{"connection-100"}
			},
			connectiontypes.ErrConnectionNotFound,
		},
		{
			"invalid upgrade ordering",
			func() {
				path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Ordering = channeltypes.ORDERED
				path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Ordering = channeltypes.ORDERED
			},
			channeltypes.ErrInvalidChannelOrdering,
		},
		{
			"invalid upgrade version",
			func() {
				path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = ibctesting.InvalidID
				path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = ibctesting.InvalidID
			},
			types.ErrInvalidVersion,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			// configure the channel upgrade to modify the underlying connection
			upgradePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			upgradePath.SetupConnections()

			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.ConnectionHops = []string{upgradePath.EndpointA.ConnectionID}
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.ConnectionHops = []string{upgradePath.EndpointB.ConnectionID}

			tc.malleate()

			err := path.EndpointA.ChanUpgradeInit()

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				upgrade := path.EndpointA.GetChannelUpgrade()
				suite.Require().Equal(upgradePath.EndpointA.ConnectionID, upgrade.Fields.ConnectionHops[0])
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnChanUpgradeTry() {
	var (
		counterpartyUpgrade channeltypes.Upgrade
		path                *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {}, // successful happy path for a standalone transfer app is swapping out the underlying connection
			nil,
		},
		{
			"success: invalid upgrade version from counterparty, we use our proposed version",
			func() {
				counterpartyUpgrade.Fields.Version = ibctesting.InvalidID
			},
			nil,
		},
		{
			"invalid upgrade ordering",
			func() {
				counterpartyUpgrade.Fields.Ordering = channeltypes.ORDERED
			},
			channeltypes.ErrInvalidChannelOrdering,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			// configure the channel upgrade to modify the underlying connection
			upgradePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			upgradePath.SetupConnections()

			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.ConnectionHops = []string{upgradePath.EndpointA.ConnectionID}
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.ConnectionHops = []string{upgradePath.EndpointB.ConnectionID}

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			counterpartyUpgrade = path.EndpointA.GetChannelUpgrade()

			tc.malleate()

			module, _, err := suite.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainB.GetContext(), types.PortID)
			suite.Require().NoError(err)

			app, ok := suite.chainB.App.GetIBCKeeper().PortKeeper.Route(module)
			suite.Require().True(ok)

			cbs, ok := app.(porttypes.UpgradableModule)
			suite.Require().True(ok)

			version, err := cbs.OnChanUpgradeTry(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
				counterpartyUpgrade.Fields.Ordering, counterpartyUpgrade.Fields.ConnectionHops, counterpartyUpgrade.Fields.Version,
			)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(types.V2, version)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnChanUpgradeAck() {
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {}, // successful happy path for a standalone transfer app is swapping out the underlying connection
			nil,
		},
		{
			"invalid upgrade version",
			func() {
				path.EndpointB.ChannelConfig.Version = ibctesting.InvalidID
			},
			types.ErrInvalidVersion,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			// configure the channel upgrade to modify the underlying connection
			upgradePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			upgradePath.SetupConnections()

			path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.ConnectionHops = []string{upgradePath.EndpointA.ConnectionID}
			path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.ConnectionHops = []string{upgradePath.EndpointB.ConnectionID}

			err := path.EndpointA.ChanUpgradeInit()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanUpgradeTry()
			suite.Require().NoError(err)

			tc.malleate()

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), types.PortID)
			suite.Require().NoError(err)

			app, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(module)
			suite.Require().True(ok)

			cbs, ok := app.(porttypes.UpgradableModule)
			suite.Require().True(ok)

			err = cbs.OnChanUpgradeAck(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.Version)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *TransferTestSuite) TestPacketDataUnmarshalerInterface() {
	var (
		sender   = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
		receiver = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

		path              *ibctesting.Path
		data              []byte
		initialPacketData interface{}
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: valid packet data single denom -> multidenom conversion with memo",
			func() {
				path.EndpointA.ChannelConfig.Version = types.V1
				initialPacketData = types.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     "some memo",
				}

				data = initialPacketData.(types.FungibleTokenPacketData).GetBytes()
			},
			nil,
		},
		{
			"success: valid packet data single denom -> multidenom conversion without memo",
			func() {
				path.EndpointA.ChannelConfig.Version = types.V1
				initialPacketData = types.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     "",
				}

				data = initialPacketData.(types.FungibleTokenPacketData).GetBytes()
			},
			nil,
		},
		{
			"success: valid packet data single denom with trace -> multidenom conversion with trace",
			func() {
				path.EndpointA.ChannelConfig.Version = types.V1
				initialPacketData = types.FungibleTokenPacketData{
					Denom:    "transfer/channel-0/atom",
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     "",
				}

				data = initialPacketData.(types.FungibleTokenPacketData).GetBytes()
			},
			nil,
		},
		{
			"success: valid packet data multidenom with memo",
			func() {
				initialPacketData = types.FungibleTokenPacketDataV2{
					Tokens: []types.Token{
						{
							Denom:  types.NewDenom("atom", types.NewHop("transfer", "channel-0")),
							Amount: ibctesting.TestCoin.Amount.String(),
						},
					},
					Sender:   sender,
					Receiver: receiver,
					Memo:     "some memo",
				}

				data = initialPacketData.(types.FungibleTokenPacketDataV2).GetBytes()
			},
			nil,
		},
		{
			"success: valid packet data multidenom nil trace",
			func() {
				path.EndpointA.ChannelConfig.Version = types.V2
				initialPacketData = types.FungibleTokenPacketDataV2{
					Tokens: []types.Token{
						{
							Denom:  types.NewDenom(ibctesting.TestCoin.Denom),
							Amount: ibctesting.TestCoin.Amount.String(),
						},
					},
					Sender:   sender,
					Receiver: receiver,
					Memo:     "",
				}

				data = initialPacketData.(types.FungibleTokenPacketDataV2).GetBytes()
			},
			nil,
		},
		{
			"failure: invalid token trace",
			func() {
				path.EndpointA.ChannelConfig.Version = types.V2
				initialPacketData = types.FungibleTokenPacketDataV2{
					Tokens: []types.Token{
						{
							Denom:  types.NewDenom(ibctesting.TestCoin.Denom, []types.Hop{{}}...),
							Amount: ibctesting.TestCoin.Amount.String(),
						},
					},
					Sender:   sender,
					Receiver: receiver,
					Memo:     "",
				}

				data = initialPacketData.(types.FungibleTokenPacketDataV2).GetBytes()
			},
			errors.New("invalid token denom: invalid trace: invalid hop source port ID : identifier cannot be blank: invalid identifier"),
		},
		{
			"failure: invalid packet data",
			func() {
				data = []byte("invalid packet data")
			},
			errors.New("cannot unmarshal ICS20-V2 transfer packet data"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)

			tc.malleate()

			path.Setup()

			transferStack, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(types.ModuleName)
			suite.Require().True(ok)

			unmarshalerStack, ok := transferStack.(porttypes.PacketDataUnmarshaler)
			suite.Require().True(ok)

			packetData, err := unmarshalerStack.UnmarshalPacketData(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, data)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				v2PacketData, ok := packetData.(types.FungibleTokenPacketDataV2)
				suite.Require().True(ok)

				if v1PacketData, ok := initialPacketData.(types.FungibleTokenPacketData); ok {
					// Note: testing of the denom trace parsing/conversion should be done as part of testing internal conversion functions
					suite.Require().Equal(v1PacketData.Amount, v2PacketData.Tokens[0].Amount)
					suite.Require().Equal(v1PacketData.Sender, v2PacketData.Sender)
					suite.Require().Equal(v1PacketData.Receiver, v2PacketData.Receiver)
					suite.Require().Equal(v1PacketData.Memo, v2PacketData.Memo)
				} else {
					suite.Require().Equal(initialPacketData.(types.FungibleTokenPacketDataV2), v2PacketData)
				}
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}
