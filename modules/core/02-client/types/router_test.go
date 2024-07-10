package types_test

import (
	"errors"
	"fmt"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
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
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			cdc := suite.chainA.App.AppCodec()

			storeKey := storetypes.NewKVStoreKey("store-key")
			tmLightClientModule := ibctm.NewLightClientModule(cdc, authtypes.NewModuleAddress(govtypes.ModuleName).String())
			router = types.NewRouter(storeKey)

			tc.malleate()

			expPass := tc.expError == nil
			if expPass {
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
	var (
		err        error
		clientType string
		clientID   string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				clientID = fmt.Sprintf("%s-%d", exported.Tendermint, 0)
				clientType, _, err = types.ParseClientIdentifier(clientID)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"failure: route does not exist",
			func() {
				clientID = "06-solomachine-0"
				clientType, _, err = types.ParseClientIdentifier(clientID)
				suite.Require().NoError(err)
			},
			false,
		},
		{
			"failure: invalid client ID",
			func() {
				clientID = "invalid-client-id"
				clientType = "invalid-client-type"
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			cdc := suite.chainA.App.AppCodec()

			storeKey := storetypes.NewKVStoreKey("store-key")
			tmLightClientModule := ibctm.NewLightClientModule(cdc, authtypes.NewModuleAddress(govtypes.ModuleName).String())
			router := types.NewRouter(storeKey)
			router.AddRoute(exported.Tendermint, &tmLightClientModule)

			tc.malleate()

			hasRoute := router.HasRoute(clientType)
			route, ok := router.GetRoute(clientID)

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
