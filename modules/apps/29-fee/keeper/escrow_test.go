package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/testing/mock"
)

func (s *KeeperTestSuite) TestDistributeFee() {
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
				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)
				s.Require().False(s.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(s.chainA.GetContext(), packetID))

				// check if the reverse relayer is paid
				expectedReverseAccBal := reverseRelayerBal.Add(defaultAckFee[0]).Add(defaultAckFee[0])
				balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), reverseRelayer, sdk.DefaultBondDenom)
				s.Require().Equal(expectedReverseAccBal, balance)

				// check if the forward relayer is paid
				forward, err := sdk.AccAddressFromBech32(forwardRelayer)
				s.Require().NoError(err)

				expectedForwardAccBal := forwardRelayerBal.Add(defaultRecvFee[0]).Add(defaultRecvFee[0])
				balance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), forward, sdk.DefaultBondDenom)
				s.Require().Equal(expectedForwardAccBal, balance)

				// check if the refund acc has been refunded the timeoutFee
				expectedRefundAccBal := refundAccBal.Add(defaultTimeoutFee[0].Add(defaultTimeoutFee[0]))
				balance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				s.Require().Equal(expectedRefundAccBal, balance)

				// check the module acc wallet is now empty
				balance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				s.Require().Equal(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(0)), balance)
			},
		},
		{
			"success: refund account is module account",
			func() {
				refundAcc = s.chainA.GetSimApp().AccountKeeper.GetModuleAddress(mock.ModuleName)

				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				// fund mock account
				err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), mock.ModuleName, packetFee.Fee.Total().Add(packetFee.Fee.Total()...))
				s.Require().NoError(err)
			},
			func() {
				// check if the refund acc has been refunded the timeoutFee
				expectedRefundAccBal := refundAccBal.Add(defaultTimeoutFee[0]).Add(defaultTimeoutFee[0])
				balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				s.Require().Equal(expectedRefundAccBal, balance)
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
				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)

				s.Require().True(s.chainA.GetSimApp().IBCFeeKeeper.IsLocked(s.chainA.GetContext()))
				s.Require().True(s.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(s.chainA.GetContext(), packetID))

				// check if the module acc contains all the fees
				expectedModuleAccBal := packetFee.Fee.Total().Add(packetFee.Fee.Total()...)
				balance := s.chainA.GetSimApp().BankKeeper.GetAllBalances(s.chainA.GetContext(), s.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress())
				s.Require().Equal(expectedModuleAccBal, balance)
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
				balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				s.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid forward address: blocked address",
			func() {
				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				forwardRelayer = s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
			},
			func() {
				// check if the refund acc has been refunded the timeoutFee & recvFee
				expectedRefundAccBal := refundAccBal.Add(defaultTimeoutFee[0]).Add(defaultRecvFee[0]).Add(defaultTimeoutFee[0]).Add(defaultRecvFee[0])
				balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				s.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid receiver address: ack fee returned to sender",
			func() {
				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				reverseRelayer = s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress()
			},
			func() {
				// check if the refund acc has been refunded the timeoutFee & ackFee
				expectedRefundAccBal := refundAccBal.Add(defaultTimeoutFee[0]).Add(defaultAckFee[0]).Add(defaultTimeoutFee[0]).Add(defaultAckFee[0])
				balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				s.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid refund address: no-op, timeout fee remains in escrow",
			func() {
				packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
				packetFees = []types.PacketFee{packetFee, packetFee}

				packetFees[0].RefundAddress = s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
				packetFees[1].RefundAddress = s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
			},
			func() {
				// check if the module acc contains the timeoutFee
				expectedModuleAccBal := sdk.NewCoin(sdk.DefaultBondDenom, defaultTimeoutFee.Add(defaultTimeoutFee...).AmountOf(sdk.DefaultBondDenom))
				balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				s.Require().Equal(expectedModuleAccBal, balance)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()               // reset
			s.coordinator.Setup(s.path) // setup channel

			// setup accounts
			forwardRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
			reverseRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
			refundAcc = s.chainA.SenderAccount.GetAddress()

			packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)
			fee = types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

			tc.malleate()

			// escrow the packet fees & store the fees in state
			s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))
			err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, packetFee.Fee.Total().Add(packetFee.Fee.Total()...))
			s.Require().NoError(err)

			// fetch the account balances before fee distribution (forward, reverse, refund)
			forwardAccAddress, _ := sdk.AccAddressFromBech32(forwardRelayer)
			forwardRelayerBal = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), forwardAccAddress, sdk.DefaultBondDenom)
			reverseRelayerBal = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), reverseRelayer, sdk.DefaultBondDenom)
			refundAccBal = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)

			s.chainA.GetSimApp().IBCFeeKeeper.DistributePacketFeesOnAcknowledgement(s.chainA.GetContext(), forwardRelayer, reverseRelayer, packetFees, packetID)
			tc.expResult()
		})
	}
}

