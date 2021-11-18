package keeper_test

import (
	"github.com/tendermint/tendermint/crypto/secp256k1"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

var (
	validCoins   = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	validCoins2  = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(200)}}
	validCoins3  = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(300)}}
	invalidCoins = sdk.Coins{sdk.Coin{Denom: "invalidDenom", Amount: sdk.NewInt(100)}}
)

func (suite *KeeperTestSuite) TestEscrowPacketFee() {
	var (
		err        error
		refundAcc  sdk.AccAddress
		ackFee     sdk.Coins
		receiveFee sdk.Coins
		timeoutFee sdk.Coins
	)

	// refundAcc does not have balance for the following Coin
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
				// this acc does not exist on chainA
				refundAcc = suite.chainB.SenderAccount.GetAddress()
			}, false,
		},
		{
			"ackFee balance not found", func() {
				ackFee = invalidCoins
			}, false,
		},
		{
			"receive balance not found", func() {
				receiveFee = invalidCoins
			}, false,
		},
		{
			"timeout balance not found", func() {
				timeoutFee = invalidCoins
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// setup
			refundAcc = suite.chainA.SenderAccount.GetAddress()
			ackFee = validCoins
			receiveFee = validCoins2
			timeoutFee = validCoins3
			packetId := &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(1)}

			tc.malleate()
			fee := types.Fee{ackFee, receiveFee, timeoutFee}
			identifiedPacketFee := &types.IdentifiedPacketFee{PacketId: packetId, Fee: fee, RefundAddress: refundAcc.String(), Relayers: []string{}}

			// refundAcc balance before escrow
			originalBal := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)

			// escrow the packet fee
			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedPacketFee)

			if tc.expPass {
				feeInEscrow, _ := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeInEscrow(suite.chainA.GetContext(), packetId)
				// check if the escrowed fee is set in state
				suite.Require().True(feeInEscrow.Fee.AckFee.IsEqual(fee.AckFee))
				suite.Require().True(feeInEscrow.Fee.ReceiveFee.IsEqual(fee.ReceiveFee))
				suite.Require().True(feeInEscrow.Fee.TimeoutFee.IsEqual(fee.TimeoutFee))
				// check if the fee is escrowed correctly
				hasBalance := suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(600)})
				suite.Require().True(hasBalance)
				expectedBal := originalBal.Amount.Sub(sdk.NewInt(600))
				// check if the refund acc has sent the fee
				hasBalance = suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), refundAcc, sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: expectedBal})
				suite.Require().True(hasBalance)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestDistributeFee() {
	var (
		err            error
		ackFee         sdk.Coins
		receiveFee     sdk.Coins
		timeoutFee     sdk.Coins
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
			reverseRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
			forwardRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

			ackFee = validCoins
			receiveFee = validCoins2
			timeoutFee = validCoins3
			packetId = &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: validSeq}
			fee := types.Fee{receiveFee, ackFee, timeoutFee}

			// escrow the packet fee & store the fee in state
			identifiedPacketFee := types.IdentifiedPacketFee{PacketId: packetId, Fee: fee, RefundAddress: refundAcc.String(), Relayers: []string{}}

			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), &identifiedPacketFee)
			suite.Require().NoError(err)

			tc.malleate()

			// refundAcc balance after escrow
			refundAccBal := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)

			err = suite.chainA.GetSimApp().IBCFeeKeeper.DistributeFee(suite.chainA.GetContext(), refundAcc, forwardRelayer, reverseRelayer, packetId)

			if tc.expPass {
				suite.Require().NoError(err)
				hasFeeInEscrow := suite.chainA.GetSimApp().IBCFeeKeeper.HasFeeInEscrow(suite.chainA.GetContext(), packetId)
				// there should no longer be a fee in escrow for this packet
				suite.Require().False(hasFeeInEscrow)
				// check if the reverse relayer is paid
				hasBalance := suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), reverseRelayer, ackFee[0])
				suite.Require().True(hasBalance)
				// check if the forward relayer is paid
				hasBalance = suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), forwardRelayer, receiveFee[0])
				suite.Require().True(hasBalance)
				// check if the refund acc has been refunded the timeoutFee
				expectedRefundAccBal := refundAccBal.Add(timeoutFee[0])
				hasBalance = suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), refundAcc, expectedRefundAccBal)
				suite.Require().True(hasBalance)
				// check the module acc wallet is now empty
				hasBalance = suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(0)})
				suite.Require().True(hasBalance)

				suite.Require().NoError(err)

			} else {
				suite.Require().Error(err)
				invalidPacketID := &channeltypes.PacketId{PortId: types.PortKey, ChannelId: validChannelId, Sequence: 1}
				hasFeeInEscrow := suite.chainA.GetSimApp().IBCFeeKeeper.HasFeeInEscrow(suite.chainA.GetContext(), invalidPacketID)
				// there should still be a fee in escrow for this packet
				suite.Require().True(hasFeeInEscrow)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestDistributeTimeoutFee() {
	var (
		err        error
		ackFee     sdk.Coins
		receiveFee sdk.Coins
		timeoutFee sdk.Coins
		packetId   *channeltypes.PacketId
		refundAcc  sdk.AccAddress
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
			timeoutRelayer := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

			ackFee = validCoins
			receiveFee = validCoins2
			timeoutFee = validCoins3
			packetId = &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: validSeq}
			fee := types.Fee{receiveFee, ackFee, timeoutFee}

			// escrow the packet fee & store the fee in state
			identifiedPacketFee := types.IdentifiedPacketFee{PacketId: packetId, Fee: fee, RefundAddress: refundAcc.String(), Relayers: []string{}}
			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), &identifiedPacketFee)
			suite.Require().NoError(err)

			tc.malleate()

			// refundAcc balance after escrow
			refundAccBal := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)

			err = suite.chainA.GetSimApp().IBCFeeKeeper.DistributeFeeTimeout(suite.chainA.GetContext(), refundAcc, timeoutRelayer, packetId)

			if tc.expPass {
				suite.Require().NoError(err)
				hasFeeInEscrow := suite.chainA.GetSimApp().IBCFeeKeeper.HasFeeInEscrow(suite.chainA.GetContext(), packetId)
				// there should no longer be a fee in escrow for this packet
				suite.Require().False(hasFeeInEscrow)
				// check if the timeoutRelayer has been paid
				hasBalance := suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), timeoutRelayer, timeoutFee[0])
				suite.Require().True(hasBalance)
				// check if the refund acc has been refunded the recv & ack fees
				expectedRefundAccBal := refundAccBal.Add(ackFee[0])
				expectedRefundAccBal = refundAccBal.Add(receiveFee[0])
				hasBalance = suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), refundAcc, expectedRefundAccBal)
				suite.Require().True(hasBalance)
				// check the module acc wallet is now empty
				hasBalance = suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(0)})
				suite.Require().True(hasBalance)

			} else {
				suite.Require().Error(err)
				invalidPacketID := &channeltypes.PacketId{PortId: types.PortKey, ChannelId: validChannelId, Sequence: 1}
				hasFeeInEscrow := suite.chainA.GetSimApp().IBCFeeKeeper.HasFeeInEscrow(suite.chainA.GetContext(), invalidPacketID)
				// there should still be a fee in escrow for this packet
				suite.Require().True(hasFeeInEscrow)
			}
		})
	}
}
