package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *KeeperTestSuite) TestQueryIncentivizedPackets() {
	var (
		req             *types.QueryIncentivizedPacketsRequest
		expectedPackets []types.IdentifiedPacketFees
	)

	testCases := []struct {
		name     string
		malleate func()
		errMsg   string
	}{
		{
			"success",
			func() {
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

				fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
				packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))

				for i := 0; i < 3; i++ {
					// escrow packet fees for three different packets
					packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, uint64(i+1))
					suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))

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
			"",
		},
		{
			"empty pagination",
			func() {
				expectedPackets = nil
				req = &types.QueryIncentivizedPacketsRequest{}
			},
			"",
		},
		{
			"empty request",
			func() {
				req = nil
			},
			"InvalidArgument",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate() // malleate mutates test data

			ctx := suite.chainA.GetContext()

			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.IncentivizedPackets(ctx, req)

			if tc.errMsg == "" {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expectedPackets, res.IncentivizedPackets)
			} else {
				suite.Require().ErrorContains(err, tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryIncentivizedPacket() {
	var req *types.QueryIncentivizedPacketRequest

	testCases := []struct {
		name     string
		malleate func()
		errMsg   string
	}{
		{
			"success",
			func() {},
			"",
		},
		{
			"empty request",
			func() {
				req = nil
			},
			"InvalidArgument",
		},
		{
			"fees not found for packet id",
			func() {
				req = &types.QueryIncentivizedPacketRequest{
					PacketId:    channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100),
					QueryHeight: 0,
				}
			},
			"NotFound",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)
			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryIncentivizedPacketRequest{
				PacketId:    packetID,
				QueryHeight: 0,
			}

			tc.malleate() // malleate mutates test data

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.IncentivizedPacket(ctx, req)

			if tc.errMsg == "" {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(types.NewIdentifiedPacketFees(packetID, []types.PacketFee{packetFee, packetFee, packetFee}), res.IncentivizedPacket)
			} else {
				suite.Require().ErrorContains(err, tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryIncentivizedPacketsForChannel() {
	var (
		req                     *types.QueryIncentivizedPacketsForChannelRequest
		expIdentifiedPacketFees []*types.IdentifiedPacketFees
		packetFees              types.PacketFees
		path                    *ibctesting.Path
	)

	fee := types.Fee{
		AckFee:     sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(100)}},
		RecvFee:    sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(100)}},
		TimeoutFee: sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(100)}},
	}

	testCases := []struct {
		msg      string
		malleate func()
		errMsg   string
	}{
		{
			"empty pagination",
			func() {
				path := ibctesting.NewTransferPathWithFeeEnabled(suite.chainA, suite.chainB)
				path.Setup()
				expIdentifiedPacketFees = nil
				req = &types.QueryIncentivizedPacketsForChannelRequest{
					Pagination:  &query.PageRequest{},
					PortId:      path.EndpointA.ChannelConfig.PortID,
					ChannelId:   path.EndpointA.ChannelID,
					QueryHeight: 0,
				}
			},
			"",
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
					ChannelId:   path.EndpointA.ChannelID,
					QueryHeight: 0,
				}

				expIdentifiedPacketFees = []*types.IdentifiedPacketFees{}
				for i := 0; i < 3; i++ {
					packetID := channeltypes.NewPacketID(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, uint64(i))
					identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)
					expIdentifiedPacketFees = append(expIdentifiedPacketFees, &identifiedPacketFees)
					suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), identifiedPacketFees.PacketId, types.NewPacketFees(identifiedPacketFees.PacketFees))
				}
			},
			"",
		},
		{
			"no packets for specified channel",
			func() {
				path := ibctesting.NewTransferPathWithFeeEnabled(suite.chainA, suite.chainB)
				path.Setup()
				expIdentifiedPacketFees = nil
				req = &types.QueryIncentivizedPacketsForChannelRequest{
					Pagination: &query.PageRequest{
						Limit:      5,
						CountTotal: false,
					},
					PortId:      path.EndpointA.ChannelConfig.PortID,
					ChannelId:   path.EndpointA.ChannelID,
					QueryHeight: 0,
				}
			},
			"",
		},
		{
			"empty request",
			func() {
				req = nil
			},
			"InvalidArgument",
		},
		{
			"channel not found",
			func() {
				req = &types.QueryIncentivizedPacketsForChannelRequest{
					PortId:    ibctesting.MockFeePort,
					ChannelId: ibctesting.InvalidID,
				}
			},
			"NotFound",
		},
		{
			"invalid ID",
			func() {
				req = &types.QueryIncentivizedPacketsForChannelRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			"InvalidArgument",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			// setup
			refundAcc := suite.chainA.SenderAccount.GetAddress()
			packetFee := types.NewPacketFee(fee, refundAcc.String(), nil)
			packetFees = types.NewPacketFees([]types.PacketFee{packetFee, packetFee, packetFee})

			identifiedFees1 := types.NewIdentifiedPacketFees(channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1), packetFees.PacketFees)
			identifiedFees2 := types.NewIdentifiedPacketFees(channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 2), packetFees.PacketFees)
			identifiedFees3 := types.NewIdentifiedPacketFees(channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 3), packetFees.PacketFees)

			expIdentifiedPacketFees = append(expIdentifiedPacketFees, &identifiedFees1, &identifiedFees2, &identifiedFees3)

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)
			for _, identifiedPacketFees := range expIdentifiedPacketFees {
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), identifiedPacketFees.PacketId, types.NewPacketFees(identifiedPacketFees.PacketFees))
			}

			path = ibctesting.NewTransferPathWithFeeEnabled(suite.chainA, suite.chainB)
			path.Setup()

			tc.malleate()
			ctx := suite.chainA.GetContext()

			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.IncentivizedPacketsForChannel(ctx, req)

			if tc.errMsg == "" {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expIdentifiedPacketFees, res.IncentivizedPackets)
			} else {
				suite.Require().ErrorContains(err, tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryTotalRecvFees() {
	var req *types.QueryTotalRecvFeesRequest

	testCases := []struct {
		name     string
		malleate func()
		errMsg   string
	}{
		{
			"success",
			func() {},
			"",
		},
		{
			"empty request",
			func() {
				req = nil
			},
			"InvalidArgument",
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100)
			},
			"NotFound",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)

			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryTotalRecvFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.TotalRecvFees(ctx, req)

			if tc.errMsg == "" {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				// expected total is three times the default recv fee
				expectedFees := defaultRecvFee.Add(defaultRecvFee...).Add(defaultRecvFee...)
				suite.Require().Equal(expectedFees, res.RecvFees)
			} else {
				suite.Require().ErrorContains(err, tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryTotalAckFees() {
	var req *types.QueryTotalAckFeesRequest

	testCases := []struct {
		name     string
		malleate func()
		errMsg   string
	}{
		{
			"success",
			func() {},
			"",
		},
		{
			"empty request",
			func() {
				req = nil
			},
			"InvalidArgument",
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100)
			},
			"NotFound",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)

			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryTotalAckFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.TotalAckFees(ctx, req)

			if tc.errMsg == "" {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				// expected total is three times the default acknowledgement fee
				expectedFees := defaultAckFee.Add(defaultAckFee...).Add(defaultAckFee...)
				suite.Require().Equal(expectedFees, res.AckFees)
			} else {
				suite.Require().ErrorContains(err, tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryTotalTimeoutFees() {
	var req *types.QueryTotalTimeoutFeesRequest

	testCases := []struct {
		name     string
		malleate func()
		errMsg   string
	}{
		{
			"success",
			func() {},
			"",
		},
		{
			"empty request",
			func() {
				req = nil
			},
			"InvalidArgument",
		},
		{
			"packet not found",
			func() {
				req.PacketId = channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 100)
			},
			"NotFound",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)

			packetID := channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)

			fee := types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee)
			packetFee := types.NewPacketFee(fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))

			packetFees := []types.PacketFee{packetFee, packetFee, packetFee}
			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees(packetFees))

			req = &types.QueryTotalTimeoutFeesRequest{
				PacketId: packetID,
			}

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.TotalTimeoutFees(ctx, req)

			if tc.errMsg == "" {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				// expected total is three times the default acknowledgement fee
				expectedFees := defaultTimeoutFee.Add(defaultTimeoutFee...).Add(defaultTimeoutFee...)
				suite.Require().Equal(expectedFees, res.TimeoutFees)
			} else {
				suite.Require().ErrorContains(err, tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryPayee() {
	var req *types.QueryPayeeRequest

	testCases := []struct {
		name     string
		malleate func()
		errMsg   string
	}{
		{
			"success",
			func() {},
			"",
		},
		{
			"empty request",
			func() {
				req = nil
			},
			"InvalidArgument",
		},
		{
			"payee address not found: invalid channel",
			func() {
				req.ChannelId = "invalid-channel-id" //nolint:goconst
			},
			"NotFound",
		},
		{
			"payee address not found: invalid relayer address",
			func() {
				req.Relayer = "invalid-addr" //nolint:goconst
			},
			"NotFound",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			pk := secp256k1.GenPrivKey().PubKey()
			expPayeeAddr := sdk.AccAddress(pk.Address())

			suite.chainA.GetSimApp().IBCFeeKeeper.SetPayeeAddress(
				suite.chainA.GetContext(),
				suite.chainA.SenderAccount.GetAddress().String(),
				expPayeeAddr.String(),
				suite.path.EndpointA.ChannelID,
			)

			req = &types.QueryPayeeRequest{
				ChannelId: suite.path.EndpointA.ChannelID,
				Relayer:   suite.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.Payee(ctx, req)

			if tc.errMsg == "" {
				suite.Require().NoError(err)
				suite.Require().Equal(expPayeeAddr.String(), res.PayeeAddress)
			} else {
				suite.Require().ErrorContains(err, tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryCounterpartyPayee() {
	var req *types.QueryCounterpartyPayeeRequest

	testCases := []struct {
		name     string
		malleate func()
		errMsg   string
	}{
		{
			"success",
			func() {},
			"",
		},
		{
			"empty request",
			func() {
				req = nil
			},
			"InvalidArgument",
		},
		{
			"counterparty address not found: invalid channel",
			func() {
				req.ChannelId = "invalid-channel-id"
			},
			"NotFound",
		},
		{
			"counterparty address not found: invalid address",
			func() {
				req.Relayer = "invalid-addr"
			},
			"NotFound",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			pk := secp256k1.GenPrivKey().PubKey()
			expCounterpartyPayeeAddr := sdk.AccAddress(pk.Address())

			suite.chainA.GetSimApp().IBCFeeKeeper.SetCounterpartyPayeeAddress(
				suite.chainA.GetContext(),
				suite.chainA.SenderAccount.GetAddress().String(),
				expCounterpartyPayeeAddr.String(),
				suite.path.EndpointA.ChannelID,
			)

			req = &types.QueryCounterpartyPayeeRequest{
				ChannelId: suite.path.EndpointA.ChannelID,
				Relayer:   suite.chainA.SenderAccount.GetAddress().String(),
			}

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.CounterpartyPayee(ctx, req)

			if tc.errMsg == "" {
				suite.Require().NoError(err)
				suite.Require().Equal(expCounterpartyPayeeAddr.String(), res.CounterpartyPayee)
			} else {
				suite.Require().ErrorContains(err, tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryFeeEnabledChannelsWithPagination() {
	suite.SetupTest() // reset

	suite.path.Setup()

	expChannel := types.FeeEnabledChannel{
		PortId:    suite.path.EndpointA.ChannelConfig.PortID,
		ChannelId: suite.path.EndpointA.ChannelID,
	}

	expFeeEnabledChannels := []types.FeeEnabledChannel{expChannel}

	req := &types.QueryFeeEnabledChannelsRequest{
		Pagination: &query.PageRequest{
			Limit:      5,
			CountTotal: false,
		},
		QueryHeight: 0,
	}

	// Extract the next available sequence number for channel IDs.
	nextSeq := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetNextChannelSequence(suite.chainA.GetContext())
	for i := 0; i < 8; i++ {
		channelID := channeltypes.FormatChannelIdentifier(uint64(i + int(nextSeq)))
		suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, channelID)

		expChannel := types.FeeEnabledChannel{
			PortId:    ibctesting.MockFeePort,
			ChannelId: channelID,
		}

		if i < 4 { // add only the first 5 channels, as our default pagination limit is 5
			expFeeEnabledChannels = append(expFeeEnabledChannels, expChannel)
		}
	}

	suite.chainA.NextBlock()

	ctx := suite.chainA.GetContext()
	res, err := suite.chainA.GetSimApp().IBCFeeKeeper.FeeEnabledChannels(ctx, req)

	suite.Require().NoError(err)
	suite.Require().Equal(expFeeEnabledChannels, res.FeeEnabledChannels)
}

func (suite *KeeperTestSuite) TestQueryFeeEnabledChannels() {
	var (
		req                   *types.QueryFeeEnabledChannelsRequest
		expFeeEnabledChannels []types.FeeEnabledChannel
	)

	testCases := []struct {
		name     string
		malleate func()
		errMsg   string
	}{
		{
			"success",
			func() {},
			"",
		},
		{
			"success: empty pagination",
			func() {
				req = &types.QueryFeeEnabledChannelsRequest{}
			},
			"",
		},
		{
			"success: with multiple fee enabled channels",
			func() {
				suite.pathAToC.Setup()

				expChannel := types.FeeEnabledChannel{
					PortId:    suite.pathAToC.EndpointA.ChannelConfig.PortID,
					ChannelId: suite.pathAToC.EndpointA.ChannelID,
				}

				expFeeEnabledChannels = append(expFeeEnabledChannels, expChannel)
			},
			"",
		},
		{
			"failure: empty request",
			func() {
				req = nil
			},
			"InvalidArgument",
		},
		{
			"empty response",
			func() {
				suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
				expFeeEnabledChannels = nil

				suite.chainA.NextBlock()
			},
			"",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.path.Setup()

			expChannel := types.FeeEnabledChannel{
				PortId:    suite.path.EndpointA.ChannelConfig.PortID,
				ChannelId: suite.path.EndpointA.ChannelID,
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

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.FeeEnabledChannels(ctx, req)

			if tc.errMsg == "" {
				suite.Require().NoError(err)
				suite.Require().Equal(expFeeEnabledChannels, res.FeeEnabledChannels)
			} else {
				suite.Require().ErrorContains(err, tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryFeeEnabledChannel() {
	var (
		req        *types.QueryFeeEnabledChannelRequest
		expEnabled bool
		path       *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		errMsg   string
	}{
		{
			"success",
			func() {},
			"",
		},
		{
			"fee not enabled on channel",
			func() {
				expEnabled = false
				path = ibctesting.NewPath(suite.chainA, suite.chainB)
				path.Setup()

				req = &types.QueryFeeEnabledChannelRequest{
					PortId:    path.EndpointA.ChannelConfig.PortID,
					ChannelId: path.EndpointA.ChannelID,
				}
			},
			"",
		},
		{
			"empty request",
			func() {
				req = nil
				expEnabled = false
			},
			"InvalidArgument",
		},
		{
			"channel not found",
			func() {
				req.ChannelId = ibctesting.InvalidID
			},
			"NotFound",
		},
		{
			"invalid ID",
			func() {
				req = &types.QueryFeeEnabledChannelRequest{
					PortId:    "",
					ChannelId: "test-channel-id",
				}
			},
			"InvalidArgument",
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			expEnabled = true

			path = ibctesting.NewPathWithFeeEnabled(suite.chainA, suite.chainB)
			path.Setup()

			req = &types.QueryFeeEnabledChannelRequest{
				PortId:    path.EndpointA.ChannelConfig.PortID,
				ChannelId: path.EndpointA.ChannelID,
			}

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.FeeEnabledChannel(ctx, req)

			if tc.errMsg == "" {
				suite.Require().NoError(err)
				suite.Require().Equal(expEnabled, res.FeeEnabled)
			} else {
				suite.Require().ErrorContains(err, tc.errMsg)
			}
		})
	}
}
