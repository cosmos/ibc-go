package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

func (suite *KeeperTestSuite) TestQueryIncentivizedPacket() {

	var (
		req *types.QueryIncentivizedPacketRequest
	)

	// setup
	channelId := "channel-0"
	validPacketId := &channeltypes.PacketId{ChannelId: channelId, PortId: types.PortKey, Sequence: uint64(1)}
	invalidPacketId := &channeltypes.PacketId{ChannelId: channelId, PortId: types.PortKey, Sequence: uint64(2)}
	coins := sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	ackFee := coins
	receiveFee := coins
	timeoutFee := coins
	fee := &types.Fee{
		AckFee:     ackFee,
		ReceiveFee: receiveFee,
		TimeoutFee: timeoutFee,
	}
	identifiedPacketFee := types.IdentifiedPacketFee{PacketId: validPacketId, Fee: *fee, Relayers: []string(nil)}

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
			suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), refundAcc, &identifiedPacketFee)
			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.IncentivizedPacket(ctx, req)

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
	validChannelId := "channel-0"
	coins := sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	ackFee := coins
	receiveFee := coins
	timeoutFee := coins
	fee := &types.Fee{
		AckFee:     ackFee,
		ReceiveFee: receiveFee,
		TimeoutFee: timeoutFee,
	}

	id1 := &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(1)}
	id2 := &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(2)}
	id3 := &channeltypes.PacketId{ChannelId: validChannelId, PortId: types.PortKey, Sequence: uint64(3)}
	fee1 := types.IdentifiedPacketFee{PacketId: id1, Fee: *fee, Relayers: []string(nil)}
	fee2 := types.IdentifiedPacketFee{PacketId: id2, Fee: *fee, Relayers: []string(nil)}
	fee3 := types.IdentifiedPacketFee{PacketId: id3, Fee: *fee, Relayers: []string(nil)}

	expPackets := []*types.IdentifiedPacketFee(nil)
	expPackets = append(expPackets, &fee1)
	expPackets = append(expPackets, &fee2)
	expPackets = append(expPackets, &fee3)

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

			res, err := suite.chainA.GetSimApp().IBCFeeKeeper.IncentivizedPackets(ctx, req)

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
