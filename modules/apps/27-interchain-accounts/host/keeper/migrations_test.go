package keeper_test

import (
	"fmt"

	"cosmossdk.io/log"
	govtypes "cosmossdk.io/x/gov/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	icahostkeeper "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
)

func (suite *KeeperTestSuite) TestMigratorMigrateParams() {
	testCases := []struct {
		msg            string
		malleate       func()
		expectedParams icahosttypes.Params
	}{
		{
			"success: default params",
			func() {
				params := icahosttypes.DefaultParams()
				subspace := suite.chainA.GetSimApp().GetSubspace(icahosttypes.SubModuleName) // get subspace
				subspace.SetParamSet(suite.chainA.GetContext(), &params)                     // set params
			},
			icahosttypes.DefaultParams(),
		},
		{
			"success: no legacy params pre-migration",
			func() {
				suite.chainA.GetSimApp().ICAHostKeeper = icahostkeeper.NewKeeper(
					suite.chainA.Codec,
					runtime.NewEnvironment(runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(icahosttypes.StoreKey)), log.NewNopLogger()),
					nil, // assign a nil legacy param subspace
					suite.chainA.GetSimApp().IBCFeeKeeper,
					suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
					suite.chainA.GetSimApp().AuthKeeper,
					authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				)
			},
			icahosttypes.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate() // explicitly set params

			migrator := icahostkeeper.NewMigrator(&suite.chainA.GetSimApp().ICAHostKeeper)
			err := migrator.MigrateParams(suite.chainA.GetContext())
			suite.Require().NoError(err)

			params := suite.chainA.GetSimApp().ICAHostKeeper.GetParams(suite.chainA.GetContext())
			suite.Require().Equal(tc.expectedParams, params)
		})
	}
}
