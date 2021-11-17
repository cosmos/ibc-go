package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func (suite *KeeperTestSuite) TestQueryIncentivizedPacket() {

	var (
		req *types.QueryIncentivizedPacketRequest
	)

	// setup
	validPacketId := types.NewPacketId(ibctesting.FirstChannelID, 1)
	invalidPacketId := types.NewPacketId(ibctesting.FirstChannelID, 2)
	identifiedPacketFee := types.NewIdentifiedPacketFee(
		validPacketId,
		types.Fee{
			AckFee:     sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
			ReceiveFee: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
			TimeoutFee: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
		},
		suite.chainA.SenderAccount.GetAddress().String(),
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

			tc.malleate()
			ctx := sdk.WrapSDKContext(suite.chainA.GetContext())
			suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), refundAcc, identifiedPacketFee)
			res, err := suite.queryClient.IncentivizedPacket(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(identifiedPacketFee, res.IncentivizedPacket)
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
		ReceiveFee: sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}},
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

				fee1 := types.NewIdentifiedPacketFee(types.NewPacketId(ibctesting.FirstChannelID, 1), fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))
				fee2 := types.NewIdentifiedPacketFee(types.NewPacketId(ibctesting.FirstChannelID, 2), fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))
				fee3 := types.NewIdentifiedPacketFee(types.NewPacketId(ibctesting.FirstChannelID, 3), fee, suite.chainA.SenderAccount.GetAddress().String(), []string(nil))

				expPackets = []*types.IdentifiedPacketFee{}
				expPackets = append(expPackets, fee1, fee2, fee3)

				for _, p := range expPackets {
					suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), refundAcc, p)
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
				fmt.Println(expPackets)
				suite.Require().Equal(expPackets, res.IncentivizedPackets)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
