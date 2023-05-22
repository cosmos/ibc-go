package keeper_test

import (
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
)

func (suite *KeeperTestSuite) TestUpdateParams() {
	msg := types.MsgUpdateParams{}

	testCases := []struct {
		name     string
		malleate func(authority string)
		expPass  bool
	}{
		{
			"success",
			func(authority string) {
				msg.Authority = authority
				msg.Params = types.DefaultParams()
			},
			true,
		},
		{
			"invalid authority address",
			func(authority string) {
				msg.Authority = "authority"
				msg.Params = types.DefaultParams()
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			ICAHostKeeper := &suite.chainA.GetSimApp().ICAHostKeeper
			tc.malleate(ICAHostKeeper.GetAuthority()) // malleate mutates test data

			ctx := suite.chainA.GetContext()
			msgServer := keeper.NewMsgServerImpl(ICAHostKeeper)
			res, err := msgServer.UpdateParams(ctx, &msg)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(res)
			}
		})
	}
}
