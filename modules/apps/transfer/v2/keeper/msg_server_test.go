package keeper_test

import (
	"bytes"
	"time"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	mockv2 "github.com/cosmos/ibc-go/v9/testing/mock/v2"
)

// TestMsgSendPacketTransfer tests the MsgSendPacket rpc handler for the transfer v2 application.
func (suite *KeeperTestSuite) TestMsgSendPacketTransfer() {
	var payload channeltypesv2.Payload
	var path *ibctesting.Path

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
			"success: ABI encoded payload",
			func() {
				ftpd := transfertypes.NewFungibleTokenPacketData(sdk.DefaultBondDenom, ibctesting.DefaultCoinAmount.String(), suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "")
				bz, err := transfertypes.EncodeABIFungibleTokenPacketData(ftpd)
				suite.Require().NoError(err)
				payload = channeltypesv2.NewPayload(transfertypes.ModuleName, transfertypes.ModuleName, transfertypes.V1, transfertypes.EncodingABI, bz)
			},
			nil,
		},
		{
			"failure: send transfers disabled",
			func() {
				suite.chainA.GetSimApp().TransferKeeperV2.SetParams(suite.chainA.GetContext(),
					transfertypes.Params{
						SendEnabled: false,
					},
				)
			},
			transfertypes.ErrSendDisabled,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			tokens := []transfertypes.Token{
				{
					Denom: transfertypes.Denom{
						Base:  sdk.DefaultBondDenom,
						Trace: nil,
					},
					Amount: ibctesting.DefaultCoinAmount.String(),
				},
			}

			ftpd := transfertypes.NewFungibleTokenPacketDataV2(tokens, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "", transfertypes.ForwardingPacketData{})
			bz := suite.chainA.Codec.MustMarshal(&ftpd)

			timestamp := suite.chainA.GetTimeoutTimestampSecs()
			payload = channeltypesv2.NewPayload(transfertypes.ModuleName, transfertypes.ModuleName, transfertypes.V2, transfertypes.EncodingProtobuf, bz)

			tc.malleate()

			packet, err := path.EndpointA.MsgSendPacket(timestamp, payload)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(packet)

				// ensure every token sent is escrowed.
				for _, t := range tokens {
					escrowedAmount := suite.chainA.GetSimApp().TransferKeeperV2.GetTotalEscrowForDenom(suite.chainA.GetContext(), t.Denom.IBCDenom())
					expected, err := t.ToCoin()
					suite.Require().NoError(err)
					suite.Require().Equal(expected, escrowedAmount, "escrowed amount is not equal to expected amount")
				}
			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError, "expected error %q but got %q", tc.expError, err)
				suite.Require().Empty(packet)
			}
		})
	}
}

