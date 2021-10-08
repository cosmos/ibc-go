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
			packetId := channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(1)}

			tc.malleate()
			fee := types.Fee{ackFee, receiveFee, timeoutFee}

			// escrow the packet fee
			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), refundAcc, fee, packetId)

			if tc.expPass {
				feeInEscrow, _ := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeInEscrow(suite.chainA.GetContext(), refundAcc.String(), packetId.ChannelId, packetId.Sequence)
				suite.Require().Equal(fee.AckFee, feeInEscrow.AckFee)
				suite.Require().Equal(fee.ReceiveFee, feeInEscrow.ReceiveFee)
				suite.Require().Equal(fee.TimeoutFee, feeInEscrow.TimeoutFee)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
