package keeper_test

import (
	"fmt"

	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	icahostkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
)

func (suite *KeeperTestSuite) TestMigratorMigrateParams() {
	testCases := []struct {
		msg            string
		malleate       func(subspace paramstypes.Subspace)
		expectedParams icahosttypes.Params
	}{
		{
			"success: default params",
			func(subspace paramstypes.Subspace) {
				params := icahosttypes.DefaultParams()
				subspace.SetParamSet(suite.chainA.GetContext(), &params) // set params
			},
			icahosttypes.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.msg), func() {
			suite.SetupTest() // reset

			subspace := suite.chainA.GetSimApp().GetSubspace(icahosttypes.SubModuleName) // get subspace
			tc.malleate(subspace)                                                        // explicitly set params

			migrator := icahostkeeper.NewMigrator(&suite.chainA.GetSimApp().ICAHostKeeper, subspace)
			err := migrator.MigrateParams(suite.chainA.GetContext())
			suite.Require().NoError(err)

			params := suite.chainA.GetSimApp().ICAHostKeeper.GetParams(suite.chainA.GetContext())
			suite.Require().Equal(tc.expectedParams, params)
		})
	}
}