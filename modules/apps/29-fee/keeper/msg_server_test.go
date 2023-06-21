package keeper_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

func (s *KeeperTestSuite) TestRegisterPayee() {
	var msg *types.MsgRegisterPayee

	testCases := []struct {
		name     string
		expPass  bool
		malleate func()
	}{
		{
			"success",
			true,
			func() {},
		},
		{
			"channel does not exist",
			false,
			func() {
				msg.ChannelId = "channel-100" //nolint:goconst
			},
		},
		{
			"channel is not fee enabled",
			false,
			func() {
				s.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
			},
		},
		{
			"given payee is not an sdk address",
			false,
			func() {
				msg.Payee = "invalid-addr"
			},
		},
		{
			"payee is a blocked address",
			false,
			func() {
				msg.Payee = s.chainA.GetSimApp().AccountKeeper.GetModuleAddress(transfertypes.ModuleName).String()
			},
		},
	}

	for _, tc := range testCases {
		s.SetupTest()
		s.coordinator.Setup(s.path)

		msg = types.NewMsgRegisterPayee(
			s.path.EndpointA.ChannelConfig.PortID,
			s.path.EndpointA.ChannelID,
			s.chainA.SenderAccounts[0].SenderAccount.GetAddress().String(),
			s.chainA.SenderAccounts[1].SenderAccount.GetAddress().String(),
		)

		tc.malleate()

		res, err := s.chainA.GetSimApp().IBCFeeKeeper.RegisterPayee(sdk.WrapSDKContext(s.chainA.GetContext()), msg)

		if tc.expPass {
			s.Require().NoError(err)
			s.Require().NotNil(res)

			payeeAddr, found := s.chainA.GetSimApp().IBCFeeKeeper.GetPayeeAddress(
				s.chainA.GetContext(),
				s.chainA.SenderAccount.GetAddress().String(),
				s.path.EndpointA.ChannelID,
			)

			s.Require().True(found)
			s.Require().Equal(s.chainA.SenderAccounts[1].SenderAccount.GetAddress().String(), payeeAddr)
		} else {
			s.Require().Error(err)
		}
	}
}

func (s *KeeperTestSuite) TestRegisterCounterpartyPayee() {
	var (
		msg                  *types.MsgRegisterCounterpartyPayee
		expCounterpartyPayee string
	)

	testCases := []struct {
		name     string
		expPass  bool
		malleate func()
	}{
		{
			"success",
			true,
			func() {},
		},
		{
			"counterparty payee is an arbitrary string",
			true,
			func() {
				msg.CounterpartyPayee = "arbitrary-string"
				expCounterpartyPayee = "arbitrary-string"
			},
		},
		{
			"channel does not exist",
			false,
			func() {
				msg.ChannelId = "channel-100"
			},
		},
		{
			"channel is not fee enabled",
			false,
			func() {
				s.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
			},
		},
	}

	for _, tc := range testCases {
		s.SetupTest()
		s.coordinator.Setup(s.path) // setup channel

		expCounterpartyPayee = s.chainA.SenderAccounts[1].SenderAccount.GetAddress().String()
		msg = types.NewMsgRegisterCounterpartyPayee(
			s.path.EndpointA.ChannelConfig.PortID,
			s.path.EndpointA.ChannelID,
			s.chainA.SenderAccounts[0].SenderAccount.GetAddress().String(),
			expCounterpartyPayee,
		)

		tc.malleate()

		res, err := s.chainA.GetSimApp().IBCFeeKeeper.RegisterCounterpartyPayee(sdk.WrapSDKContext(s.chainA.GetContext()), msg)

		if tc.expPass {
			s.Require().NoError(err)
			s.Require().NotNil(res)

			counterpartyPayee, found := s.chainA.GetSimApp().IBCFeeKeeper.GetCounterpartyPayeeAddress(
				s.chainA.GetContext(),
				s.chainA.SenderAccount.GetAddress().String(),
				ibctesting.FirstChannelID,
			)

			s.Require().True(found)
			s.Require().Equal(expCounterpartyPayee, counterpartyPayee)
		} else {
			s.Require().Error(err)
		}
	}
}

