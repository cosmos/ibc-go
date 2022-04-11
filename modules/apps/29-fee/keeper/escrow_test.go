package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *KeeperTestSuite) TestEscrowPacketFee() {
	var (
		err        error
		refundAcc  sdk.AccAddress
		ackFee     sdk.Coins
		receiveFee sdk.Coins
		timeoutFee sdk.Coins
		packetID   channeltypes.PacketId
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"success with existing packet fee", func() {
				fee := types.Fee{
					RecvFee:    receiveFee,
					AckFee:     ackFee,
					TimeoutFee: timeoutFee,
				}

				packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})
				feesInEscrow := types.NewPacketFees([]types.PacketFee{packetFee})

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, feesInEscrow)
			}, true,
		},
		{
			"fee not enabled on this channel", func() {
				packetID.ChannelId = "disabled_channel"
			}, false,
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
			suite.SetupTest()                   // reset
			suite.coordinator.Setup(suite.path) // setup channel

			// setup
			refundAcc = suite.chainA.SenderAccount.GetAddress()
			receiveFee = defaultReceiveFee
			ackFee = defaultAckFee
			timeoutFee = defaultTimeoutFee
			packetID = channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(1))

			tc.malleate()
			fee := types.Fee{
				RecvFee:    receiveFee,
				AckFee:     ackFee,
				TimeoutFee: timeoutFee,
			}
			packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})

			// refundAcc balance before escrow
			originalBal := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)

			// escrow the packet fee
			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)

			if tc.expPass {
				feesInEscrow, found := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeesInEscrow(suite.chainA.GetContext(), packetID)
				suite.Require().True(found)
				// check if the escrowed fee is set in state
				suite.Require().True(feesInEscrow.PacketFees[0].Fee.AckFee.IsEqual(fee.AckFee))
				suite.Require().True(feesInEscrow.PacketFees[0].Fee.RecvFee.IsEqual(fee.RecvFee))
				suite.Require().True(feesInEscrow.PacketFees[0].Fee.TimeoutFee.IsEqual(fee.TimeoutFee))
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
		reverseRelayer sdk.AccAddress
		forwardRelayer string
		refundAcc      sdk.AccAddress
	)

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
			"invalid forward address", func() {
				forwardRelayer = "invalid address"
			}, false,
		},
		{
			"invalid forward address: blocked address", func() {
				forwardRelayer = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()                   // reset
			suite.coordinator.Setup(suite.path) // setup channel

			// setup
			refundAcc = suite.chainA.SenderAccount.GetAddress()
			reverseRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
			forwardRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

			packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, validSeq)
			fee := types.Fee{
				RecvFee:    defaultReceiveFee,
				AckFee:     defaultAckFee,
				TimeoutFee: defaultTimeoutFee,
			}

			// escrow the packet fee & store the fee in state
			packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})

			err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
			suite.Require().NoError(err)
			// escrow a second packet fee to test with multiple fees distributed
			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
			suite.Require().NoError(err)

			tc.malleate()

			// refundAcc balance after escrow
			refundAccBal := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)

			suite.chainA.GetSimApp().IBCFeeKeeper.DistributePacketFees(suite.chainA.GetContext(), forwardRelayer, reverseRelayer, []types.PacketFee{packetFee, packetFee})

			if tc.expPass {
				// check if the reverse relayer is paid
				hasBalance := suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), reverseRelayer, fee.AckFee[0].Add(fee.AckFee[0]))
				suite.Require().True(hasBalance)

				// check if the forward relayer is paid
				forward, err := sdk.AccAddressFromBech32(forwardRelayer)
				suite.Require().NoError(err)
				hasBalance = suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), forward, fee.RecvFee[0].Add(fee.RecvFee[0]))
				suite.Require().True(hasBalance)

				// check if the refund acc has been refunded the timeoutFee
				expectedRefundAccBal := refundAccBal.Add(fee.TimeoutFee[0].Add(fee.TimeoutFee[0]))
				hasBalance = suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), refundAcc, expectedRefundAccBal)
				suite.Require().True(hasBalance)

				// check the module acc wallet is now empty
				hasBalance = suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(0)})
				suite.Require().True(hasBalance)
			} else {
				// check if the refund acc has been refunded the timeoutFee & onRecvFee
				expectedRefundAccBal := refundAccBal.Add(fee.TimeoutFee[0]).Add(fee.RecvFee[0]).Add(fee.TimeoutFee[0]).Add(fee.RecvFee[0])
				hasBalance := suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), refundAcc, expectedRefundAccBal)
				suite.Require().True(hasBalance)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestDistributeTimeoutFee() {
	suite.coordinator.Setup(suite.path) // setup channel

	// setup
	refundAcc := suite.chainA.SenderAccount.GetAddress()
	timeoutRelayer := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	packetID := channeltypes.NewPacketId(
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		1,
	)

	fee := types.Fee{
		RecvFee:    defaultReceiveFee,
		AckFee:     defaultAckFee,
		TimeoutFee: defaultTimeoutFee,
	}

	// escrow the packet fee & store the fee in state
	packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})

	err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
	suite.Require().NoError(err)
	// escrow a second packet fee to test with multiple fees distributed
	err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
	suite.Require().NoError(err)

	// refundAcc balance after escrow
	refundAccBal := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)

	suite.chainA.GetSimApp().IBCFeeKeeper.DistributePacketFeesOnTimeout(suite.chainA.GetContext(), timeoutRelayer, []types.PacketFee{packetFee, packetFee})

	// check if the timeoutRelayer has been paid
	hasBalance := suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), timeoutRelayer, fee.TimeoutFee[0])
	suite.Require().True(hasBalance)

	// check if the refund acc has been refunded the recv & ack fees
	expectedRefundAccBal := refundAccBal.Add(fee.AckFee[0]).Add(fee.AckFee[0])
	expectedRefundAccBal = refundAccBal.Add(fee.RecvFee[0]).Add(fee.RecvFee[0])
	hasBalance = suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), refundAcc, expectedRefundAccBal)
	suite.Require().True(hasBalance)

	// check the module acc wallet is now empty
	hasBalance = suite.chainA.GetSimApp().BankKeeper.HasBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(0)})
	suite.Require().True(hasBalance)
}

