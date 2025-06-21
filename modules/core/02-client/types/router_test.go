package types_test

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

func (s *TypesTestSuite) TestAddRoute() {
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
		s.Run(tc.name, func() {
			s.SetupTest()
			cdc := s.chainA.App.AppCodec()

			storeProvider := types.NewStoreProvider(runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(exported.StoreKey)))
			tmLightClientModule := ibctm.NewLightClientModule(cdc, storeProvider)
			router = types.NewRouter()

			tc.malleate()

			if tc.expError == nil {
				router.AddRoute(clientType, &tmLightClientModule)
				s.Require().True(router.HasRoute(clientType))
			} else {
				s.Require().Panics(func() {
					router.AddRoute(clientType, &tmLightClientModule)
				}, tc.expError.Error())
			}
		})
	}
}

func (s *TypesTestSuite) TestHasGetRoute() {
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
		s.Run(tc.name, func() {
			s.SetupTest()
			cdc := s.chainA.App.AppCodec()

			storeProvider := types.NewStoreProvider(runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(exported.StoreKey)))
			tmLightClientModule := ibctm.NewLightClientModule(cdc, storeProvider)
			router := types.NewRouter()
			router.AddRoute(exported.Tendermint, &tmLightClientModule)

			tc.malleate()

			hasRoute := router.HasRoute(clientType)
			route, ok := router.GetRoute(clientType)

			if tc.expPass {
				s.Require().True(hasRoute)
				s.Require().True(ok)
				s.Require().NotNil(route)
				s.Require().IsType(&ibctm.LightClientModule{}, route)
			} else {
				s.Require().False(hasRoute)
				s.Require().False(ok)
				s.Require().Nil(route)
			}
		})
	}
}