func (s *KeeperTestSuite) TestPayPacketFee() {
	var (
		expEscrowBalance sdk.Coins
		expFeesInEscrow  []types.PacketFee
		msg              *types.MsgPayPacketFee
		fee              types.Fee
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
			"success with existing packet fees in escrow",
			func() {
				fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)
				packetFee := types.NewPacketFee(fee, s.chainA.SenderAccount.GetAddress().String(), nil)
				feesInEscrow := types.NewPacketFees([]types.PacketFee{packetFee})

				s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, feesInEscrow)
				err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), types.ModuleName, fee.Total())
				s.Require().NoError(err)

				expEscrowBalance = expEscrowBalance.Add(fee.Total()...)
				expFeesInEscrow = append(expFeesInEscrow, packetFee)
			},
			true,
		},
		{
			"bank send enabled for fee denom",
			func() {
				err := s.chainA.GetSimApp().BankKeeper.SetParams(s.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: true}},
					},
				)
				s.Require().NoError(err)
			},
			true,
		},
		{
			"refund account is module account",
			func() {
				s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), ibcmock.ModuleName, fee.Total()) //nolint:errcheck // ignore error for testing
				msg.Signer = s.chainA.GetSimApp().AccountKeeper.GetModuleAddress(ibcmock.ModuleName).String()
				expPacketFee := types.NewPacketFee(fee, msg.Signer, nil)
				expFeesInEscrow = []types.PacketFee{expPacketFee}
			},
			true,
		},
		{
			"fee module is locked",
			func() {
				lockFeeModule(s.chainA)
			},
			false,
		},
		{
			"fee module disabled on channel",
			func() {
				msg.SourcePortId = "invalid-port"
				msg.SourceChannelId = "invalid-channel"
			},
			false,
		},
		{
			"invalid refund address",
			func() {
				msg.Signer = "invalid-address"
			},
			false,
		},
		{
			"refund account does not exist",
			func() {
				msg.Signer = s.chainB.SenderAccount.GetAddress().String()
			},
			false,
		},
		{
			"refund account is a blocked address",
			func() {
				blockedAddr := s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress()
				msg.Signer = blockedAddr.String()
			},
			false,
		},
		{
			"bank send disabled for fee denom",
			func() {
				err := s.chainA.GetSimApp().BankKeeper.SetParams(s.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: false}},
					},
				)
				s.Require().NoError(err)
			},
			false,
		},
		{
			"acknowledgement fee balance not found",
			func() {
				msg.Fee.AckFee = invalidCoins
			},
			false,
		},
		{
			"receive fee balance not found",
			func() {
				msg.Fee.RecvFee = invalidCoins
			},
			false,
		},
		{
			"timeout fee balance not found",
			func() {
				msg.Fee.TimeoutFee = invalidCoins
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()
			s.coordinator.Setup(s.path) // setup channel

			fee = types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			msg = types.NewMsgPayPacketFee(
				fee,
				s.path.EndpointA.ChannelConfig.PortID,
				s.path.EndpointA.ChannelID,
				s.chainA.SenderAccount.GetAddress().String(),
				nil,
			)

			expEscrowBalance = fee.Total()
			expPacketFee := types.NewPacketFee(fee, s.chainA.SenderAccount.GetAddress().String(), nil)
			expFeesInEscrow = []types.PacketFee{expPacketFee}

			tc.malleate()

			_, err := s.chainA.GetSimApp().IBCFeeKeeper.PayPacketFee(sdk.WrapSDKContext(s.chainA.GetContext()), msg)

			if tc.expPass {
				s.Require().NoError(err) // message committed

				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)
				feesInEscrow, found := s.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(s.chainA.GetContext(), packetID)
				s.Require().True(found)
				s.Require().Equal(expFeesInEscrow, feesInEscrow.PacketFees)

				escrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				s.Require().Equal(expEscrowBalance.AmountOf(sdk.DefaultBondDenom), escrowBalance.Amount)
			} else {
				s.Require().Error(err)

				escrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				s.Require().Equal(sdkmath.NewInt(0), escrowBalance.Amount)
			}
		})
	}
}

