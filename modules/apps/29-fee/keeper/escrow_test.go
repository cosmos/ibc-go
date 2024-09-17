package keeper_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/mock"
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
		packetFees        []types.PacketFee
		fee               types.Fee
	)

	testCases := []struct {
		name      string
		malleate  func()
		expResult func()
	}{
		{
			"success",
			func() {
				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}
			},
			func() {
				// check if fees has been deleted
				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
				suite.Require().False(suite.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(suite.chainA.GetContext(), packetID))

				// check if the reverse relayer is paid
				expectedReverseAccBal := reverseRelayerBal.Add(defaultAckFee[0]).Add(defaultAckFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), reverseRelayer, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedReverseAccBal, balance)

				// check if the forward relayer is paid
				forward, err := sdk.AccAddressFromBech32(forwardRelayer)
				suite.Require().NoError(err)

				expectedForwardAccBal := forwardRelayerBal.Add(defaultRecvFee[0]).Add(defaultRecvFee[0])
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), forward, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedForwardAccBal, balance)

				// check if the refund amount is zero
				expectedRefundAccBal := refundAccBal
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)

				// check the module acc wallet is now empty
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(0)), balance)
			},
		},
		{
			"success: refund timeout_fee - (recv_fee + ack_fee)",
			func() {
				// set the timeout fee to be greater than recv + ack fee so that the refund amount is non-zero
				fee.TimeoutFee = fee.Total().Add(ibctesting.TestCoin)

				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}
			},
			func() {
				// check if fees has been deleted
				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
				suite.Require().False(suite.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(suite.chainA.GetContext(), packetID))

				// check if the reverse relayer is paid
				expectedReverseAccBal := reverseRelayerBal.Add(defaultAckFee[0]).Add(defaultAckFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), reverseRelayer, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedReverseAccBal, balance)

				// check if the forward relayer is paid
				forward, err := sdk.AccAddressFromBech32(forwardRelayer)
				suite.Require().NoError(err)

				expectedForwardAccBal := forwardRelayerBal.Add(defaultRecvFee[0]).Add(defaultRecvFee[0])
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), forward, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedForwardAccBal, balance)

				// check if the refund amount is correct
				refundCoins := fee.Total().Sub(defaultRecvFee[0]).Sub(defaultAckFee[0]).MulInt(sdkmath.NewInt(2))
				expectedRefundAccBal := refundAccBal.Add(refundCoins[0])
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)

				// check the module acc wallet is now empty
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(0)), balance)
			},
		},
		{
			"success: refund account is module account",
			func() {
				// set the timeout fee to be greater than recv + ack fee so that the refund amount is non-zero
				fee.TimeoutFee = fee.Total().Add(ibctesting.TestCoin)

				refundAcc = suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(mock.ModuleName)

				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				// fund mock account
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), mock.ModuleName, packetFee.Fee.Total().Add(packetFee.Fee.Total()...))
				suite.Require().NoError(err)
			},
			func() {
				// check if the refund acc has been refunded the correct amount
				refundCoins := fee.Total().Sub(defaultRecvFee[0]).Sub(defaultAckFee[0]).MulInt(sdkmath.NewInt(2))
				expectedRefundAccBal := refundAccBal.Add(refundCoins[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"escrow account out of balance, fee module becomes locked - no distribution", func() {
				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				// pass in an extra packet fee
				packetFees = append(packetFees, packetFee)
			},
			func() {
				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)

				suite.Require().True(suite.chainA.GetSimApp().IBCFeeKeeper.IsLocked(suite.chainA.GetContext()))
				suite.Require().True(suite.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(suite.chainA.GetContext(), packetID))

				// check if the module acc contains all the fees
				expectedModuleAccBal := packetFee.Fee.Total().Add(packetFee.Fee.Total()...)
				balance := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress())
				suite.Require().Equal(expectedModuleAccBal, balance)
			},
		},
		{
			"invalid forward address",
			func() {
				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				forwardRelayer = "invalid address"
			},
			func() {
				// check if the refund acc has been refunded the recvFee
				expectedRefundAccBal := refundAccBal.Add(defaultRecvFee[0]).Add(defaultRecvFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid forward address: blocked address",
			func() {
				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				forwardRelayer = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
			},
			func() {
				// check if the refund acc has been refunded the timeoutFee & recvFee
				expectedRefundAccBal := refundAccBal.Add(defaultRecvFee[0]).Add(defaultRecvFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid receiver address: ack fee returned to sender",
			func() {
				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				reverseRelayer = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress()
			},
			func() {
				// check if the refund acc has been refunded the ackFee
				expectedRefundAccBal := refundAccBal.Add(defaultAckFee[0]).Add(defaultAckFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid refund address: no-op, timeout_fee - (recv_fee + ack_fee) remains in escrow",
			func() {
				// set the timeout fee to be greater than recv + ack fee so that the refund amount is non-zero
				fee.TimeoutFee = fee.Total().Add(ibctesting.TestCoin)

				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				packetFees[0].RefundAddress = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
				packetFees[1].RefundAddress = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
			},
			func() {
				// check if the module acc contains the timeoutFee
				refundCoins := fee.Total().Sub(defaultRecvFee[0]).Sub(defaultAckFee[0]).MulInt(sdkmath.NewInt(2))
				expectedModuleAccBal := sdk.NewCoin(sdk.DefaultBondDenom, refundCoins.AmountOf(sdk.DefaultBondDenom))
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(expectedModuleAccBal, balance)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()  // reset
			suite.path.Setup() // setup channel

			// setup accounts
			forwardRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
			reverseRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
			refundAcc = suite.chainA.SenderAccount.GetAddress()

			packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
			fee = types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

			tc.malleate()

			// escrow the packet fees & store the fees in state
			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))
			err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, packetFee.Fee.Total().Add(packetFee.Fee.Total()...))
			suite.Require().NoError(err)

			// fetch the account balances before fee distribution (forward, reverse, refund)
			forwardAccAddress, _ := sdk.AccAddressFromBech32(forwardRelayer)
			forwardRelayerBal = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), forwardAccAddress, sdk.DefaultBondDenom)
			reverseRelayerBal = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), reverseRelayer, sdk.DefaultBondDenom)
			refundAccBal = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)

			suite.chainA.GetSimApp().IBCFeeKeeper.DistributePacketFeesOnAcknowledgement(suite.chainA.GetContext(), forwardRelayer, reverseRelayer, packetFees, packetID)
			tc.expResult()
		})
	}
}

