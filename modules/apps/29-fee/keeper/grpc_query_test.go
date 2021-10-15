package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

var (
	expPackets []*types.IdentifiedPacketFee
	refundAcc  sdk.AccAddress
	ackFee     sdk.Coins
	receiveFee sdk.Coins
	timeoutFee sdk.Coins
)

func (suite *KeeperTestSuite) TestQueryIncentivizedPacket() {

	var (
		req *types.QueryIncentivizedPacketRequest
	)

	// setup
	validChannelId := "channel-0"
	validPacketId := &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(1)}
	refundAcc = suite.chainA.SenderAccount.GetAddress()
	fmt.Println(suite.chainA.SenderAccount.GetAddress())

	validCoins = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	ackFee = validCoins
	receiveFee = validCoins
	timeoutFee = validCoins
	fee := &types.Fee{
		AckFee:     ackFee,
		ReceiveFee: receiveFee,
		TimeoutFee: timeoutFee,
	}
	identifiedPacketFee := types.IdentifiedPacketFee{PacketId: validPacketId, Fee: fee, Relayers: []string{}}

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"packetId not found",
			func() {
				req = &types.QueryIncentivizedPacketRequest{
					PacketId:    validPacketId,
					QueryHeight: 0,
				}
			},
			false,
		},
		{
			"success",
			func() {
				err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), refundAcc, &identifiedPacketFee)
				fmt.Println("FRSTERRRRO", err)
				req = &types.QueryIncentivizedPacketRequest{
					PacketId:    validPacketId,
					QueryHeight: 0,
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
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
		req *types.QueryIncentivizedPacketsRequest
	)
	refundAcc = suite.chainA.SenderAccount.GetAddress()
	validChannelId := "channel-0"
	validCoins = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	ackFee = validCoins
	receiveFee = validCoins
	timeoutFee = validCoins
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
				id1 := &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(1)}
				id2 := &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(2)}
				id3 := &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(3)}
				fee1 := types.IdentifiedPacketFee{PacketId: id1, Fee: fee, Relayers: []string{}}
				fee2 := types.IdentifiedPacketFee{PacketId: id2, Fee: fee, Relayers: []string{}}
				fee3 := types.IdentifiedPacketFee{PacketId: id3, Fee: fee, Relayers: []string{}}

				expPackets = append(expPackets, &fee1)
				expPackets = append(expPackets, &fee2)
				expPackets = append(expPackets, &fee3)

				for _, p := range expPackets {
					err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), refundAcc, p)
					fmt.Println("ERRR", err)
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