func (s *KeeperTestSuite) TestDistributePacketFeesOnTimeout() {
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
				balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), timeoutRelayer, sdk.DefaultBondDenom)
				s.Require().Equal(expectedTimeoutAccBal, balance)

				// check if the refund acc has been refunded the recv/ack fees
				expectedRefundAccBal := refundAccBal.Add(defaultAckFee[0]).Add(defaultAckFee[0]).Add(defaultRecvFee[0]).Add(defaultRecvFee[0])
				balance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				s.Require().Equal(expectedRefundAccBal, balance)

				// check the module acc wallet is now empty
				balance = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				s.Require().Equal(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(0)), balance)
			},
		},
		{
			"escrow account out of balance, fee module becomes locked - no distribution", func() {
				// pass in an extra packet fee
				packetFees = append(packetFees, packetFee)
			},
			func() {
				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)

				s.Require().True(s.chainA.GetSimApp().IBCFeeKeeper.IsLocked(s.chainA.GetContext()))
				s.Require().True(s.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(s.chainA.GetContext(), packetID))

				// check if the module acc contains all the fees
				expectedModuleAccBal := packetFee.Fee.Total().Add(packetFee.Fee.Total()...)
				balance := s.chainA.GetSimApp().BankKeeper.GetAllBalances(s.chainA.GetContext(), s.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress())
				s.Require().Equal(expectedModuleAccBal, balance)
			},
		},
		{
			"invalid timeout relayer address: timeout fee returned to sender",
			func() {
				timeoutRelayer = s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress()
			},
			func() {
				// check if the refund acc has been refunded the all the fees
				expectedRefundAccBal := sdk.Coins{refundAccBal}.Add(packetFee.Fee.Total()...).Add(packetFee.Fee.Total()...)[0]
				balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)
				s.Require().Equal(expectedRefundAccBal, balance)
			},
		},
		{
			"invalid refund address: no-op, recv and ack fees remain in escrow",
			func() {
				packetFees[0].RefundAddress = s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
				packetFees[1].RefundAddress = s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()
			},
			func() {
				// check if the module acc contains the timeoutFee
				expectedModuleAccBal := sdk.NewCoin(sdk.DefaultBondDenom, defaultRecvFee.Add(defaultRecvFee[0]).Add(defaultAckFee[0]).Add(defaultAckFee[0]).AmountOf(sdk.DefaultBondDenom))
				balance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), s.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress(), sdk.DefaultBondDenom)
				s.Require().Equal(expectedModuleAccBal, balance)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()               // reset
			s.coordinator.Setup(s.path) // setup channel

			// setup accounts
			timeoutRelayer = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
			refundAcc = s.chainA.SenderAccount.GetAddress()

			packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)
			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)

			// escrow the packet fees & store the fees in state
			packetFee = types.NewPacketFee(fee, refundAcc.String(), []string{})
			packetFees = []types.PacketFee{packetFee, packetFee}

			s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))
			err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, packetFee.Fee.Total().Add(packetFee.Fee.Total()...))
			s.Require().NoError(err)

			tc.malleate()

			// fetch the account balances before fee distribution (forward, reverse, refund)
			timeoutRelayerBal = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), timeoutRelayer, sdk.DefaultBondDenom)
			refundAccBal = s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAcc, sdk.DefaultBondDenom)

			s.chainA.GetSimApp().IBCFeeKeeper.DistributePacketFeesOnTimeout(s.chainA.GetContext(), timeoutRelayer, packetFees, packetID)

			tc.expResult()
		})
	}
}

