package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *KeeperTestSuite) TestDistributeFee() {
	var (
		forwardRelayer    string
		forwardRelayerBal sdk.Coin
		reverseRelayer    sdk.AccAddress
		reverseRelayerBal sdk.Coin
		refundAcc         sdk.AccAddress
		refundAccBal      sdk.Coin
		packetFee         types.PacketFee
	)

	testCases := []struct {
		name      string
		malleate  func()
		expResult func()
	}{
		{
			"success",
			func() {},
			func() {
				// check if the reverse relayer is paid
				expectedReverseAccBal := reverseRelayerBal.Add(defaultAckFee[0]).Add(defaultAckFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), reverseRelayer, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedReverseAccBal, balance)

				// check if the forward relayer is paid
				forward, err := sdk.AccAddressFromBech32(forwardRelayer)
				suite.Require().NoError(err)

				expectedForwardAccBal := forwardRelayerBal.Add(defaultReceiveFee[0]).Add(defaultReceiveFee[0])
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), forward, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedForwardAccBal, balance)

				// check if the refund acc has been refunded the timeoutFee
				expectedRefundAccBal := refundAccBal.Add(defaultTimeoutFee[0].Add(defaultTimeoutFee[0]))
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)

				// check the module acc wallet is now empty
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(0)), balance)
			},
		},
		{
			"invalid forward address",
			func() {
				forwardRelayer = "invalid address"
			},
			func() {
				// check if the refund acc has been refunded the timeoutFee & recvFee
				expectedRefundAccBal := refundAccBal.Add(defaultTimeoutFee[0]).Add(defaultReceiveFee[0]).Add(defaultTimeoutFee[0]).Add(defaultReceiveFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid forward address: blocked address",
			func() {
				forwardRelayer = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
			},
			func() {
				// check if the refund acc has been refunded the timeoutFee & recvFee
				expectedRefundAccBal := refundAccBal.Add(defaultTimeoutFee[0]).Add(defaultReceiveFee[0]).Add(defaultTimeoutFee[0]).Add(defaultReceiveFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid receiver address: ack fee returned to sender",
			func() {
				reverseRelayer = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress()
			},
			func() {
				// check if the refund acc has been refunded the timeoutFee & ackFee
				expectedRefundAccBal := refundAccBal.Add(defaultTimeoutFee[0]).Add(defaultAckFee[0]).Add(defaultTimeoutFee[0]).Add(defaultAckFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid refund address: no-op, timeout fee remains in escrow",
			func() {
				packetFee.RefundAddress = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
			},
			func() {
				// check if the module acc contains the timeoutFee
				expectedModuleAccBal := sdk.NewCoin(sdk.DefaultBondDenom, defaultTimeoutFee.Add(defaultTimeoutFee...).AmountOf(sdk.DefaultBondDenom))
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(expectedModuleAccBal, balance)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()                   // reset
			suite.coordinator.Setup(suite.path) // setup channel

			// setup accounts
			forwardRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
			reverseRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
			refundAcc = suite.chainA.SenderAccount.GetAddress()

			packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
			fee := types.NewFee(defaultReceiveFee, defaultAckFee, defaultTimeoutFee)

			// escrow the packet fee & store the fee in state
			packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})

			packetFees := []types.PacketFee{packetFee, packetFee}
			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))
			suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, packetFee.Fee.Total().Add(packetFee.Fee.Total()...))

			tc.malleate()

			// fetch the account balances before fee distribution (forward, reverse, refund)
			forwardAccAddress, _ := sdk.AccAddressFromBech32(forwardRelayer)
			forwardRelayerBal = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), forwardAccAddress, sdk.DefaultBondDenom)
			reverseRelayerBal = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), reverseRelayer, sdk.DefaultBondDenom)
			refundAccBal = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)

			suite.chainA.GetSimApp().IBCFeeKeeper.DistributePacketFees(suite.chainA.GetContext(), forwardRelayer, reverseRelayer, []types.PacketFee{packetFee, packetFee})

			tc.expResult()
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

	packetFees := []types.PacketFee{packetFee, packetFee}
	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))
	suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, packetFee.Fee.Total().Add(packetFee.Fee.Total()...))

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

func (suite *KeeperTestSuite) TestRefundFeesOnChannel() {
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

		suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))
		suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, packetFee.Fee.Total())
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

	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))
	suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, packetFee.Fee.Total())

	// check that refunding all fees on channel-0 refunds all fees except for fee on channel-1
	err := suite.chainA.GetSimApp().IBCFeeKeeper.RefundFeesOnChannel(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
	suite.Require().NoError(err, "refund fees returned unexpected error")

	// add fee sent to channel-1 to after balance to recover original balance
	afterBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), refundAcc)
	suite.Require().Equal(prevBal, afterBal.Add(fee.RecvFee...).Add(fee.AckFee...).Add(fee.TimeoutFee...), "refund account not back to original balance after refunding all tokens")

	// create escrow and then change module account balance to cause error on refund
	packetID = channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(6))

	packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})

	suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))
	suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, packetFee.Fee.Total())

	suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, refundAcc, fee.TimeoutFee)

	err = suite.chainA.GetSimApp().IBCFeeKeeper.RefundFeesOnChannel(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
	suite.Require().Error(err, "refund fees returned no error with insufficient balance on module account")
}
