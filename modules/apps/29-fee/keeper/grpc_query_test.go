package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

func (suite *KeeperTestSuite) TestQueryIncentivizedPacket() {

	var (
		req *types.QueryIncentivizedPacketRequest
	)

	// setup
	suite.coordinator.Setup(suite.path) // setup channel
	validPacketId := channeltypes.NewPacketId(suite.path.EndpointA.ChannelID, suite.path.EndpointA.ChannelConfig.PortID, 1)
	invalidPacketId := channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 2)
	identifiedPacketFee := types.NewIdentifiedPacketFee(
		validPacketId,
		types.Fee{
			AckFee:     sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
			RecvFee:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
			TimeoutFee: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
		},
		"", // leave empty here and then populate on each testcase since suite resets
		[]string(nil),
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				req = &types.QueryIncentivizedPacketRequest{
					PacketId:    validPacketId,
					QueryHeight: 0,
				}
			},
			true,
		},
		{
			"packetId not found",
			func() {
				req = &types.QueryIncentivizedPacketRequest{
					PacketId:    invalidPacketId,
					QueryHeight: 0,
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			refundAcc := suite.chainA.SenderAccount.GetAddress()
			// populate RefundAddress field
			identifiedPacketFee.RefundAddress = refundAcc.String()

			tc.malleate()

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), validPacketId.PortId, validPacketId.ChannelId)
			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeInEscrow(suite.chainA.GetContext(), identifiedPacketFee)

			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			res, err := suite.queryClient.IncentivizedPacket(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(&identifiedPacketFee, res.IncentivizedPacket)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryIncentivizedPackets() {
	var (
		req        *types.QueryIncentivizedPacketsRequest
		expPackets []*types.IdentifiedPacketFee
	)

	fee := types.Fee{
		AckFee:     sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
		RecvFee:    sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
		TimeoutFee: sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
	}

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty pagination",
			func() {
				req = &types.QueryIncentivizedPacketsRequest{}
			},
			true,
		},
		{
			"success",
			func() {
				refundAcc := suite.chainA.SenderAccount.GetAddress()

				fee1 := types.NewIdentifiedPacketFee(channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 1), fee, refundAcc.String(), []string(nil))
				fee2 := types.NewIdentifiedPacketFee(channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 2), fee, refundAcc.String(), []string(nil))
				fee3 := types.NewIdentifiedPacketFee(channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 3), fee, refundAcc.String(), []string(nil))

				expPackets = []*types.IdentifiedPacketFee{}
				expPackets = append(expPackets, &fee1, &fee2, &fee3)

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)
				for _, packetFee := range expPackets {
					suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeInEscrow(suite.chainA.GetContext(), *packetFee)
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
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			tc.malleate()
			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())

			res, err := suite.queryClient.IncentivizedPackets(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expPackets, res.IncentivizedPackets)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryIncentivizedPacketsForChannel() {
	var (
		req                     *types.QueryIncentivizedPacketsForChannelRequest
		expIdentifiedPacketFees []*types.IdentifiedPacketFees
	)

	fee := types.Fee{
		AckFee:     sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
		RecvFee:    sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
		TimeoutFee: sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
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
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			// setup
			refundAcc := suite.chainA.SenderAccount.GetAddress()
			packetFee := types.NewPacketFee(fee, refundAcc.String(), nil)
			packetFees := types.NewPacketFees([]types.PacketFee{packetFee, packetFee, packetFee})

			identifiedFees1 := types.NewIdentifiedPacketFees(channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 1), packetFees.PacketFees)
			identifiedFees2 := types.NewIdentifiedPacketFees(channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 2), packetFees.PacketFees)
			identifiedFees3 := types.NewIdentifiedPacketFees(channeltypes.NewPacketId(ibctesting.FirstChannelID, ibctesting.MockFeePort, 3), packetFees.PacketFees)

			expIdentifiedPacketFees = append(expIdentifiedPacketFees, &identifiedFees1, &identifiedFees2, &identifiedFees3)

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), ibctesting.MockFeePort, ibctesting.FirstChannelID)
			for _, identifiedPacketFees := range expIdentifiedPacketFees {
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), identifiedPacketFees.PacketId, types.NewPacketFees(identifiedPacketFees.PacketFees))
			}

			tc.malleate()
			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())

			res, err := suite.queryClient.IncentivizedPacketsForChannel(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expIdentifiedPacketFees, res.IncentivizedPackets)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