func (s *KeeperTestSuite) TestRefundFeesOnChannelClosure() {
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
					packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, uint64(i))
					packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
					identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

					s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, packetFees)

					err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
					s.Require().NoError(err)

					expIdentifiedPacketFees = append(expIdentifiedPacketFees, identifiedPacketFees)
				}
			}, true,
		},
		{
			"success with undistributed packet fees on a different channel", func() {
				for i := 1; i < 6; i++ {
					// store the fee in state & update escrow account balance
					packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, uint64(i))
					packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
					identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

					s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, packetFees)

					err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
					s.Require().NoError(err)

					expIdentifiedPacketFees = append(expIdentifiedPacketFees, identifiedPacketFees)
				}

				// set packet fee for a different channel
				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, "channel-1", uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
				s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, "channel-1")

				s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, packetFees)
				err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				s.Require().NoError(err)

				expEscrowBal = fee.Total()
				expRefundBal = expRefundBal.Sub(fee.Total()...)
			}, true,
		},
		{
			"escrow account empty, module should become locked", func() {
				locked = true

				// store the fee in state without updating escrow account balance
				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
				identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

				s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, packetFees)

				expIdentifiedPacketFees = []types.IdentifiedPacketFees{identifiedPacketFees}
			},
			true,
		},
		{
			"escrow account goes negative on second packet, module should become locked", func() {
				locked = true

				// store 2 fees in state
				packetID1 := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, uint64(1))
				packetID2 := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, uint64(2))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, refundAcc.String(), nil)})
				identifiedPacketFee1 := types.NewIdentifiedPacketFees(packetID1, packetFees.PacketFees)
				identifiedPacketFee2 := types.NewIdentifiedPacketFees(packetID2, packetFees.PacketFees)

				s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID1, packetFees)
				s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID2, packetFees)

				// update escrow account balance for 1 fee
				err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				s.Require().NoError(err)

				expIdentifiedPacketFees = []types.IdentifiedPacketFees{identifiedPacketFee1, identifiedPacketFee2}
			}, true,
		},
		{
			"invalid refund acc address", func() {
				// store the fee in state & update escrow account balance
				expectEscrowFeesToBeDeleted = false
				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, "invalid refund address", nil)})
				identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

				s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, packetFees)

				err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				s.Require().NoError(err)

				expIdentifiedPacketFees = []types.IdentifiedPacketFees{identifiedPacketFees}

				expEscrowBal = fee.Total()
				expRefundBal = expRefundBal.Sub(fee.Total()...)
			}, true,
		},
		{
			"distributing to blocked address is skipped", func() {
				expectEscrowFeesToBeDeleted = false
				blockedAddr := s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress().String()

				// store the fee in state & update escrow account balance
				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, blockedAddr, nil)})
				identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

				s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, packetFees)

				err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				s.Require().NoError(err)

				expIdentifiedPacketFees = []types.IdentifiedPacketFees{identifiedPacketFees}

				expEscrowBal = fee.Total()
				expRefundBal = expRefundBal.Sub(fee.Total()...)
			}, true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()               // reset
			s.coordinator.Setup(s.path) // setup channel
			expIdentifiedPacketFees = []types.IdentifiedPacketFees{}
			expEscrowBal = sdk.Coins{}
			locked = false
			expectEscrowFeesToBeDeleted = true

			// setup
			refundAcc = s.chainA.SenderAccount.GetAddress()
			moduleAcc := s.chainA.GetSimApp().IBCFeeKeeper.GetFeeModuleAddress()

			// expected refund balance if the refunds are successful
			// NOTE: tc.malleate() should transfer from refund balance to correctly set the escrow balance
			expRefundBal = s.chainA.GetSimApp().BankKeeper.GetAllBalances(s.chainA.GetContext(), refundAcc)

			fee = types.Fee{
				RecvFee:    defaultRecvFee,
				AckFee:     defaultAckFee,
				TimeoutFee: defaultTimeoutFee,
			}

			tc.malleate()

			// refundAcc balance before distribution
			originalRefundBal := s.chainA.GetSimApp().BankKeeper.GetAllBalances(s.chainA.GetContext(), refundAcc)
			originalEscrowBal := s.chainA.GetSimApp().BankKeeper.GetAllBalances(s.chainA.GetContext(), moduleAcc)

			err := s.chainA.GetSimApp().IBCFeeKeeper.RefundFeesOnChannelClosure(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)

			// refundAcc balance after RefundFeesOnChannelClosure
			refundBal := s.chainA.GetSimApp().BankKeeper.GetAllBalances(s.chainA.GetContext(), refundAcc)
			escrowBal := s.chainA.GetSimApp().BankKeeper.GetAllBalances(s.chainA.GetContext(), moduleAcc)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}

			s.Require().Equal(locked, s.chainA.GetSimApp().IBCFeeKeeper.IsLocked(s.chainA.GetContext()))

			if locked || !tc.expPass {
				// refund account and escrow account balances should remain unchanged
				s.Require().Equal(originalRefundBal, refundBal)
				s.Require().Equal(originalEscrowBal, escrowBal)

				// ensure none of the fees were deleted
				s.Require().Equal(expIdentifiedPacketFees, s.chainA.GetSimApp().IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID))
			} else {
				s.Require().Equal(expEscrowBal, escrowBal) // escrow balance should be empty
				s.Require().Equal(expRefundBal, refundBal) // all packets should have been refunded

				// all fees in escrow should be deleted if expected for this channel
				s.Require().Equal(expectEscrowFeesToBeDeleted, len(s.chainA.GetSimApp().IBCFeeKeeper.GetIdentifiedPacketFeesForChannel(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)) == 0)
			}
		})
	}
}