// TestMsgRecvPacketTransfer tests the MsgRecvPacket rpc handler for the transfer v2 application.
func (suite *KeeperTestSuite) TestMsgRecvPacketTransfer() {
	var (
		path        *ibctesting.Path
		packet      channeltypesv2.Packet
		expectedAck channeltypesv2.Acknowledgement
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
			"failure: invalid destination channel on received packet",
			func() {
				packet.DestinationChannel = ibctesting.InvalidID
			},
			channeltypesv2.ErrChannelNotFound,
		},
		{
			"failure: counter party channel does not match source channel",
			func() {
				packet.SourceChannel = ibctesting.InvalidID
			},
			channeltypes.ErrInvalidChannelIdentifier,
		},
		{
			"failure: receive is disabled",
			func() {
				expectedAck.AppAcknowledgements[0] = channeltypes.NewErrorAcknowledgement(transfertypes.ErrReceiveDisabled).Acknowledgement()
				suite.chainB.GetSimApp().TransferKeeperV2.SetParams(suite.chainB.GetContext(),
					transfertypes.Params{
						ReceiveEnabled: false,
					})
			},
			nil,
		},
		// TODO: async tests
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			tokens := []transfertypes.Token{
				{
					Denom: transfertypes.Denom{
						Base:  sdk.DefaultBondDenom,
						Trace: nil,
					},
					Amount: ibctesting.DefaultCoinAmount.String(),
				},
			}

			ftpd := transfertypes.NewFungibleTokenPacketDataV2(tokens, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "", transfertypes.ForwardingPacketData{})
			bz := suite.chainA.Codec.MustMarshal(&ftpd)

			timestamp := suite.chainA.GetTimeoutTimestampSecs()
			payload := channeltypesv2.NewPayload(transfertypes.ModuleName, transfertypes.ModuleName, transfertypes.V2, transfertypes.EncodingProtobuf, bz)
			var err error
			packet, err = path.EndpointA.MsgSendPacket(timestamp, payload)
			suite.Require().NoError(err)

			// by default, we assume a successful acknowledgement will be written.
			ackBytes := channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement()
			expectedAck = channeltypesv2.Acknowledgement{AppAcknowledgements: [][]byte{ackBytes}}
			tc.malleate()

			err = path.EndpointB.MsgRecvPacket(packet)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				actualAckHash := suite.chainB.GetSimApp().IBCKeeper.ChannelKeeperV2.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.DestinationChannel, packet.Sequence)
				expectedHash := channeltypesv2.CommitAcknowledgement(expectedAck)

				suite.Require().Equal(expectedHash, actualAckHash)

				denom := transfertypes.Denom{
					Base: sdk.DefaultBondDenom,
					Trace: []transfertypes.Hop{
						transfertypes.NewHop(payload.DestinationPort, packet.DestinationChannel),
					},
				}

				actualBalance := path.EndpointB.Chain.GetSimApp().TransferKeeperV2.BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), denom.IBCDenom())

				var expectedBalance sdk.Coin
				// on a successful ack we expect the full amount to be transferred
				if bytes.Equal(expectedAck.AppAcknowledgements[0], ackBytes) {
					expectedBalance = sdk.NewCoin(denom.IBCDenom(), ibctesting.DefaultCoinAmount)
				} else {
					// otherwise the tokens do not make it to the address.
					expectedBalance = sdk.NewCoin(denom.IBCDenom(), sdkmath.NewInt(0))
				}

				suite.Require().Equal(expectedBalance.Amount, actualBalance.Amount)

			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError, "expected error %q but got %q", tc.expError, err)
			}
		})
	}
}

