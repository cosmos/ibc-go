package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
)

func (suite *KeeperTestSuite) TestQueryIncentivizedPacket() {

	var (
		req *types.QueryIncentivizedPacketRequest
	)

	// setup
	validPacketId := suite.NewPacketId(uint64(1))
	invalidPacketId := suite.NewPacketId(uint64(2))
	coins := sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	ackFee := coins
	receiveFee := coins
	timeoutFee := coins
	fee := &types.Fee{
		AckFee:     ackFee,
		ReceiveFee: receiveFee,
		TimeoutFee: timeoutFee,
	}

	identifiedPacketFee := suite.NewIdentifiedPacketFee(validPacketId, *fee)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				req = &types.QueryIncentivizedPacketRequest{
					PacketId:    &validPacketId,
					QueryHeight: 0,
				}
			},
			true,
		},
		{
			"packetId not found",
			func() {
				req = &types.QueryIncentivizedPacketRequest{
					PacketId:    &invalidPacketId,
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
			suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), refundAcc, &identifiedPacketFee)
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

	coins := sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	ackFee := coins
	receiveFee := coins
	timeoutFee := coins
	fee := &types.Fee{
		AckFee:     ackFee,
		ReceiveFee: receiveFee,
		TimeoutFee: timeoutFee,
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

				id1 := suite.NewPacketId(uint64(1))
				id2 := suite.NewPacketId(uint64(2))
				id3 := suite.NewPacketId(uint64(3))
				fee1 := suite.NewIdentifiedPacketFee(id1, *fee)
				fee2 := suite.NewIdentifiedPacketFee(id2, *fee)
				fee3 := suite.NewIdentifiedPacketFee(id3, *fee)

				expPackets = []*types.IdentifiedPacketFee{}
				expPackets = append(expPackets, &fee1)
				expPackets = append(expPackets, &fee2)
				expPackets = append(expPackets, &fee3)

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
				suite.Require().Equal(expPackets, res.IncentivizedPackets)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
