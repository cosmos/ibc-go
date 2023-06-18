package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestQueryIncentivizedPackets() {
	var (
		req             *types.QueryIncentivizedPacketsRequest
		expectedPackets []types.IdentifiedPacketFees
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

				fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
				packetFee := types.NewPacketFee(fee, s.chainA.SenderAccount.GetAddress().String(), []string(nil))

				for i := 0; i < 3; i++ {
					// escrow packet fees for three different packets
					packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, uint64(i+1))
					s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))

					expectedPackets = append(expectedPackets, types.NewIdentifiedPacketFees(packetID, []types.PacketFee{packetFee}))
				}

				req = &types.QueryIncentivizedPacketsRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
					QueryHeight: 0,
				}
			},
			true,
		},
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"empty pagination",
			func() {
				expectedPackets = nil
				req = &types.QueryIncentivizedPacketsRequest{}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			tc.malleate() // malleate mutates test data

			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.GetSimApp().IBCFeeKeeper.IncentivizedPackets(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expectedPackets, res.IncentivizedPackets)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryIncentivizedPacket() {
	var req *types.QueryIncentivizedPacketRequest

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
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"fees not found for packet id",
			func() {
				req = &types.QueryIncentivizedPacketRequest{
					PacketId:    channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100),
					QueryHeight: 0,
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)
			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, s.chainA.SenderAccount.GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryIncentivizedPacketRequest{
				PacketId:    packetID,
				QueryHeight: 0,
			}

			tc.malleate() // malleate mutates test data

			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.GetSimApp().IBCFeeKeeper.IncentivizedPacket(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(types.NewIdentifiedPacketFees(packetID, []types.PacketFee{packetFee, packetFee, packetFee}), res.IncentivizedPacket)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryIncentivizedPacketsForChannel() {
	var (
		req                     *types.QueryIncentivizedPacketsForChannelRequest
		expIdentifiedPacketFees []*types.IdentifiedPacketFees
	)

	fee := types.Fee{
		AckFee:     sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(100)}},
		RecvFee:    sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(100)}},
		TimeoutFee: sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(100)}},
	}

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty pagination",
			func() {
				expIdentifiedPacketFees = nil
				req = &types.QueryIncentivizedPacketsForChannelRequest{}
			},
			true,
		},
		{
			"success",
			func() {
				req = &types.QueryIncentivizedPacketsForChannelRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
					PortId:      ibctesting.MockFeePort,
					ChannelId:   ibctesting.FirstChannelID,
					QueryHeight: 0,
				}
			},
			true,
		},
		{
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"no packets for specified channel",
			func() {
				expIdentifiedPacketFees = nil
				req = &types.QueryIncentivizedPacketsForChannelRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
					PortId:      ibctesting.MockFeePort,
					ChannelId:   "channel-10",
					QueryHeight: 0,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			// setup
			refundAcc := s.chainA.SenderAccount.GetAddress()
			packetFee := types.NewPacketFee(fee, refundAcc.String(), nil)
			packetFees := types.NewPacketFees([]types.PacketFee{packetFee, packetFee, packetFee})

			identifiedFees1 := types.NewIdentifiedPacketFees(channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1), packetFees.PacketFees)
			identifiedFees2 := types.NewIdentifiedPacketFees(channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 2), packetFees.PacketFees)
			identifiedFees3 := types.NewIdentifiedPacketFees(channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 3), packetFees.PacketFees)

			expIdentifiedPacketFees = append(expIdentifiedPacketFees, &identifiedFees1, &identifiedFees2, &identifiedFees3)

			s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)
			for _, identifiedPacketFees := range expIdentifiedPacketFees {
				s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), identifiedPacketFees.PacketId, types.NewPacketFees(identifiedPacketFees.PacketFees))
			}

			tc.malleate()
			ctx := sdk.WrapSDKContext(s.chainA.GetContext())

			res, err := s.chainA.GetSimApp().IBCFeeKeeper.IncentivizedPacketsForChannel(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
				s.Require().Equal(expIdentifiedPacketFees, res.IncentivizedPackets)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryTotalRecvFees() {
	var req *types.QueryTotalRecvFeesRequest

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
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100)
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)

			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, s.chainA.SenderAccount.GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryTotalRecvFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.GetSimApp().IBCFeeKeeper.TotalRecvFees(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)

				// expected total is three times the default recv fee
				expectedFees := defaultRecvFee.Add(defaultRecvFee...).Add(defaultRecvFee...)
				s.Require().Equal(expectedFees, res.RecvFees)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryTotalAckFees() {
	var req *types.QueryTotalAckFeesRequest

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
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100)
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)

			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, s.chainA.SenderAccount.GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryTotalAckFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.GetSimApp().IBCFeeKeeper.TotalAckFees(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)

				// expected total is three times the default acknowledgement fee
				expectedFees := defaultAckFee.Add(defaultAckFee...).Add(defaultAckFee...)
				s.Require().Equal(expectedFees, res.AckFees)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryTotalTimeoutFees() {
	var req *types.QueryTotalTimeoutFeesRequest

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
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100)
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)

			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, s.chainA.SenderAccount.GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryTotalTimeoutFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.GetSimApp().IBCFeeKeeper.TotalTimeoutFees(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)

				// expected total is three times the default acknowledgement fee
				expectedFees := defaultTimeoutFee.Add(defaultTimeoutFee...).Add(defaultTimeoutFee...)
				s.Require().Equal(expectedFees, res.TimeoutFees)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryPayee() {
	var req *types.QueryPayeeRequest

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
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"payee address not found: invalid channel",
			func() {
				req.ChannelId = "invalid-channel-id" //nolint:goconst
			},
			false,
		},
		{
			"payee address not found: invalid relayer address",
			func() {
				req.Relayer = "invalid-addr" //nolint:goconst
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			pk := secp256k1.GenPrivKey().PubKey()
			expPayeeAddr := sdk.AccAddress(pk.Address())

			s.chainA.GetSimApp().IBCFeeKeeper.SetPayeeAddress(
				s.chainA.GetContext(),
				s.chainA.SenderAccount.GetAddress().String(),
				expPayeeAddr.String(),
				s.path.EndpointA.ChannelID,
			)

			req = &types.QueryPayeeRequest{
				ChannelId: s.path.EndpointA.ChannelID,
				Relayer:   s.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.GetSimApp().IBCFeeKeeper.Payee(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(expPayeeAddr.String(), res.PayeeAddress)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryCounterpartyPayee() {
	var req *types.QueryCounterpartyPayeeRequest

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
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"counterparty address not found: invalid channel",
			func() {
				req.ChannelId = "invalid-channel-id"
			},
			false,
		},
		{
			"counterparty address not found: invalid address",
			func() {
				req.Relayer = "invalid-addr"
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			pk := secp256k1.GenPrivKey().PubKey()
			expCounterpartyPayeeAddr := sdk.AccAddress(pk.Address())

			s.chainA.GetSimApp().IBCFeeKeeper.SetCounterpartyPayeeAddress(
				s.chainA.GetContext(),
				s.chainA.SenderAccount.GetAddress().String(),
				expCounterpartyPayeeAddr.String(),
				s.path.EndpointA.ChannelID,
			)

			req = &types.QueryCounterpartyPayeeRequest{
				ChannelId: s.path.EndpointA.ChannelID,
				Relayer:   s.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.GetSimApp().IBCFeeKeeper.CounterpartyPayee(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(expCounterpartyPayeeAddr.String(), res.CounterpartyPayee)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryFeeEnabledChannels() {
	var (
		req                   *types.QueryFeeEnabledChannelsRequest
		expFeeEnabledChannels []types.FeeEnabledChannel
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
			"empty request",
			func() {
				req = nil
			},
			false,
		},
		{
			"success: empty pagination",
			func() {
				req = &types.QueryFeeEnabledChannelsRequest{}
			},
			true,
		},
		{
			"success: with multiple fee enabled channels",
			func() {
				s.coordinator.Setup(s.pathAToC)

				expChannel := types.FeeEnabledChannel{
					PortId:    s.pathAToC.EndpointA.ChannelConfig.PortID,
					ChannelId: s.pathAToC.EndpointA.ChannelID,
				}

				expFeeEnabledChannels = append(expFeeEnabledChannels, expChannel)
			},
			true,
		},
		{
			"success: pagination with multiple fee enabled channels",
			func() {
				// start at index 1, as channel-0 is already added to expFeeEnabledChannels below
				for i := 1; i < 10; i++ {
					channelID := channeltypes.FormatChannelIdentifier(uint64(i))
					s.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainA.GetContext(), ibctesting.MockFeePort, channelID)

					expChannel := types.FeeEnabledChannel{
						PortId:    ibctesting.MockFeePort,
						ChannelId: channelID,
					}

					if i < 5 { // add only the first 5 channels, as our default pagination limit is 5
						expFeeEnabledChannels = append(expFeeEnabledChannels, expChannel)
					}
				}

				s.chainA.NextBlock()
			},
			true,
		},
		{
			"empty response",
			func() {
				s.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
				expFeeEnabledChannels = nil

				s.chainA.NextBlock()
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			s.coordinator.Setup(s.path)

			expChannel := types.FeeEnabledChannel{
				PortId:    s.path.EndpointA.ChannelConfig.PortID,
				ChannelId: s.path.EndpointA.ChannelID,
			}

			expFeeEnabledChannels = []types.FeeEnabledChannel{expChannel}

			req = &types.QueryFeeEnabledChannelsRequest{
				Pagination: &query.PageRequest{
					Limit:      5,
					CountTotal: false,
				},
				QueryHeight: 0,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.GetSimApp().IBCFeeKeeper.FeeEnabledChannels(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(expFeeEnabledChannels, res.FeeEnabledChannels)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryFeeEnabledChannel() {
	var (
		req        *types.QueryFeeEnabledChannelRequest
		expEnabled bool
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
			"empty request",
			func() {
				req = nil
				expEnabled = false
			},
			false,
		},
		{
			"fee not enabled on channel",
			func() {
				req.ChannelId = "invalid-channel-id"
				req.PortId = "invalid-port-id"
				expEnabled = false
			},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			expEnabled = true

			s.coordinator.Setup(s.path)

			req = &types.QueryFeeEnabledChannelRequest{
				PortId:    s.path.EndpointA.ChannelConfig.PortID,
				ChannelId: s.path.EndpointA.ChannelID,
			}

			tc.malleate()

			ctx := sdk.WrapSDKContext(s.chainA.GetContext())
			res, err := s.chainA.GetSimApp().IBCFeeKeeper.FeeEnabledChannel(ctx, req)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().Equal(expEnabled, res.FeeEnabled)
			} else {
				s.Require().Error(err)
			}
		})
	}
}
