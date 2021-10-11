package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

func (suite *KeeperTestSuite) TestEscrowPacketFee() {
	var (
		err        error
		refundAcc  sdk.AccAddress
		ackFee     *sdk.Coin
		receiveFee *sdk.Coin
		timeoutFee *sdk.Coin
	)

	// refundAcc does not have balance for the following Coin
	invalidCoin := &sdk.Coin{Denom: "cosmos", Amount: sdk.NewInt(8000000000)}
	validChannelId := "channel-0"

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"refundAcc does not exist", func() {
				// this acc does nto exist on chainA
				refundAcc = suite.chainB.SenderAccount.GetAddress()
			}, false,
		},
		{
			"ackFee balance not found", func() {
				ackFee = invalidCoin
			}, false,
		},
		{
			"receive balance not found", func() {
				receiveFee = invalidCoin
			}, false,
		},
		{
			"timeout balance not found", func() {
				timeoutFee = invalidCoin
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// setup
			refundAcc = suite.chainA.SenderAccount.GetAddress()
			validCoin := &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}
			ackFee = validCoin
			receiveFee = validCoin
			timeoutFee = validCoin
			packetId := &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(1)}

			tc.malleate()
			fee := &types.Fee{ackFee, receiveFee, timeoutFee}
			identifiedPacketFee := types.IdentifiedPacketFee{PacketId: packetId, Fee: fee, Relayers: []string{}}

			// escrow the packet fee
			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), refundAcc, identifiedPacketFee)

			if tc.expPass {
				feeInEscrow, _ := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeInEscrow(suite.chainA.GetContext(), packetId.ChannelId, packetId.Sequence)
				// check if the escrowed fee is set in state
				suite.Require().Equal(fee.AckFee, feeInEscrow.Fee.AckFee)
				suite.Require().Equal(fee.ReceiveFee, feeInEscrow.Fee.ReceiveFee)
				suite.Require().Equal(fee.TimeoutFee, feeInEscrow.Fee.TimeoutFee)
				// check if the fee is escrowed correctly
				hasBalance := suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(300)})
				suite.Require().True(hasBalance)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestPayFee() {
	var (
		err            error
		ackFee         *sdk.Coin
		receiveFee     *sdk.Coin
		timeoutFee     *sdk.Coin
		packetId       *channeltypes.PacketId
		reverseRelayer sdk.AccAddress
		forwardRelayer sdk.AccAddress
		refundAcc      sdk.AccAddress
	)

	// refundAcc does not have balance for the following Coin
	validChannelId := "channel-0"
	validSeq := uint64(1)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"fee not found for packet", func() {
				// setting packetId with an invalid sequence of 2
				packetId = &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(2)}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// setup
			refundAcc = suite.chainA.SenderAccount.GetAddress()
			reverseRelayer = suite.chainA.SenderAccount.GetAddress()
			forwardRelayer = suite.chainA.SenderAccount.GetAddress()

			validCoin := &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}
			ackFee = validCoin
			receiveFee = validCoin
			timeoutFee = validCoin
			packetId = &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: validSeq}
			fee := &types.Fee{ackFee, receiveFee, timeoutFee}

			// escrow the packet fee & store the fee in state
			identifiedPacketFee := types.IdentifiedPacketFee{PacketId: packetId, Fee: fee, Relayers: []string{}}

			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), refundAcc, identifiedPacketFee)
			suite.Require().NoError(err)

			tc.malleate()

			err = suite.chainA.GetSimApp().IBCFeeKeeper.PayFee(suite.chainA.GetContext(), refundAcc, forwardRelayer, reverseRelayer, packetId)

			if tc.expPass {
				suite.Require().NoError(err)
				hasFeeInEscrow := suite.chainA.GetSimApp().IBCFeeKeeper.HasFeeInEscrow(suite.chainA.GetContext(), packetId.ChannelId, packetId.Sequence)
				// there should no longer be a fee in escrow for this packet
				suite.Require().False(hasFeeInEscrow)
			} else {
				suite.Require().Error(err)
				hasFeeInEscrow := suite.chainA.GetSimApp().IBCFeeKeeper.HasFeeInEscrow(suite.chainA.GetContext(), packetId.ChannelId, 1)
				// there should still be a fee in escrow for this packet
				suite.Require().True(hasFeeInEscrow)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestPayTimeoutFee() {
	var (
		err            error
		ackFee         *sdk.Coin
		receiveFee     *sdk.Coin
		timeoutFee     *sdk.Coin
		packetId       *channeltypes.PacketId
		reverseRelayer sdk.AccAddress
		refundAcc      sdk.AccAddress
	)

	// refundAcc does not have balance for the following Coin
	validChannelId := "channel-0"
	validSeq := uint64(1)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"fee not found for packet", func() {
				// setting packetId with an invalid sequence of 2
				packetId = &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(2)}
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// setup
			refundAcc = suite.chainA.SenderAccount.GetAddress()
			reverseRelayer = suite.chainA.SenderAccount.GetAddress()

			validCoin := &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}
			ackFee = validCoin
			receiveFee = validCoin
			timeoutFee = validCoin
			packetId = &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: validSeq}
			fee := &types.Fee{ackFee, receiveFee, timeoutFee}

			// escrow the packet fee & store the fee in state
			identifiedPacketFee := types.IdentifiedPacketFee{PacketId: packetId, Fee: fee, Relayers: []string{}}
			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), refundAcc, identifiedPacketFee)
			suite.Require().NoError(err)

			tc.malleate()

			err = suite.chainA.GetSimApp().IBCFeeKeeper.PayFeeTimeout(suite.chainA.GetContext(), refundAcc, reverseRelayer, packetId)

			if tc.expPass {
				suite.Require().NoError(err)
				hasFeeInEscrow := suite.chainA.GetSimApp().IBCFeeKeeper.HasFeeInEscrow(suite.chainA.GetContext(), packetId.ChannelId, packetId.Sequence)
				// there should no longer be a fee in escrow for this packet
				suite.Require().False(hasFeeInEscrow)
			} else {
				suite.Require().Error(err)
				hasFeeInEscrow := suite.chainA.GetSimApp().IBCFeeKeeper.HasFeeInEscrow(suite.chainA.GetContext(), packetId.ChannelId, 1)
				// there should still be a fee in escrow for this packet
				suite.Require().True(hasFeeInEscrow)
			}
		})
	}
}