// TestMsgAckPacketTransfer tests the MsgAcknowledgePacket rpc handler for the transfer v2 application.
func (suite *KeeperTestSuite) TestMsgAckPacketTransfer() {
	var (
		path        *ibctesting.Path
		packet      channeltypesv2.Packet
		expectedAck channeltypesv2.Acknowledgement
	)

	testCases := []struct {
		name               string
		malleate           func()
		expError           error
		causeFailureOnRecv bool
	}{
		{
			"success",
			func() {},
			nil,
			false,
		},
		{
			"failure: proof verification failure",
			func() {
				expectedAck.AppAcknowledgements[0] = mockv2.MockFailRecvPacketResult.Acknowledgement
			},
			commitmenttypes.ErrInvalidProof,
			false,
		},
		{
			"failure: escrowed tokens are refunded",
			func() {
				expectedAck.AppAcknowledgements[0] = channeltypes.NewErrorAcknowledgement(transfertypes.ErrReceiveDisabled).Acknowledgement()
			},
			nil,
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			tokens := []transfertypes.Token{
				{
					Denom: transfertypes.Denom{
						Base:  sdk.DefaultBondDenom,
						Trace: nil,
					},
					Amount: ibctesting.DefaultCoinAmount.String(),
				},
			}

			ftpd := transfertypes.NewFungibleTokenPacketDataV2(tokens, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "", transfertypes.ForwardingPacketData{})
			bz := suite.chainA.Codec.MustMarshal(&ftpd)

			timestamp := suite.chainA.GetTimeoutTimestampSecs()
			payload := channeltypesv2.NewPayload(transfertypes.ModuleName, transfertypes.ModuleName, transfertypes.V2, transfertypes.EncodingProtobuf, bz)

			var err error
			packet, err = path.EndpointA.MsgSendPacket(timestamp, payload)
			suite.Require().NoError(err)

			if tc.causeFailureOnRecv {
				// ensure that the recv packet fails at the application level, but succeeds at the IBC handler level
				// this will ensure that a failed ack will be written to state.
				suite.chainB.GetSimApp().TransferKeeperV2.SetParams(suite.chainB.GetContext(),
					transfertypes.Params{
						ReceiveEnabled: false,
					})
			}

			err = path.EndpointB.MsgRecvPacket(packet)
			suite.Require().NoError(err)

			ackBytes := channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement()
			expectedAck = channeltypesv2.Acknowledgement{AppAcknowledgements: [][]byte{ackBytes}}
			tc.malleate()

			err = path.EndpointA.MsgAcknowledgePacket(packet, expectedAck)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				if bytes.Equal(expectedAck.AppAcknowledgements[0], ackBytes) {
					// tokens remain escrowed
					for _, t := range tokens {
						escrowedAmount := suite.chainA.GetSimApp().TransferKeeperV2.GetTotalEscrowForDenom(suite.chainA.GetContext(), t.Denom.IBCDenom())
						expected, err := t.ToCoin()
						suite.Require().NoError(err)
						suite.Require().Equal(expected, escrowedAmount, "escrowed amount is not equal to expected amount")
					}
				} else {
					// tokens have been unescrowed
					for _, t := range tokens {
						escrowedAmount := suite.chainA.GetSimApp().TransferKeeperV2.GetTotalEscrowForDenom(suite.chainA.GetContext(), t.Denom.IBCDenom())
						suite.Require().Equal(sdk.NewCoin(t.Denom.IBCDenom(), sdkmath.NewInt(0)), escrowedAmount, "escrowed amount is not equal to expected amount")
					}
				}
			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError, "expected error %q but got %q", tc.expError, err)
			}
		})
	}
}

// TestMsgTimeoutPacketTransfer tests the MsgTimeoutPacket rpc handler for the transfer v2 application.
func (suite *KeeperTestSuite) TestMsgTimeoutPacketTransfer() {
	var (
		path             *ibctesting.Path
		packet           channeltypesv2.Packet
		timeoutTimestamp uint64
	)

	testCases := []struct {
		name          string
		malleate      func()
		timeoutPacket bool
		expError      error
	}{
		{
			"success",
			func() {},
			true,
			nil,
		},
		{
			"failure: packet has not timed out",
			func() {},
			false,
			channeltypes.ErrTimeoutNotReached,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupV2()

			tokens := []transfertypes.Token{
				{
					Denom: transfertypes.Denom{
						Base:  sdk.DefaultBondDenom,
						Trace: nil,
					},
					Amount: ibctesting.DefaultCoinAmount.String(),
				},
			}

			ftpd := transfertypes.NewFungibleTokenPacketDataV2(tokens, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "", transfertypes.ForwardingPacketData{})
			bz := suite.chainA.Codec.MustMarshal(&ftpd)

			timeoutTimestamp = uint64(suite.chainA.GetContext().BlockTime().Unix()) + uint64(time.Hour.Seconds())
			payload := channeltypesv2.NewPayload(transfertypes.ModuleName, transfertypes.ModuleName, transfertypes.V2, transfertypes.EncodingProtobuf, bz)

			var err error
			packet, err = path.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
			suite.Require().NoError(err)

			if tc.timeoutPacket {
				suite.coordinator.IncrementTimeBy(time.Hour * 2)
			}

			// ensure that chainA has an update to date client of chain B.
			suite.Require().NoError(path.EndpointA.UpdateClient())

			tc.malleate()

			err = path.EndpointA.MsgTimeoutPacket(packet)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				// ensure funds are un-escrowed
				for _, t := range tokens {
					escrowedAmount := suite.chainA.GetSimApp().TransferKeeperV2.GetTotalEscrowForDenom(suite.chainA.GetContext(), t.Denom.IBCDenom())
					suite.Require().Equal(sdk.NewCoin(t.Denom.IBCDenom(), sdkmath.NewInt(0)), escrowedAmount, "escrowed amount is not equal to expected amount")
				}

			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expError, "expected error %q but got %q", tc.expError, err)
				// tokens remain escrowed if there is a timeout failure
				for _, t := range tokens {
					escrowedAmount := suite.chainA.GetSimApp().TransferKeeperV2.GetTotalEscrowForDenom(suite.chainA.GetContext(), t.Denom.IBCDenom())
					expected, err := t.ToCoin()
					suite.Require().NoError(err)
					suite.Require().Equal(expected, escrowedAmount, "escrowed amount is not equal to expected amount")
				}
			}
		})
	}
}

