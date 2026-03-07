package keeper_test

import (
	"github.com/cosmos/ibc-go/v11/modules/apps/27-interchain-accounts/host/types"
)

func (s *KeeperTestSuite) TestQueryParams() {
	ctx := s.chainA.GetContext()
	expParams := types.DefaultParams()
	res, _ := s.chainA.GetSimApp().ICAHostKeeper.Params(ctx, &types.QueryParamsRequest{})
	s.Require().Equal(&expParams, res.Params)
}
