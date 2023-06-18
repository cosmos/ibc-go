package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
)

func (s *KeeperTestSuite) TestQueryParams() {
	ctx := sdk.WrapSDKContext(s.chainA.GetContext())
	expParams := types.DefaultParams()
	res, _ := s.chainA.GetSimApp().ICAHostKeeper.Params(ctx, &types.QueryParamsRequest{})
	s.Require().Equal(&expParams, res.Params)
}
