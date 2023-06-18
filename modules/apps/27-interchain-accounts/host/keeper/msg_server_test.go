package keeper_test

import (
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
)

func (s *KeeperTestSuite) TestUpdateParams() {
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

		s.Run(tc.name, func() {
			s.SetupTest()

			ICAHostKeeper := &s.chainA.GetSimApp().ICAHostKeeper
			tc.malleate(ICAHostKeeper.GetAuthority()) // malleate mutates test data

			ctx := s.chainA.GetContext()
			msgServer := keeper.NewMsgServerImpl(ICAHostKeeper)
			res, err := msgServer.UpdateParams(ctx, &msg)

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
