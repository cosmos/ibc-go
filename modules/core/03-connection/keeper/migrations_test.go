package keeper_test

import (
	"reflect"

	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/codec"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/cosmos/ibc-go/v8/modules/core/03-connection/keeper"
	"github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// TestMigrateParams tests that the params for the connection are properly migrated
func (suite *KeeperTestSuite) TestMigrateParams() {
	testCases := []struct {
		name           string
		malleate       func()
		expectedParams types.Params
	}{
		{
			"success: default params",
			func() {
				// set old params via direct store manipulation in order to make sure that initializing a keytable works correctly in the migration handler
				params := types.DefaultParams()
				sk := suite.chainA.GetSimApp().GetKey(paramtypes.StoreKey)
				store := suite.chainA.GetContext().KVStore(sk)
				connectionStore := prefix.NewStore(store, append([]byte(ibcexported.ModuleName), '/'))
				v := reflect.Indirect(reflect.ValueOf(params.MaxExpectedTimePerBlock)).Interface()
				aminoCodec := codec.NewLegacyAmino()
				bz, err := aminoCodec.MarshalJSON(v)
				suite.Require().NoError(err)

				connectionStore.Set(types.KeyMaxExpectedTimePerBlock, bz)
			},
			types.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()

			ctx := suite.chainA.GetContext()
			migrator := keeper.NewMigrator(suite.chainA.GetSimApp().IBCKeeper.ConnectionKeeper)
			err := migrator.MigrateParams(ctx)
			suite.Require().NoError(err)

			params := suite.chainA.GetSimApp().IBCKeeper.ConnectionKeeper.GetParams(ctx)
			suite.Require().Equal(tc.expectedParams, params)
		})
	}
}
