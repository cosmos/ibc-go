package keeper_test

import (
	"fmt"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/runtime"

	icacontrollerkeeper "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
)

func (suite *KeeperTestSuite) TestMigratorMigrateParams() {
	testCases := []struct {
		msg            string
		malleate       func()
		expectedParams icacontrollertypes.Params
	}{
		{
			"success: default params",
			func() {
				params := icacontrollertypes.DefaultParams()
				subspace := suite.chainA.GetSimApp().GetSubspace(icacontrollertypes.SubModuleName) // get subspace
				subspace.SetParamSet(suite.chainA.GetContext(), &params)                           // set params
			},
			icacontrollertypes.DefaultParams(),
		},
		{
			"success: no legacy params pre-migration",
			func() {
				suite.chainA.GetSimApp().ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
					suite.chainA.Codec,
					runtime.NewEnvironment(runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(icacontrollertypes.StoreKey)), log.NewNopLogger()),
					nil, // assign a nil legacy param subspace
					suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
					suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
					suite.chainA.GetSimApp().ICAControllerKeeper.GetAuthority(),
				)
			},
			icacontrollertypes.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate() // explicitly set params

			migrator := icacontrollerkeeper.NewMigrator(&suite.chainA.GetSimApp().ICAControllerKeeper)
			err := migrator.MigrateParams(suite.chainA.GetContext())
			suite.Require().NoError(err)

			params := suite.chainA.GetSimApp().ICAControllerKeeper.GetParams(suite.chainA.GetContext())
			suite.Require().Equal(tc.expectedParams, params)
		})
	}
}
