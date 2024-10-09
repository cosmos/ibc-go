package keeper_test

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *KeeperTestSuite) TestQueryChannel() {
	var (
		req        *types.QueryChannelRequest
		expCreator string
		expChannel types.Channel
	)

	testCases := []struct {
		msg      string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				ctx := suite.chainA.GetContext()
				suite.chainA.App.GetIBCKeeper().PacketServerKeeper.SetCreator(ctx, ibctesting.FirstChannelID, expCreator)
				suite.chainA.App.GetIBCKeeper().PacketServerKeeper.SetChannel(ctx, ibctesting.FirstChannelID, expChannel)

				req = &types.QueryChannelRequest{
					ChannelId: ibctesting.FirstChannelID,
				}
			},
			nil,
		},
		{
			"success: no creator",
			func() {
				expCreator = ""

				suite.chainA.App.GetIBCKeeper().PacketServerKeeper.SetChannel(suite.chainA.GetContext(), ibctesting.FirstChannelID, expChannel)

				req = &types.QueryChannelRequest{
					ChannelId: ibctesting.FirstChannelID,
				}
			},
			nil,
		},
		{
			"success: no channel",
			func() {
				expChannel = types.Channel{}

				suite.chainA.App.GetIBCKeeper().PacketServerKeeper.SetCreator(suite.chainA.GetContext(), ibctesting.FirstChannelID, expCreator)

				req = &types.QueryChannelRequest{
					ChannelId: ibctesting.FirstChannelID,
				}
			},
			nil,
		},
		{
			"req is nil",
			func() {
				req = nil
			},
			status.Error(codes.InvalidArgument, "empty request"),
		},
		{
			"no creator and no counterparty",
			func() {
				req = &types.QueryChannelRequest{
					ChannelId: ibctesting.FirstChannelID,
				}
			},
			status.Error(codes.NotFound, fmt.Sprintf("client-id: %s: counterparty not found", ibctesting.FirstChannelID)),
		},
		{
			"invalid channelID",
			func() {
				req = &types.QueryChannelRequest{}
			},
			status.Error(codes.InvalidArgument, "identifier cannot be blank: invalid identifier"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			expCreator = ibctesting.TestAccAddress
			merklePathPrefix := commitmenttypes.NewMerklePath([]byte("prefix"))
			expChannel = types.Channel{ClientId: ibctesting.SecondClientID, CounterpartyChannelId: ibctesting.SecondChannelID, MerklePathPrefix: merklePathPrefix}

			tc.malleate()

			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.PacketServerKeeper)
			res, err := queryServer.Channel(suite.chainA.GetContext(), req)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expCreator, res.Creator)
				suite.Require().Equal(expChannel, res.Channel)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Nil(res)
			}
		})
	}
}
