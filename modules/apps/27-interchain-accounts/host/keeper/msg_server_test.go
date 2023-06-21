package keeper_test

import (
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
)

func (s *KeeperTestSuite) TestUpdateParams() {
	testCases := []struct {
		name    string
		msg     *types.MsgUpdateParams
		expPass bool
	}{
		{
			"success",
			types.NewMsgUpdateParams(s.chainA.GetSimApp().ICAHostKeeper.GetAuthority(), types.DefaultParams()),
			true,
		},
		{
			"invalid authority address",
			types.NewMsgUpdateParams("authority", types.DefaultParams()),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()

			ctx := s.chainA.GetContext()
			msgServer := keeper.NewMsgServerImpl(&s.chainA.GetSimApp().ICAHostKeeper)
			res, err := msgServer.UpdateParams(ctx, tc.msg)

			if tc.expPass {
				s.Require().NoError(err)
				s.Require().NotNil(res)
			} else {
				s.Require().Error(err)
				s.Require().Nil(res)
			}
		})
	}
}
