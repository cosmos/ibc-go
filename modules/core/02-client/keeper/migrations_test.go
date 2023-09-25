package keeper_test

import (
	"reflect"

	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/codec"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// TestMigrateParams tests the migration for the client params
func (suite *KeeperTestSuite) TestMigrateParams() {
	testCases := []struct {
		name           string
		malleate       func()
		expectedParams types.Params
	}{
		{
			"success: default params",
			func() {
				params := types.DefaultParams()
				sk := suite.chainA.GetSimApp().GetKey(paramtypes.StoreKey)
				store := suite.chainA.GetContext().KVStore(sk)
				clientStore := prefix.NewStore(store, append([]byte(ibcexported.ModuleName), '/'))
				v := reflect.Indirect(reflect.ValueOf(params.AllowedClients)).Interface()
				aminoCodec := codec.NewLegacyAmino()
				bz, err := aminoCodec.MarshalJSON(v)
				suite.Require().NoError(err)

				clientStore.Set(types.KeyAllowedClients, bz)
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
			migrator := keeper.NewMigrator(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			err := migrator.MigrateParams(ctx)
			suite.Require().NoError(err)

			params := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetParams(ctx)
			suite.Require().Equal(tc.expectedParams, params)
		})
	}
}
