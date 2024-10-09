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

func (suite *KeeperTestSuite) TestQueryClient() {
	var (
		req             *types.QueryClientRequest
		expCreator      string
		expCounterparty types.Counterparty
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
				suite.chainA.App.GetIBCKeeper().PacketServerKeeper.SetCreator(ctx, ibctesting.FirstClientID, expCreator)
				suite.chainA.App.GetIBCKeeper().PacketServerKeeper.SetCounterparty(ctx, ibctesting.FirstClientID, expCounterparty)

				req = &types.QueryClientRequest{
					ClientId: ibctesting.FirstClientID,
				}
			},
			nil,
		},
		{
			"success: no creator",
			func() {
				expCreator = ""

				suite.chainA.App.GetIBCKeeper().PacketServerKeeper.SetCounterparty(suite.chainA.GetContext(), ibctesting.FirstClientID, expCounterparty)

				req = &types.QueryClientRequest{
					ClientId: ibctesting.FirstClientID,
				}
			},
			nil,
		},
		{
			"success: no counterparty",
			func() {
				expCounterparty = types.Counterparty{}

				suite.chainA.App.GetIBCKeeper().PacketServerKeeper.SetCreator(suite.chainA.GetContext(), ibctesting.FirstClientID, expCreator)

				req = &types.QueryClientRequest{
					ClientId: ibctesting.FirstClientID,
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
				req = &types.QueryClientRequest{
					ClientId: ibctesting.FirstClientID,
				}
			},
			status.Error(codes.NotFound, fmt.Sprintf("client-id: %s: counterparty not found", ibctesting.FirstClientID)),
		},
		{
			"invalid clientID",
			func() {
				req = &types.QueryClientRequest{}
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
			expCounterparty = types.Counterparty{ClientId: ibctesting.SecondClientID, MerklePathPrefix: merklePathPrefix}

			tc.malleate()

			queryServer := keeper.NewQueryServer(suite.chainA.GetSimApp().IBCKeeper.PacketServerKeeper)
			res, err := queryServer.Client(suite.chainA.GetContext(), req)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expCreator, res.Creator)
				suite.Require().Equal(expCounterparty, res.Counterparty)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Nil(res)
			}
		})
	}
}
