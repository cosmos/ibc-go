package transfer_test

import (
	"errors"
	"math"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (suite *TransferTestSuite) TestOnChanOpenInit() {
	var (
		channel      *channeltypes.Channel
		path         *ibctesting.Path
		counterparty channeltypes.Counterparty
	)

	testCases := []struct {
		name       string
		malleate   func()
		expError   error
		expVersion string
	}{
		{
			"success", func() {}, nil, types.V1,
		},
		{
			// connection hops is not used in the transfer application callback,
			"success: invalid connection hops", func() {
				path.EndpointA.ConnectionID = ibctesting.InvalidID
			}, nil, types.V1,
		},
		{
			"success: empty version string", func() {
				channel.Version = ""
			}, nil, types.V1,
		},
		{
			"success: ics20-1", func() {
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
	}

	for _, tc := range testCases {
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
				Version:        types.V1,
			}

			tc.malleate() // explicitly change fields in channel and testChannel

			transferModule := transfer.NewIBCModule(suite.chainA.GetSimApp().TransferKeeper)
			version, err := transferModule.OnChanOpenInit(suite.chainA.GetContext(), channel.Ordering, channel.ConnectionHops,
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, counterparty, channel.Version,
			)

			if tc.expError == nil {
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
			"success", func() {}, nil, types.V1,
		},
		{
			"success: counterparty version is ics20-1", func() {
				counterpartyVersion = types.V1
			}, nil, types.V1,
		},
		{
			"success: invalid counterparty version, we propose new version", func() {
				// transfer module will propose the default version
				counterpartyVersion = "version"
			}, nil, types.V1,
		},
		{
			"failure: max channels reached", func() {
				path.EndpointA.ChannelID = channeltypes.FormatChannelIdentifier(math.MaxUint32 + 1)
			}, types.ErrMaxTransferChannels, "",
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
				Version:        types.V1,
			}
			counterpartyVersion = types.V1

			cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(ibctesting.TransferPort)
			suite.Require().True(ok)

			tc.malleate() // explicitly change fields in channel and testChannel

			version, err := cbs.OnChanOpenTry(suite.chainA.GetContext(), channel.Ordering, channel.ConnectionHops,
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel.Counterparty, counterpartyVersion,
			)
			if tc.expError == nil {
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
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.SetupConnections()
			path.EndpointA.ChannelID = ibctesting.FirstChannelID
			counterpartyVersion = types.V1

			cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(ibctesting.TransferPort)
			suite.Require().True(ok)

			tc.malleate() // explicitly change fields in channel and testChannel

			err := cbs.OnChanOpenAck(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointA.Counterparty.ChannelID, counterpartyVersion)

			if tc.expError == nil {
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
			"failure: invalid packet data bytes",
			func() {
				packet.Data = []byte("invalid data")

				// Override expected attributes because this fails on unmarshaling packet data (so can't get the attributes)
				expectedAttributes = []sdk.Attribute{
					sdk.NewAttribute(types.AttributeKeySender, ""),
					sdk.NewAttribute(types.AttributeKeyReceiver, ""),
					sdk.NewAttribute(types.AttributeKeyDenom, ""),
					sdk.NewAttribute(types.AttributeKeyAmount, ""),
					sdk.NewAttribute(types.AttributeKeyMemo, ""),
					sdk.NewAttribute(types.AttributeKeyAckSuccess, "false"),
					sdk.NewAttribute(types.AttributeKeyAckError, "cannot unmarshal ICS20-V1 transfer packet data: invalid character 'i' looking for beginning of value: invalid type"),
				}
			},
			channeltypes.NewErrorAcknowledgement(ibcerrors.ErrInvalidType),
			"cannot unmarshal ICS20-V1 transfer packet data: invalid character 'i' looking for beginning of value: invalid type",
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

			token := types.Token{
				Denom:  types.NewDenom(sdk.DefaultBondDenom),
				Amount: sdkmath.NewInt(100).String(),
			}
			packetData := types.NewFungibleTokenPacketData(
				token.Denom.Path(),
				token.Amount,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				"",
			)

			expectedAttributes = []sdk.Attribute{
				sdk.NewAttribute(types.AttributeKeySender, packetData.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, packetData.Receiver),
				sdk.NewAttribute(types.AttributeKeyDenom, packetData.Denom),
				sdk.NewAttribute(types.AttributeKeyAmount, packetData.Amount),
				sdk.NewAttribute(types.AttributeKeyMemo, packetData.Memo),
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
			cbs, ok := suite.chainB.App.GetIBCKeeper().PortKeeper.Route(ibctesting.TransferPort)
			suite.Require().True(ok)

			tc.malleate() // change fields in packet

			ack := cbs.OnRecvPacket(ctx, path.EndpointB.GetChannel().Version, packet, suite.chainB.SenderAccount.GetAddress())

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

func (suite *TransferTestSuite) TestOnAcknowledgePacket() {
	var (
		path   *ibctesting.Path
		packet channeltypes.Packet
		ack    []byte
	)

	testCases := []struct {
		name      string
		malleate  func()
		expError  error
		expRefund bool
	}{
		{
			"success",
			func() {},
			nil,
			false,
		},
		{
			"success: refund coins",
			func() {
				ack = channeltypes.NewErrorAcknowledgement(ibcerrors.ErrInsufficientFunds).Acknowledgement()
			},
			nil,
			true,
		},
		{
			"cannot refund ack on non-existent channel",
			func() {
				ack = channeltypes.NewErrorAcknowledgement(ibcerrors.ErrInsufficientFunds).Acknowledgement()

				packet.SourceChannel = "channel-100"
			},
			errors.New("unable to unescrow tokens"),
			false,
		},
		{
			"invalid packet data",
			func() {
				packet.Data = []byte("invalid data")
			},
			ibcerrors.ErrInvalidType,
			false,
		},
		{
			"invalid acknowledgement",
			func() {
				ack = []byte("invalid ack")
			},
			ibcerrors.ErrUnknownRequest,
			false,
		},
		{
			"cannot refund already acknowledged packet",
			func() {
				ack = channeltypes.NewErrorAcknowledgement(ibcerrors.ErrInsufficientFunds).Acknowledgement()

				cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(ibctesting.TransferPort)
				suite.Require().True(ok)

				suite.Require().NoError(cbs.OnAcknowledgementPacket(suite.chainA.GetContext(), path.EndpointA.GetChannel().Version, packet, ack, suite.chainA.SenderAccount.GetAddress()))
			},
			errors.New("unable to unescrow tokens"),
			false,
		},
		{
			// See https://github.com/cosmos/ibc-go/security/advisories/GHSA-jg6f-48ff-5xrw
			"non-deterministic JSON ack serialization should return an error",
			func() {
				// Create a valid acknowledgement using deterministic serialization.
				ack = channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement()
				// Introduce non-determinism: insert an extra space after the first character '{'
				// This will deserialize correctly but fail to re-serialize to the expected bytes.
				if len(ack) > 0 && ack[0] == '{' {
					ack = []byte("{ " + string(ack[1:]))
				}
			},
			errors.New("acknowledgement did not marshal to expected bytes"),
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			timeoutHeight := suite.chainA.GetTimeoutHeight()
			msg := types.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				ibctesting.TestCoin,
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				timeoutHeight,
				0,
				"",
			)
			res, err := suite.chainA.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			packet, err = ibctesting.ParsePacketFromEvents(res.Events)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(ibctesting.TransferPort)
			suite.Require().True(ok)

			ack = channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement()

			tc.malleate() // change fields in packet

			err = cbs.OnAcknowledgementPacket(suite.chainA.GetContext(), path.EndpointA.GetChannel().Version, packet, ack, suite.chainA.SenderAccount.GetAddress())

			if tc.expError == nil {
				suite.Require().NoError(err)

				if tc.expRefund {
					escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
					escrowBalanceAfter := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, sdk.DefaultBondDenom)
					suite.Require().Equal(sdkmath.NewInt(0), escrowBalanceAfter.Amount)
				}
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}

func (suite *TransferTestSuite) TestOnTimeoutPacket() {
	var path *ibctesting.Path
	var packet channeltypes.Packet

	testCases := []struct {
		name           string
		coinsToSendToB sdk.Coin
		malleate       func()
		expError       error
	}{
		{
			"success",
			ibctesting.TestCoin,
			func() {},
			nil,
		},
		{
			"non-existent channel",
			ibctesting.TestCoin,
			func() {
				packet.SourceChannel = "channel-100"
			},
			errors.New("unable to unescrow tokens"),
		},
		{
			"invalid packet data",
			ibctesting.TestCoin,
			func() {
				packet.Data = []byte("invalid data")
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"already timed-out packet",
			ibctesting.TestCoin,
			func() {
				cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(ibctesting.TransferPort)
				suite.Require().True(ok)

				suite.Require().NoError(cbs.OnTimeoutPacket(suite.chainA.GetContext(), path.EndpointA.GetChannel().Version, packet, suite.chainA.SenderAccount.GetAddress()))
			},
			errors.New("unable to unescrow tokens"),
		},
	}

	for _, tc := range testCases {
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
			)
			res, err := suite.chainA.SendMsgs(msg)
			suite.Require().NoError(err) // message committed

			packet, err = ibctesting.ParsePacketFromEvents(res.Events)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(ibctesting.TransferPort)
			suite.Require().True(ok)

			tc.malleate() // change fields in packet

			err = cbs.OnTimeoutPacket(suite.chainA.GetContext(), path.EndpointA.GetChannel().Version, packet, suite.chainA.SenderAccount.GetAddress())

			if tc.expError == nil {
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

func (suite *TransferTestSuite) TestPacketDataUnmarshalerInterface() {
	var (
		sender   = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
		receiver = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

		data              []byte
		initialPacketData any
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: valid packet data with memo",
			func() {
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
			"success: valid packet data denom with trace",
			func() {
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
			"failure: invalid packet data",
			func() {
				data = []byte("invalid packet data")
			},
			errors.New("cannot unmarshal ICS20-V1 transfer packet data: invalid character 'i' looking for beginning of value: invalid type"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()

			path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			transferStack, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(types.ModuleName)
			suite.Require().True(ok)

			unmarshalerStack, ok := transferStack.(porttypes.PacketDataUnmarshaler)
			suite.Require().True(ok)

			packetData, version, err := unmarshalerStack.UnmarshalPacketData(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, data)

			if tc.expError == nil {
				suite.Require().NoError(err)

				v2PacketData, ok := packetData.(types.InternalTransferRepresentation)
				suite.Require().True(ok)
				suite.Require().Equal(path.EndpointA.ChannelConfig.Version, version)

				if v1PacketData, ok := initialPacketData.(types.FungibleTokenPacketData); ok {
					// Note: testing of the denom trace parsing/conversion should be done as part of testing internal conversion functions
					suite.Require().Equal(v1PacketData.Amount, v2PacketData.Token.Amount)
					suite.Require().Equal(v1PacketData.Sender, v2PacketData.Sender)
					suite.Require().Equal(v1PacketData.Receiver, v2PacketData.Receiver)
					suite.Require().Equal(v1PacketData.Memo, v2PacketData.Memo)
				} else {
					suite.Require().Equal(initialPacketData.(types.InternalTransferRepresentation), v2PacketData)
				}
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expError.Error())
			}
		})
	}
}
