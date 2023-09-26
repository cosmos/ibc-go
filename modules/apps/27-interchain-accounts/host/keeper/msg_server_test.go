package keeper_test

import (
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
)

func (suite *KeeperTestSuite) TestUpdateParams() {
	testCases := []struct {
		name    string
		msg     *types.MsgUpdateParams
		expPass bool
	}{
		{
			"success",
			types.NewMsgUpdateParams(suite.chainA.GetSimApp().ICAHostKeeper.GetAuthority(), types.DefaultParams()),
			true,
		},
		{
			"invalid signer address",
			types.NewMsgUpdateParams("signer", types.DefaultParams()),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()

			ctx := suite.chainA.GetContext()
			msgServer := keeper.NewMsgServerImpl(&suite.chainA.GetSimApp().ICAHostKeeper)
			res, err := msgServer.UpdateParams(ctx, tc.msg)

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