func (suite *KeeperTestSuite) TestRefundFeesOnChannelClosure() {
	suite.coordinator.Setup(suite.path)

	// setup
	refundAcc := suite.chainA.SenderAccount.GetAddress()

	// refundAcc balance before escrow
	prevBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), refundAcc)

	for i := 0; i < 5; i++ {
		packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(i))
		fee := types.Fee{
			RecvFee:    defaultReceiveFee,
			AckFee:     defaultAckFee,
			TimeoutFee: defaultTimeoutFee,
		}

		packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})
		suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
		err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
		suite.Require().NoError(err)
	}

	// send a packet over a different channel to ensure this fee is not refunded
	packetID := channeltypes.NewPacketId(ibctesting.MockFeePort, "channel-1", 1)
	fee := types.Fee{
		RecvFee:    defaultReceiveFee,
		AckFee:     defaultAckFee,
		TimeoutFee: defaultTimeoutFee,
	}

	packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, "channel-1")
	err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
	suite.Require().NoError(err)

	// check that refunding all fees on channel-0 refunds all fees except for fee on channel-1
	err = suite.chainA.GetSimApp().IBCFeeKeeper.RefundFeesOnChannelClosure(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
	suite.Require().NoError(err, "refund fees returned unexpected error")

	// add fee sent to channel-1 to after balance to recover original balance
	afterBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), refundAcc)
	suite.Require().Equal(prevBal, afterBal.Add(fee.RecvFee...).Add(fee.AckFee...).Add(fee.TimeoutFee...), "refund account not back to original balance after refunding all tokens")

	// create escrow and then change module account balance to cause error on refund
	packetID = channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(6))

	packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
	err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), packetID, packetFee)
	suite.Require().NoError(err)

	suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, refundAcc, fee.TimeoutFee)

	err = suite.chainA.GetSimApp().IBCFeeKeeper.RefundFeesOnChannelClosure(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
	suite.Require().Error(err, "refund fees returned no error with insufficient balance on module account")
}