func (suite *KeeperTestSuite) TestV2RetainsFungibility() {
	suite.SetupTest()

	path := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path.Setup()

	pathv2 := ibctesting.NewPath(suite.chainB, suite.chainC)
	pathv2.SetupV2()

	denomA := transfertypes.Denom{
		Base: sdk.DefaultBondDenom,
	}

	denomAtoB := transfertypes.Denom{
		Base: sdk.DefaultBondDenom,
		Trace: []transfertypes.Hop{
			transfertypes.NewHop(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID),
		},
	}

	denomBtoC := transfertypes.Denom{
		Base: sdk.DefaultBondDenom,
		Trace: []transfertypes.Hop{
			transfertypes.NewHop(transfertypes.ModuleName, pathv2.EndpointB.ChannelID),
			transfertypes.NewHop(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID),
		},
	}

	ackBytes := channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement()
	successfulAck := channeltypesv2.Acknowledgement{AppAcknowledgements: [][]byte{ackBytes}}

	originalAmount, ok := sdkmath.NewIntFromString(ibctesting.DefaultGenesisAccBalance)
	suite.Require().True(ok)

	suite.Run("between A and B", func() {
		var packet channeltypes.Packet
		suite.Run("transfer packet", func() {
			transferMsg := transfertypes.NewMsgTransfer(
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				sdk.NewCoins(sdk.NewCoin(denomA.IBCDenom(), ibctesting.TestCoin.Amount)),
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
				clienttypes.ZeroHeight(),
				suite.chainA.GetTimeoutTimestamp(),
				"memo",
				nil,
			)

			result, err := suite.chainA.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			remainingAmount := originalAmount.Sub(ibctesting.DefaultCoinAmount)
			suite.assertAmountOnChain(suite.chainA, balance, remainingAmount, denomA.IBCDenom())

			packet, err = ibctesting.ParsePacketFromEvents(result.Events)
			suite.Require().NoError(err)
		})

		suite.Run("recv and ack packet", func() {
			err := path.RelayPacket(packet)
			suite.Require().NoError(err)
		})
	})

	suite.Run("between B and C", func() {
		var packetV2 channeltypesv2.Packet

		suite.Run("send packet", func() {
			tokens := []transfertypes.Token{
				{
					Denom:  denomAtoB,
					Amount: ibctesting.DefaultCoinAmount.String(),
				},
			}

			ftpd := transfertypes.NewFungibleTokenPacketDataV2(tokens, suite.chainB.SenderAccount.GetAddress().String(), suite.chainC.SenderAccount.GetAddress().String(), "", transfertypes.ForwardingPacketData{})
			bz := suite.chainB.Codec.MustMarshal(&ftpd)

			timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().Unix()) + uint64(time.Hour.Seconds())
			payload := channeltypesv2.NewPayload(transfertypes.ModuleName, transfertypes.ModuleName, transfertypes.V2, transfertypes.EncodingProtobuf, bz)

			var err error
			packetV2, err = pathv2.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
			suite.Require().NoError(err)
			// the escrow account on chain B should have escrowed the tokens after sending from B to C
			suite.assertAmountOnChain(suite.chainB, escrow, ibctesting.DefaultCoinAmount, denomAtoB.IBCDenom())
		})

		suite.Run("recv packet", func() {
			err := pathv2.EndpointB.MsgRecvPacket(packetV2)
			suite.Require().NoError(err)

			// the receiving chain should have received the tokens
			suite.assertAmountOnChain(suite.chainC, balance, ibctesting.DefaultCoinAmount, denomBtoC.IBCDenom())
		})

		suite.Run("ack packet", func() {
			err := pathv2.EndpointA.MsgAcknowledgePacket(packetV2, successfulAck)
			suite.Require().NoError(err)
		})
	})

	suite.Run("between C and B", func() {
		var packetV2 channeltypesv2.Packet

		suite.Run("send packet", func() {
			// send from C to B
			tokens := []transfertypes.Token{
				{
					Denom:  denomBtoC,
					Amount: ibctesting.DefaultCoinAmount.String(),
				},
			}

			ftpd := transfertypes.NewFungibleTokenPacketDataV2(tokens, suite.chainC.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), "", transfertypes.ForwardingPacketData{})
			bz := suite.chainC.Codec.MustMarshal(&ftpd)

			timeoutTimestamp := uint64(suite.chainC.GetContext().BlockTime().Unix()) + uint64(time.Hour.Seconds())
			payload := channeltypesv2.NewPayload(transfertypes.ModuleName, transfertypes.ModuleName, transfertypes.V2, transfertypes.EncodingProtobuf, bz)

			var err error
			packetV2, err = pathv2.EndpointB.MsgSendPacket(timeoutTimestamp, payload)
			suite.Require().NoError(err)

			// tokens have been sent from chain C, and the balance is now empty.
			suite.assertAmountOnChain(suite.chainC, balance, sdkmath.NewInt(0), denomBtoC.IBCDenom())
		})

		suite.Run("recv packet", func() {
			err := pathv2.EndpointA.MsgRecvPacket(packetV2)
			suite.Require().NoError(err)

			// chain B should have received the tokens from chain C.
			suite.assertAmountOnChain(suite.chainB, balance, ibctesting.DefaultCoinAmount, denomAtoB.IBCDenom())
		})

		suite.Run("ack packet", func() {
			err := pathv2.EndpointB.MsgAcknowledgePacket(packetV2, successfulAck)
			suite.Require().NoError(err)
		})
	})

	suite.Run("between B and A", func() {
		var packet channeltypes.Packet

		suite.Run("transfer packet", func() {
			// send from B to A using MsgTransfer
			transferMsg := transfertypes.NewMsgTransfer(
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				sdk.NewCoins(sdk.NewCoin(denomAtoB.IBCDenom(), ibctesting.TestCoin.Amount)),
				suite.chainB.SenderAccount.GetAddress().String(),
				suite.chainA.SenderAccount.GetAddress().String(),
				clienttypes.ZeroHeight(),
				suite.chainB.GetTimeoutTimestamp(),
				"memo",
				nil,
			)

			result, err := suite.chainB.SendMsgs(transferMsg)
			suite.Require().NoError(err) // message committed

			suite.assertAmountOnChain(suite.chainB, balance, sdkmath.NewInt(0), denomAtoB.IBCDenom())

			packet, err = ibctesting.ParsePacketFromEvents(result.Events)
			suite.Require().NoError(err)
		})
		suite.Run("recv and ack packet", func() {
			// in order to recv in the other direction, we create a new path and recv
			// on that with the endpoints reversed.
			err := path.Reversed().RelayPacket(packet)
			suite.Require().NoError(err)

			suite.assertAmountOnChain(suite.chainA, balance, originalAmount, denomA.IBCDenom())
		})
	})
}