func (suite *KeeperTestSuite) TestDistributePacketFeesOnTimeout() {
	var (
		timeoutRelayer    sdk.AccAddress
		timeoutRelayerBal sdk.Coin
		refundAcc         sdk.AccAddress
		refundAccBal      sdk.Coin
		fee               types.Fee
		packetFee         types.PacketFee
		packetFees        []types.PacketFee
	)

	testCases := []struct {
		name      string
		malleate  func()
		expResult func()
	}{
		{
			"success: no refund",
			func() {},
			func() {
				// check if the timeout relayer is paid
				expectedTimeoutAccBal := timeoutRelayerBal.Add(defaultTimeoutFee[0]).Add(defaultTimeoutFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), timeoutRelayer, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedTimeoutAccBal, balance)

				// check if the refund amount is zero
				expectedRefundAccBal := refundAccBal
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)

				// check the module acc wallet is now empty
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(0)), balance)
			},
		},
		{
			"success: refund (recv_fee + ack_fee) - timeout_fee",
			func() {
				// set the recv + ack fee to be greater than timeout fee so that the refund amount is non-zero
				fee.RecvFee = fee.RecvFee.Add(ibctesting.TestCoin)
				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}
			},
			func() {
				// check if the timeout relayer is paid
				expectedTimeoutAccBal := timeoutRelayerBal.Add(defaultTimeoutFee[0]).Add(defaultTimeoutFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), timeoutRelayer, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedTimeoutAccBal, balance)

				// check if the refund amount is correct
				refundCoins := fee.Total().Sub(defaultTimeoutFee[0]).MulInt(sdkmath.NewInt(2))
				expectedRefundAccBal := refundAccBal.Add(refundCoins[0])
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)

				// check the module acc wallet is now empty
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(0)), balance)
			},
		},
		{
			"escrow account out of balance, fee module becomes locked - no distribution", func() {
				// pass in an extra packet fee
				packetFees = append(packetFees, packetFee)
			},
			func() {
				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)

				suite.Require().True(suite.chainA.GetSimApp().IBCFeeKeeper.IsLocked(suite.chainA.GetContext()))
				suite.Require().True(suite.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(suite.chainA.GetContext(), packetID))

				// check if the module acc contains all the fees
				expectedModuleAccBal := packetFee.Fee.Total().Add(packetFee.Fee.Total()...)
				balance := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress())
				suite.Require().Equal(expectedModuleAccBal, balance)
			},
		},
		{
			"invalid timeout relayer address: timeout fee returned to sender",
			func() {
				timeoutRelayer = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress()
			},
			func() {
				// check if the refund acc has been refunded the all the fees
				expectedRefundAccBal := sdk.Coins{refundAccBal}.Add(packetFee.Fee.Total()...).Add(packetFee.Fee.Total()...)[0]
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid refund address: no-op, (recv_fee + ack_fee) - timeout_fee remain in escrow",
			func() {
				// set the recv + ack fee to be greater than timeout fee so that the refund amount is non-zero
				fee.RecvFee = fee.RecvFee.Add(ibctesting.TestCoin)
				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				packetFees[0].RefundAddress = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
				packetFees[1].RefundAddress = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
			},
			func() {
				// check if the module acc contains the correct amount of fees
				refundCoins := fee.Total().Sub(defaultTimeoutFee[0]).MulInt(sdkmath.NewInt(2))

				expectedModuleAccBal := refundCoins[0]
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(expectedModuleAccBal, balance)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()  // reset
			suite.path.Setup() // setup channel

			// setup accounts
			timeoutRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
			refundAcc = suite.chainA.SenderAccount.GetAddress()

			packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
			fee = types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

			// escrow the packet fees & store the fees in state
			packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
			packetFees = []types.PacketFee{packetFee, packetFee}

			tc.malleate()

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))
			err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total().Add(fee.Total()...))
			suite.Require().NoError(err)

			// fetch the account balances before fee distribution (forward, reverse, refund)
			timeoutRelayerBal = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), timeoutRelayer, sdk.DefaultBondDenom)
			refundAccBal = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)

			suite.chainA.GetSimApp().IBCFeeKeeper.DistributePacketFeesOnTimeout(suite.chainA.GetContext(), timeoutRelayer, packetFees, packetID)

			tc.expResult()
		})
	}
}

