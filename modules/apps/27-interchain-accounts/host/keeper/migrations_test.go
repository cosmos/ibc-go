package keeper_test

import (
	"fmt"
	"reflect"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/codec"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
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
				sk := suite.chainA.GetSimApp().GetKey(paramtypes.StoreKey)
				store := suite.chainA.GetContext().KVStore(sk)
				hostStore := prefix.NewStore(store, append([]byte(icahosttypes.SubModuleName), '/'))
				enabled := reflect.Indirect(reflect.ValueOf(params.HostEnabled)).Interface()
				aminoCodec := codec.NewLegacyAmino()
				enabledBz, err := aminoCodec.MarshalJSON(enabled)
				suite.Require().NoError(err)

				hostStore.Set(icahosttypes.KeyHostEnabled, enabledBz)

				allowList := reflect.Indirect(reflect.ValueOf(params.AllowMessages)).Interface()
				allowListBz, err := aminoCodec.MarshalJSON(allowList)
				suite.Require().NoError(err)

				hostStore.Set(icahosttypes.KeyAllowMessages, allowListBz)

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
