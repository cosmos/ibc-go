package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v4/testing/mock"
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
				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
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
			"success: refund account is module account",
			func() {
				refundAcc = suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(mock.ModuleName)

				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				// fund mock account
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), mock.ModuleName, packetFee.Fee.Total().Add(packetFee.Fee.Total()...))
				suite.Require().NoError(err)
			},
			func() {
				// check if the refund acc has been refunded the timeoutFee
				expectedRefundAccBal := defaultTimeoutFee[0].Add(defaultTimeoutFee[0])
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
				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)

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
				// check if the refund acc has been refunded the timeoutFee & recvFee
				expectedRefundAccBal := refundAccBal.Add(defaultTimeoutFee[0]).Add(defaultRecvFee[0]).Add(defaultTimeoutFee[0]).Add(defaultRecvFee[0])
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
				expectedRefundAccBal := refundAccBal.Add(defaultTimeoutFee[0]).Add(defaultRecvFee[0]).Add(defaultTimeoutFee[0]).Add(defaultRecvFee[0])
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
				// check if the refund acc has been refunded the timeoutFee & ackFee
				expectedRefundAccBal := refundAccBal.Add(defaultTimeoutFee[0]).Add(defaultAckFee[0]).Add(defaultTimeoutFee[0]).Add(defaultAckFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid refund address: no-op, timeout fee remains in escrow",
			func() {
				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				packetFees[0].RefundAddress = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
				packetFees[1].RefundAddress = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
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
		packetFee         types.PacketFee
		packetFees        []types.PacketFee
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
				// check if the timeout relayer is paid
				expectedTimeoutAccBal := timeoutRelayerBal.Add(defaultTimeoutFee[0]).Add(defaultTimeoutFee[0])
				balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), timeoutRelayer, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedTimeoutAccBal, balance)

				// check if the refund acc has been refunded the recv/ack fees
				expectedRefundAccBal := refundAccBal.Add(defaultAckFee[0]).Add(defaultAckFee[0]).Add(defaultRecvFee[0]).Add(defaultRecvFee[0])
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				suite.Require().Equal(expectedRefundAccBal, balance)

				// check the module acc wallet is now empty
				balance = suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				suite.Require().Equal(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(0)), balance)
			},
		},
		{
			"escrow account out of balance, fee module becomes locked - no distribution", func() {
				// pass in an extra packet fee
				packetFees = append(packetFees, packetFee)
			},
			func() {
				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)

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
			"invalid refund address: no-op, recv and ack fees remain in escrow",
			func() {
				packetFees[0].RefundAddress = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
				packetFees[1].RefundAddress = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
			},
			func() {
				// check if the module acc contains the timeoutFee
				expectedModuleAccBal := sdk.NewCoin(sdk.DefaultBondDenom, defaultRecvFee.Add(defaultRecvFee[0]).Add(defaultAckFee[0]).Add(defaultAckFee[0]).AmountOf(sdk.DefaultBondDenom))
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
			timeoutRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
			refundAcc = suite.chainA.SenderAccount.GetAddress()

			packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

			// escrow the packet fees & store the fees in state
			packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
			packetFees = []types.PacketFee{packetFee, packetFee}

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))
			err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, packetFee.Fee.Total().Add(packetFee.Fee.Total()...))
			suite.Require().NoError(err)

			tc.malleate()

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
		expIdentifiedPacketFees     []types.IdentifiedPacketFees
		expEscrowBal                sdk.Coins
		expRefundBal                sdk.Coins
		refundAcc                   sdk.AccAddress
		fee                         types.Fee
		locked                      bool
		expectEscrowFeesToBeDeleted bool
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {
				for i := 1; i < 6; i++ {
					// store the fee in state & update escrow account balance
					packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(i))
					packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
					identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

					suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)

					err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
					suite.Require().NoError(err)

					expIdentifiedPacketFees = append(expIdentifiedPacketFees, identifiedPacketFees)
				}
			}, true,
		},
		{
			"success with undistributed packet fees on a different channel", func() {
				for i := 1; i < 6; i++ {
					// store the fee in state & update escrow account balance
					packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(i))
					packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
					identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

					suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)

					err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
					suite.Require().NoError(err)

					expIdentifiedPacketFees = append(expIdentifiedPacketFees, identifiedPacketFees)
				}

				// set packet fee for a different channel
				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, "channel-1", uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, "channel-1")

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				suite.Require().NoError(err)

				expEscrowBal = fee.Total()
				expRefundBal = expRefundBal.Sub(fee.Total())
			}, true,
		},
		{
			"escrow account empty, module should become locked", func() {
				locked = true

				// store the fee in state without updating escrow account balance
				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
				identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)

				expIdentifiedPacketFees = []types.IdentifiedPacketFees{identifiedPacketFees}
			},
			true,
		},
		{
			"escrow account goes negative on second packet, module should become locked", func() {
				locked = true

				// store 2 fees in state
				packetID1 := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(1))
				packetID2 := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(2))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
				identifiedPacketFee1 := types.NewIdentifiedPacketFees(packetID1, packetFees.PacketFees)
				identifiedPacketFee2 := types.NewIdentifiedPacketFees(packetID2, packetFees.PacketFees)

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID1, packetFees)
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID2, packetFees)

				// update escrow account balance for 1 fee
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				suite.Require().NoError(err)

				expIdentifiedPacketFees = []types.IdentifiedPacketFees{identifiedPacketFee1, identifiedPacketFee2}
			}, true,
		},
		{
			"invalid refund acc address", func() {
				// store the fee in state & update escrow account balance
				expectEscrowFeesToBeDeleted = false
				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, "invalid refund address", nil)})
				identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)

				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				suite.Require().NoError(err)

				expIdentifiedPacketFees = []types.IdentifiedPacketFees{identifiedPacketFees}

				expEscrowBal = fee.Total()
				expRefundBal = expRefundBal.Sub(fee.Total())
			}, true,
		},
		{
			"distributing to blocked address is skipped", func() {
				expectEscrowFeesToBeDeleted = false
				blockedAddr := suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()

				// store the fee in state & update escrow account balance
				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, blockedAddr, nil)})
				identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)

				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				suite.Require().NoError(err)

				expIdentifiedPacketFees = []types.IdentifiedPacketFees{identifiedPacketFees}

				expEscrowBal = fee.Total()
				expRefundBal = expRefundBal.Sub(fee.Total())
			}, true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()                   // reset
			suite.coordinator.Setup(suite.path) // setup channel
			expIdentifiedPacketFees = []types.IdentifiedPacketFees{}
			expEscrowBal = sdk.Coins{}
			locked = false
			expectEscrowFeesToBeDeleted = true

			// setup
			refundAcc = suite.chainA.SenderAccount.GetAddress()
			moduleAcc := suite.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress()

			// expected refund balance if the refunds are successful
			// NOTE: tc.malleate() should transfer from refund balance to correctly set the escrow balance
			expRefundBal = suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), refundAcc)

			fee = types.Fee{
				RecvFee:    defaultRecvFee,
				AckFee:     defaultAckFee,
				TimeoutFee: defaultTimeoutFee,
			}

			tc.malleate()

			// refundAcc balance before distribution
			originalRefundBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), refundAcc)
			originalEscrowBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), moduleAcc)

			err := suite.chainA.GetSimApp().IBCFeeKeeper.RefundFeesOnChannelClosure(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)

			// refundAcc balance after RefundFeesOnChannelClosure
			refundBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), refundAcc)
			escrowBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), moduleAcc)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

			suite.Require().Equal(locked, suite.chainA.GetSimApp().IBCFeeKeeper.IsLocked(suite.chainA.GetContext()))

			if locked || !tc.expPass {
				// refund account and escrow account balances should remain unchanged
				suite.Require().Equal(originalRefundBal, refundBal)
				suite.Require().Equal(originalEscrowBal, escrowBal)

				// ensure none of the fees were deleted
				suite.Require().Equal(expIdentifiedPacketFees, suite.chainA.GetSimApp().IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID))
			} else {
				suite.Require().Equal(expEscrowBal, escrowBal) // escrow balance should be empty
				suite.Require().Equal(expRefundBal, refundBal) // all packets should have been refunded

				// all fees in escrow should be deleted if expected for this channel
				suite.Require().Equal(expectEscrowFeesToBeDeleted, len(suite.chainA.GetSimApp().IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)) == 0)
			}
		})
	}
}
