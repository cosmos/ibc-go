package types_test

import (
	"errors"
	"fmt"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

func (suite *TypesTestSuite) TestAddRoute() {
	var (
		clientType string
		router     *types.Router
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				clientType = exported.Tendermint
			},
			nil,
		},
		{
			"failure: route has already been imported",
			func() {
				clientType = exported.Tendermint
				router.AddRoute(exported.Tendermint, &ibctm.LightClientModule{})
			},
			fmt.Errorf("route %s has already been registered", exported.Tendermint),
		},
		{
			"failure: client type is invalid",
			func() {
				clientType = ""
			},
			errors.New("failed to add route: client type cannot be blank"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			cdc := suite.chainA.App.AppCodec()

			storeProvider := types.NewStoreProvider(runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(exported.StoreKey)))
			tmLightClientModule := ibctm.NewLightClientModule(cdc, storeProvider)
			router = types.NewRouter()

			tc.malleate()

			if tc.expError == nil {
				router.AddRoute(clientType, &tmLightClientModule)
				suite.Require().True(router.HasRoute(clientType))
			} else {
				require.Panics(suite.T(), func() {
					router.AddRoute(clientType, &tmLightClientModule)
				}, tc.expError.Error())
			}
		})
	}
}

func (suite *TypesTestSuite) TestHasGetRoute() {
	var clientType string

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				clientType = exported.Tendermint
			},
			true,
		},
		{
			"failure: route does not exist",
			func() {
				clientType = exported.Solomachine
			},
			false,
		},
		{
			"failure: invalid client ID",
			func() {
				clientType = "invalid-client-type"
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			cdc := suite.chainA.App.AppCodec()

			storeProvider := types.NewStoreProvider(runtime.NewKVStoreService(suite.chainA.GetSimApp().GetKey(exported.StoreKey)))
			tmLightClientModule := ibctm.NewLightClientModule(cdc, storeProvider)
			router := types.NewRouter()
			router.AddRoute(exported.Tendermint, &tmLightClientModule)

			tc.malleate()

			hasRoute := router.HasRoute(clientType)
			route, ok := router.GetRoute(clientType)

			if tc.expPass {
				suite.Require().True(hasRoute)
				suite.Require().True(ok)
				suite.Require().NotNil(route)
				suite.Require().IsType(&ibctm.LightClientModule{}, route)
			} else {
				suite.Require().False(hasRoute)
				suite.Require().False(ok)
				suite.Require().Nil(route)
			}
		})
	}
}
