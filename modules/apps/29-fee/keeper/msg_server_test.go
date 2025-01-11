package keeper_test

import (
	"errors"
	"fmt"

	sdkmath "cosmossdk.io/math"
	banktypes "cosmossdk.io/x/bank/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	ibcmock "github.com/cosmos/ibc-go/v9/testing/mock"
)

func (suite *KeeperTestSuite) TestRegisterPayee() {
	var msg *types.MsgRegisterPayee

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
			"channel does not exist",
			func() {
				msg.ChannelId = "channel-100" //nolint:goconst
			},
			channeltypes.ErrChannelNotFound,
		},
		{
			"channel is not fee enabled",
			func() {
				suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
			},
			types.ErrFeeNotEnabled,
		},
		{
			"given payee is not an sdk address",
			func() {
				msg.Payee = "invalid-addr"
			},
			errors.New("decoding bech32 failed: invalid separator index -1"),
		},
		{
			"payee is a blocked address",
			func() {
				msg.Payee = suite.chainA.GetSimApp().AuthKeeper.GetModuleAddress(transfertypes.ModuleName).String()
			},
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.path.Setup()

			msg = types.NewMsgRegisterPayee(
				suite.path.EndpointA.ChannelConfig.PortID,
				suite.path.EndpointA.ChannelID,
				suite.chainA.SenderAccounts[0].SenderAccount.GetAddress().String(),
				suite.chainA.SenderAccounts[1].SenderAccount.GetAddress().String(),
			)

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.RegisterPayee(ctx, msg)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				payeeAddr, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetPayeeAddress(
					suite.chainA.GetContext(),
					suite.chainA.SenderAccount.GetAddress().String(),
					suite.path.EndpointA.ChannelID,
				)

				suite.Require().True(found)
				suite.Require().Equal(suite.chainA.SenderAccounts[1].SenderAccount.GetAddress().String(), payeeAddr)

				expectedEvents := sdk.Events{
					sdk.NewEvent(
						types.EventTypeRegisterPayee,
						sdk.NewAttribute(types.AttributeKeyRelayer, suite.chainA.SenderAccount.GetAddress().String()),
						sdk.NewAttribute(types.AttributeKeyPayee, payeeAddr),
						sdk.NewAttribute(types.AttributeKeyChannelID, suite.path.EndpointA.ChannelID),
					),
				}.ToABCIEvents()

				expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
				ibctesting.AssertEvents(&suite.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())

			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expErr, err.Error())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestRegisterCounterpartyPayee() {
	var (
		msg                  *types.MsgRegisterCounterpartyPayee
		expCounterpartyPayee string
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
			"counterparty payee is an arbitrary string",
			func() {
				msg.CounterpartyPayee = "arbitrary-string"
				expCounterpartyPayee = "arbitrary-string"
			},
			nil,
		},
		{
			"channel does not exist",
			func() {
				msg.ChannelId = "channel-100"
			},
			channeltypes.ErrChannelNotFound,
		},
		{
			"channel is not fee enabled",
			func() {
				suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
			},
			types.ErrFeeNotEnabled,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.path.Setup() // setup channel

			expCounterpartyPayee = suite.chainA.SenderAccounts[1].SenderAccount.GetAddress().String()
			msg = types.NewMsgRegisterCounterpartyPayee(
				suite.path.EndpointA.ChannelConfig.PortID,
				suite.path.EndpointA.ChannelID,
				suite.chainA.SenderAccounts[0].SenderAccount.GetAddress().String(),
				expCounterpartyPayee,
			)

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.RegisterCounterpartyPayee(ctx, msg)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				counterpartyPayee, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetCounterpartyPayeeAddress(
					suite.chainA.GetContext(),
					suite.chainA.SenderAccount.GetAddress().String(),
					suite.path.EndpointA.ChannelID,
				)

				suite.Require().True(found)
				suite.Require().Equal(expCounterpartyPayee, counterpartyPayee)

				expectedEvents := sdk.Events{
					sdk.NewEvent(
						types.EventTypeRegisterCounterpartyPayee,
						sdk.NewAttribute(types.AttributeKeyRelayer, suite.chainA.SenderAccount.GetAddress().String()),
						sdk.NewAttribute(types.AttributeKeyCounterpartyPayee, counterpartyPayee),
						sdk.NewAttribute(types.AttributeKeyChannelID, suite.path.EndpointA.ChannelID),
					),
				}.ToABCIEvents()

				expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
				ibctesting.AssertEvents(&suite.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())

			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestPayPacketFee() {
	var (
		expEscrowBalance sdk.Coins
		expFeesInEscrow  []types.PacketFee
		msg              *types.MsgPayPacketFee
		fee              types.Fee
		eventFee         types.Fee
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
			"success with existing packet fees in escrow",
			func() {
				escrowFee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
				packetFee := types.NewPacketFee(escrowFee, suite.chainA.SenderAccount.GetAddress().String(), nil)
				feesInEscrow := types.NewPacketFees([]types.PacketFee{packetFee})

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, feesInEscrow)
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), types.ModuleName, escrowFee.Total())
				suite.Require().NoError(err)

				expEscrowBalance = expEscrowBalance.Add(escrowFee.Total()...)
				expFeesInEscrow = append(expFeesInEscrow, packetFee)

				eventFee = types.NewFee(defaultRecvFee.Add(escrowFee.RecvFee...), defaultAckFee.Add(escrowFee.AckFee...), defaultTimeoutFee.Add(escrowFee.TimeoutFee...))
			},
			nil,
		},
		{
			"refund account is module account",
			func() {
				suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibcmock.ModuleName, fee.Total()) //nolint:errcheck // ignore error for testing
				msg.Signer = suite.chainA.GetSimApp().AuthKeeper.GetModuleAddress(ibcmock.ModuleName).String()
				expPacketFee := types.NewPacketFee(fee, msg.Signer, nil)
				expFeesInEscrow = []types.PacketFee{expPacketFee}
			},
			nil,
		},
		{
			"bank send enabled for fee denom",
			func() {
				err := suite.chainA.GetSimApp().BankKeeper.SetParams(suite.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: true}},
					},
				)
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"fee module is locked",
			func() {
				lockFeeModule(suite.chainA)
			},
			types.ErrFeeModuleLocked,
		},
		{
			"fee module disabled on channel",
			func() {
				msg.SourcePortId = "invalid-port"
				msg.SourceChannelId = "invalid-channel"
			},
			types.ErrFeeNotEnabled,
		},
		{
			"invalid refund address",
			func() {
				msg.Signer = "invalid-address"
			},
			errors.New("decoding bech32 failed"),
		},
		{
			"refund account does not exist",
			func() {
				msg.Signer = suite.chainB.SenderAccount.GetAddress().String()
			},
			types.ErrRefundAccNotFound,
		},
		{
			"refund account is a blocked address",
			func() {
				blockedAddr := suite.chainA.GetSimApp().AuthKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress()
				msg.Signer = blockedAddr.String()
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"bank send disabled for fee denom",
			func() {
				err := suite.chainA.GetSimApp().BankKeeper.SetParams(suite.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: false}},
					},
				)
				suite.Require().NoError(err)
			},
			banktypes.ErrSendDisabled,
		},
		{
			"acknowledgement fee balance not found",
			func() {
				msg.Fee.AckFee = invalidCoins
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"receive fee balance not found",
			func() {
				msg.Fee.RecvFee = invalidCoins
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"timeout fee balance not found",
			func() {
				msg.Fee.TimeoutFee = invalidCoins
			},
			sdkerrors.ErrInsufficientFunds,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.path.Setup() // setup channel

			fee = types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			msg = types.NewMsgPayPacketFee(
				fee,
				suite.path.EndpointA.ChannelConfig.PortID,
				suite.path.EndpointA.ChannelID,
				suite.chainA.SenderAccount.GetAddress().String(),
				nil,
			)

			expEscrowBalance = fee.Total()
			expPacketFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), nil)
			expFeesInEscrow = []types.PacketFee{expPacketFee}
			eventFee = fee

			tc.malleate()

			ctx := suite.chainA.GetContext()
			_, err := suite.chainA.GetSimApp().IBCFeeKeeper.PayPacketFee(ctx, msg)

			if tc.expErr == nil {
				suite.Require().NoError(err) // message committed

				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
				feesInEscrow, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(suite.chainA.GetContext(), packetID)
				suite.Require().True(found)
				suite.Require().Equal(expFeesInEscrow, feesInEscrow.PacketFees)

				escrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(expEscrowBalance.AmountOf(sdk.DefaultBondDenom), escrowBalance.Amount)

				expectedEvents := sdk.Events{
					sdk.NewEvent(
						types.EventTypeIncentivizedPacket,
						sdk.NewAttribute(channeltypes.AttributeKeyPortID, packetID.PortId),
						sdk.NewAttribute(channeltypes.AttributeKeyChannelID, packetID.ChannelId),
						sdk.NewAttribute(channeltypes.AttributeKeySequence, fmt.Sprint(packetID.Sequence)),
						sdk.NewAttribute(types.AttributeKeyRecvFee, eventFee.RecvFee.String()),
						sdk.NewAttribute(types.AttributeKeyAckFee, eventFee.AckFee.String()),
						sdk.NewAttribute(types.AttributeKeyTimeoutFee, eventFee.TimeoutFee.String()),
					),
				}.ToABCIEvents()

				expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
				ibctesting.AssertEvents(&suite.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())

			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expErr, err.Error())

				escrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(sdkmath.NewInt(0), escrowBalance.Amount)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestPayPacketFeeAsync() {
	var (
		packet           channeltypes.Packet
		expEscrowBalance sdk.Coins
		expFeesInEscrow  []types.PacketFee
		msg              *types.MsgPayPacketFeeAsync
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
			"success with existing packet fees in escrow",
			func() {
				fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
				packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), nil)
				feesInEscrow := types.NewPacketFees([]types.PacketFee{packetFee})

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, feesInEscrow)
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), types.ModuleName, fee.Total())
				suite.Require().NoError(err)

				expEscrowBalance = expEscrowBalance.Add(fee.Total()...)
				expFeesInEscrow = append(expFeesInEscrow, packetFee)
			},
			nil,
		},
		{
			"bank send enabled for fee denom",
			func() {
				err := suite.chainA.GetSimApp().BankKeeper.SetParams(suite.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: true}},
					},
				)
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"fee module is locked",
			func() {
				lockFeeModule(suite.chainA)
			},
			types.ErrFeeModuleLocked,
		},
		{
			"fee module disabled on channel",
			func() {
				msg.PacketId.PortId = "invalid-port"
				msg.PacketId.ChannelId = "invalid-channel"
			},
			types.ErrFeeNotEnabled,
		},
		{
			"channel does not exist",
			func() {
				msg.PacketId.ChannelId = "channel-100"

				// to test this functionality, we must set the fee to enabled for this non existent channel
				// NOTE: the channel doesn't exist in 04-channel keeper, but we will add a mapping within ics29 anyways
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), msg.PacketId.PortId, msg.PacketId.ChannelId)
			},
			channeltypes.ErrSequenceSendNotFound,
		},
		{
			"packet not sent",
			func() {
				msg.PacketId.Sequence++
			},
			channeltypes.ErrPacketNotSent,
		},
		{
			"packet already acknowledged",
			func() {
				err := suite.path.RelayPacket(packet)
				suite.Require().NoError(err)
			},
			channeltypes.ErrPacketCommitmentNotFound,
		},
		{
			"packet already timed out",
			func() {
				timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())

				// try to incentivize a packet which is timed out
				sequence, err := suite.path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// need to update chainA's client representing chainB to prove missing ack
				err = suite.path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID, timeoutHeight, 0)
				err = suite.path.EndpointA.TimeoutPacket(packet)
				suite.Require().NoError(err)

				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, sequence)
				msg.PacketId = packetID
			},
			channeltypes.ErrPacketCommitmentNotFound,
		},
		{
			"invalid refund address",
			func() {
				msg.PacketFee.RefundAddress = "invalid-address"
			},
			errors.New("decoding bech32 failed"),
		},
		{
			"refund account does not exist",
			func() {
				msg.PacketFee.RefundAddress = suite.chainB.SenderAccount.GetAddress().String()
			},
			types.ErrRefundAccNotFound,
		},
		{
			"refund account is a blocked address",
			func() {
				blockedAddr := suite.chainA.GetSimApp().AuthKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress()
				msg.PacketFee.RefundAddress = blockedAddr.String()
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"bank send disabled for fee denom",
			func() {
				err := suite.chainA.GetSimApp().BankKeeper.SetParams(suite.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: false}},
					},
				)
				suite.Require().NoError(err)
			},
			banktypes.ErrSendDisabled,
		},
		{
			"acknowledgement fee balance not found",
			func() {
				msg.PacketFee.Fee.AckFee = invalidCoins
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"receive fee balance not found",
			func() {
				msg.PacketFee.Fee.RecvFee = invalidCoins
			},
			sdkerrors.ErrInsufficientFunds,
		},
		{
			"timeout fee balance not found",
			func() {
				msg.PacketFee.Fee.TimeoutFee = invalidCoins
			},
			sdkerrors.ErrInsufficientFunds,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.path.Setup() // setup channel

			timeoutHeight := clienttypes.NewHeight(clienttypes.ParseChainID(suite.chainB.ChainID), 100)

			// send a packet to incentivize
			sequence, err := suite.path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			suite.Require().NoError(err)
			packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, sequence)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, packetID.Sequence, packetID.PortId, packetID.ChannelId, suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID, timeoutHeight, 0)

			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), nil)

			expEscrowBalance = fee.Total()
			expFeesInEscrow = []types.PacketFee{packetFee}
			msg = types.NewMsgPayPacketFeeAsync(packetID, packetFee)

			tc.malleate()

			_, err = suite.chainA.GetSimApp().IBCFeeKeeper.PayPacketFeeAsync(suite.chainA.GetContext(), msg)

			if tc.expErr == nil {
				suite.Require().NoError(err) // message committed

				feesInEscrow, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(suite.chainA.GetContext(), packetID)
				suite.Require().True(found)
				suite.Require().Equal(expFeesInEscrow, feesInEscrow.PacketFees)

				escrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(expEscrowBalance.AmountOf(sdk.DefaultBondDenom), escrowBalance.Amount)
			} else {
				ibctesting.RequireErrorIsOrContains(suite.T(), err, tc.expErr, err.Error())

				escrowBalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(sdkmath.NewInt(0), escrowBalance.Amount)
			}
		})
	}
}