func (s *KeeperTestSuite) TestPayPacketFeeAsync() {
	var (
		packet           channeltypes.Packet
		expEscrowBalance sdk.Coins
		expFeesInEscrow  []types.PacketFee
		msg              *types.MsgPayPacketFeeAsync
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
			"success with existing packet fees in escrow",
			func() {
				fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)
				packetFee := types.NewPacketFee(fee, s.chainA.SenderAccount.GetAddress().String(), nil)
				feesInEscrow := types.NewPacketFees([]types.PacketFee{packetFee})

				s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, feesInEscrow)
				err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), types.ModuleName, fee.Total())
				s.Require().NoError(err)

				expEscrowBalance = expEscrowBalance.Add(fee.Total()...)
				expFeesInEscrow = append(expFeesInEscrow, packetFee)
			},
			true,
		},
		{
			"bank send enabled for fee denom",
			func() {
				err := s.chainA.GetSimApp().BankKeeper.SetParams(s.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: true}},
					},
				)
				s.Require().NoError(err)
			},
			true,
		},
		{
			"fee module is locked",
			func() {
				lockFeeModule(s.chainA)
			},
			false,
		},
		{
			"fee module disabled on channel",
			func() {
				msg.PacketId.PortId = "invalid-port"
				msg.PacketId.ChannelId = "invalid-channel"
			},
			false,
		},
		{
			"channel does not exist",
			func() {
				msg.PacketId.ChannelId = "channel-100"

				// to test this functionality, we must set the fee to enabled for this non existent channel
				// NOTE: the channel doesn't exist in 04-channel keeper, but we will add a mapping within ics29 anyways
				s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), msg.PacketId.PortId, msg.PacketId.ChannelId)
			},
			false,
		},
		{
			"packet not sent",
			func() {
				msg.PacketId.Sequence++
			},
			false,
		},
		{
			"packet already acknowledged",
			func() {
				err := s.path.RelayPacket(packet)
				s.Require().NoError(err)
			},
			false,
		},
		{
			"packet already timed out",
			func() {
				timeoutHeight := clienttypes.GetSelfHeight(s.chainB.GetContext())

				// try to incentivize a packet which is timed out
				sequence, err := s.path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// need to update chainA's client representing chainB to prove missing ack
				err = s.path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				packet = channeltypes.NewPacket(ibctesting.MockPacketData, sequence, s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID, timeoutHeight, 0)
				err = s.path.EndpointA.TimeoutPacket(packet)
				s.Require().NoError(err)

				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, sequence)
				msg.PacketId = packetID
			},
			false,
		},
		{
			"invalid refund address",
			func() {
				msg.PacketFee.RefundAddress = "invalid-address"
			},
			false,
		},
		{
			"refund account does not exist",
			func() {
				msg.PacketFee.RefundAddress = s.chainB.SenderAccount.GetAddress().String()
			},
			false,
		},
		{
			"refund account is a blocked address",
			func() {
				blockedAddr := s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress()
				msg.PacketFee.RefundAddress = blockedAddr.String()
			},
			false,
		},
		{
			"bank send disabled for fee denom",
			func() {
				err := s.chainA.GetSimApp().BankKeeper.SetParams(s.chainA.GetContext(),
					banktypes.Params{
						SendEnabled: []*banktypes.SendEnabled{{Denom: sdk.DefaultBondDenom, Enabled: false}},
					},
				)
				s.Require().NoError(err)
			},
			false,
		},
		{
			"acknowledgement fee balance not found",
			func() {
				msg.PacketFee.Fee.AckFee = invalidCoins
			},
			false,
		},
		{
			"receive fee balance not found",
			func() {
				msg.PacketFee.Fee.RecvFee = invalidCoins
			},
			false,
		},
		{
			"timeout fee balance not found",
			func() {
				msg.PacketFee.Fee.TimeoutFee = invalidCoins
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()
			s.coordinator.Setup(s.path) // setup channel

			timeoutHeight := clienttypes.NewHeight(clienttypes.ParseChainID(s.chainB.ChainID), 100)

			// send a packet to incentivize
			sequence, err := s.path.EndpointA.SendPacket(timeoutHeight, 0, ibctesting.MockPacketData)
			s.Require().NoError(err)
			packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, sequence)
			packet = channeltypes.NewPacket(ibctesting.MockPacketData, packetID.Sequence, packetID.PortId, packetID.ChannelId, s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID, timeoutHeight, 0)

			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, s.chainA.SenderAccount.GetAddress().String(), nil)

			expEscrowBalance = fee.Total()
			expFeesInEscrow = []types.PacketFee{packetFee}
			msg = types.NewMsgPayPacketFeeAsync(packetID, packetFee)

			tc.malleate()

			_, err = s.chainA.GetSimApp().IBCFeeKeeper.PayPacketFeeAsync(sdk.WrapSDKContext(s.chainA.GetContext()), msg)

			if tc.expPass {
				s.Require().NoError(err) // message committed

				feesInEscrow, found := s.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(s.chainA.GetContext(), packetID)
				s.Require().True(found)
				s.Require().Equal(expFeesInEscrow, feesInEscrow.PacketFees)

				escrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				s.Require().Equal(expEscrowBalance.AmountOf(sdk.DefaultBondDenom), escrowBalance.Amount)
			} else {
				s.Require().Error(err)

				escrowBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				s.Require().Equal(sdkmath.NewInt(0), escrowBalance.Amount)
			}
		})
	}
}