func (suite *KeeperTestSuite) TestRefundFeesOnChannelClosure() {
	var (
		expIdentifiedPacketFees []types.IdentifiedPacketFees
		expEscrowBal            sdk.Coins
		expRefundBal            sdk.Coins
		refundAcc               sdk.AccAddress
	)

	fee := types.Fee{
		RecvFee:    defaultRecvFee,
		AckFee:     defaultAckFee,
		TimeoutFee: defaultTimeoutFee,
	}

	testCases := []struct {
		name      string
		malleate  func()
		expResult func(originalRefundBal, originalEscrowBal, refundBal, escrowBal sdk.Coins)
	}{
		{
			"success", func() {
				for i := 1; i < 6; i++ {
					// store the fee in state & update escrow account balance
					packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(i))
					packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})

					suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)

					err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
					suite.Require().NoError(err)
				}
			}, func(_, _, refundBal, escrowBal sdk.Coins) {
				suite.Require().False(suite.chainA.GetSimApp().IBCFeeKeeper.IsLocked(suite.chainA.GetContext()))
				suite.Require().Empty(suite.chainA.GetSimApp().IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID))
				suite.Require().Empty(escrowBal)
				suite.Require().Equal(expRefundBal, refundBal)
			},
		},
		{
			"success with undistributed packet fees on a different channel", func() {
				for i := 1; i < 6; i++ {
					// store the fee in state & update escrow account balance
					packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(i))
					packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
					suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)

					err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
					suite.Require().NoError(err)
				}

				// set packet fee for a different channel
				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, "channel-1", uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, "channel-1")

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				suite.Require().NoError(err)

				expEscrowBal = fee.Total()
				expRefundBal = expRefundBal.Sub(fee.Total()...)
			}, func(_, _, refundBal, escrowBal sdk.Coins) {
				suite.Require().False(suite.chainA.GetSimApp().IBCFeeKeeper.IsLocked(suite.chainA.GetContext()))
				suite.Require().Empty(suite.chainA.GetSimApp().IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID))
				suite.Require().Equal(expEscrowBal, escrowBal)
				suite.Require().Equal(expRefundBal, refundBal)
			},
		},
		{
			"escrow account empty, module should become locked", func() {
				// store the fee in state without updating escrow account balance
				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
				identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)

				expIdentifiedPacketFees = []types.IdentifiedPacketFees{identifiedPacketFees}
			}, func(originalRefundBal, originalEscrowBal, refundBal, escrowBal sdk.Coins) {
				suite.Require().True(suite.chainA.GetSimApp().IBCFeeKeeper.IsLocked(suite.chainA.GetContext()))
				suite.Require().Equal(expIdentifiedPacketFees, suite.chainA.GetSimApp().IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID))
				suite.Require().Equal(originalRefundBal, refundBal)
				suite.Require().Equal(originalEscrowBal, escrowBal)
			},
		},
		{
			"escrow account goes negative on second packet, module should become locked", func() {
				// store 2 fees in state
				packetID1 := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(1))
				packetID2 := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(2))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
				identifiedPacketFee1 := types.NewIdentifiedPacketFees(packetID1, packetFees.PacketFees)
				identifiedPacketFee2 := types.NewIdentifiedPacketFees(packetID2, packetFees.PacketFees)

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID1, packetFees)
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID2, packetFees)

				// update escrow account balance for 1 fee
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				suite.Require().NoError(err)

				expIdentifiedPacketFees = []types.IdentifiedPacketFees{identifiedPacketFee1, identifiedPacketFee2}
			}, func(originalRefundBal, originalEscrowBal, refundBal, escrowBal sdk.Coins) {
				suite.Require().True(suite.chainA.GetSimApp().IBCFeeKeeper.IsLocked(suite.chainA.GetContext()))
				suite.Require().Equal(expIdentifiedPacketFees, suite.chainA.GetSimApp().IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID))
				suite.Require().Equal(originalRefundBal, refundBal)
				suite.Require().Equal(originalEscrowBal, escrowBal)
			},
		},
		{
			"invalid refund acc address", func() {
				// store the fee in state & update escrow account balance
				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{
					types.NewPacketFee(fee, refundAcc.String(), nil), // this packet fee will be refunded, and will be deleted from state
					types.NewPacketFee(fee, "invalid refund address", nil),
				})
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)

				// escrow twice the fee amount to account for the packet to have been incentivized twice
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total().MulInt(sdkmath.NewInt(2)))
				suite.Require().NoError(err)

				// only the first packet fee should be refunded, the second should remain in state
				expIdentifiedPacketFees = []types.IdentifiedPacketFees{types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees[1:])}

				expEscrowBal = fee.Total()
				expRefundBal = expRefundBal.Sub(fee.Total()...)
			}, func(_, _, refundBal, escrowBal sdk.Coins) {
				suite.Require().False(suite.chainA.GetSimApp().IBCFeeKeeper.IsLocked(suite.chainA.GetContext()))
				suite.Require().Equal(expIdentifiedPacketFees, suite.chainA.GetSimApp().IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID))
				suite.Require().Equal(expEscrowBal, escrowBal)
				suite.Require().Equal(expRefundBal, refundBal)
			},
		},
		{
			"distributing to blocked address is skipped", func() {
				blockedAddr := suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()

				// store the fee in state & update escrow account balance
				packetID := channeltypes.NewPacketID(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{
					types.NewPacketFee(fee, refundAcc.String(), nil), // this packet fee will be refunded, and will be deleted from state
					types.NewPacketFee(fee, blockedAddr, nil),
				})
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)

				// escrow twice the fee amount to account for the packet to have been incentivized twice
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total().MulInt(sdkmath.NewInt(2)))
				suite.Require().NoError(err)

				// only the first packet fee should be refunded, the second should remain in state
				expIdentifiedPacketFees = []types.IdentifiedPacketFees{types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees[1:])}

				expEscrowBal = fee.Total()
				expRefundBal = expRefundBal.Sub(fee.Total()...)
			}, func(_, _, refundBal, escrowBal sdk.Coins) {
				suite.Require().False(suite.chainA.GetSimApp().IBCFeeKeeper.IsLocked(suite.chainA.GetContext()))
				suite.Require().Equal(expIdentifiedPacketFees, suite.chainA.GetSimApp().IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID))
				suite.Require().Equal(expEscrowBal, escrowBal)
				suite.Require().Equal(expRefundBal, refundBal)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()  // reset
			suite.path.Setup() // setup channel
			expIdentifiedPacketFees = []types.IdentifiedPacketFees(nil)

			refundAcc = suite.chainA.SenderAccount.GetAddress()
			moduleAcc := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress()

			// expected refund balance if the refunds are successful
			// NOTE: tc.malleate() should transfer from refund balance to correctly set the escrow balance
			expRefundBal = suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), refundAcc)

			tc.malleate()

			// original balance before RefundFeesOnChannelClosure
			originalRefundBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), refundAcc)
			originalEscrowBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), moduleAcc)

			// perform refund fees on channel closure
			err := suite.chainA.GetSimApp().IBCFeeKeeper.RefundFeesOnChannelClosure(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
			suite.NoError(err)

			// refundAcc balance after RefundFeesOnChannelClosure
			refundBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), refundAcc)
			escrowBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), moduleAcc)

			tc.expResult(originalRefundBal, originalEscrowBal, refundBal, escrowBal)
		})
	}
}
