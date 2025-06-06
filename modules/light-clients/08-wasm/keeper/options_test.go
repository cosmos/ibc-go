package keeper_test

import (
	"encoding/json"
	"errors"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/keeper"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
)

func mockErrorCustomQuerier() func(sdk.Context, json.RawMessage) ([]byte, error) {
	return func(_ sdk.Context, _ json.RawMessage) ([]byte, error) {
		return nil, errors.New("custom querier error for TestNewKeeperWithOptions")
	}
}

func mockErrorStargateQuerier() func(sdk.Context, *wasmvmtypes.StargateQuery) ([]byte, error) {
	return func(_ sdk.Context, _ *wasmvmtypes.StargateQuery) ([]byte, error) {
		return nil, errors.New("stargate querier error for TestNewKeeperWithOptions")
	}
}

func (suite *KeeperTestSuite) TestNewKeeperWithOptions() {
	var k keeper.Keeper
	testCases := []struct {
		name     string
		malleate func()
		verifyFn func(keeper.Keeper)
	}{
		{
			"success: no options",
			func() {
				k = keeper.NewKeeperWithVM(
					GetSimApp(suite.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(suite.chainA).GetKey(types.StoreKey)),
					GetSimApp(suite.chainA).IBCKeeper.ClientKeeper,
					GetSimApp(suite.chainA).WasmClientKeeper.GetAuthority(),
					GetSimApp(suite.chainA).WasmClientKeeper.GetVM(),
					GetSimApp(suite.chainA).GRPCQueryRouter(),
				)
			},
			func(k keeper.Keeper) {
				plugins := k.GetQueryPlugins()

				_, err := plugins.Custom(sdk.Context{}, nil)
				suite.Require().ErrorIs(err, wasmvmtypes.UnsupportedRequest{Kind: "Custom queries are not allowed"})

				_, err = plugins.Stargate(sdk.Context{}, &wasmvmtypes.StargateQuery{})
				suite.Require().ErrorIs(err, wasmvmtypes.UnsupportedRequest{Kind: "'' path is not allowed from the contract"})
			},
		},
		{
			"success: custom querier",
			func() {
				querierOption := keeper.WithQueryPlugins(&keeper.QueryPlugins{
					Custom: mockErrorCustomQuerier(),
				})
				k = keeper.NewKeeperWithVM(
					GetSimApp(suite.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(suite.chainA).GetKey(types.StoreKey)),
					GetSimApp(suite.chainA).IBCKeeper.ClientKeeper,
					GetSimApp(suite.chainA).WasmClientKeeper.GetAuthority(),
					GetSimApp(suite.chainA).WasmClientKeeper.GetVM(),
					GetSimApp(suite.chainA).GRPCQueryRouter(),
					querierOption,
				)
			},
			func(k keeper.Keeper) {
				plugins := k.GetQueryPlugins()

				_, err := plugins.Custom(sdk.Context{}, nil)
				suite.Require().ErrorContains(err, "custom querier error for TestNewKeeperWithOptions")

				_, err = plugins.Stargate(sdk.Context{}, &wasmvmtypes.StargateQuery{})
				suite.Require().ErrorIs(err, wasmvmtypes.UnsupportedRequest{Kind: "'' path is not allowed from the contract"})
			},
		},
		{
			"success: stargate querier",
			func() {
				querierOption := keeper.WithQueryPlugins(&keeper.QueryPlugins{
					Stargate: mockErrorStargateQuerier(),
				})
				k = keeper.NewKeeperWithVM(
					GetSimApp(suite.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(suite.chainA).GetKey(types.StoreKey)),
					GetSimApp(suite.chainA).IBCKeeper.ClientKeeper,
					GetSimApp(suite.chainA).WasmClientKeeper.GetAuthority(),
					GetSimApp(suite.chainA).WasmClientKeeper.GetVM(),
					GetSimApp(suite.chainA).GRPCQueryRouter(),
					querierOption,
				)
			},
			func(k keeper.Keeper) {
				plugins := k.GetQueryPlugins()

				_, err := plugins.Custom(sdk.Context{}, nil)
				suite.Require().ErrorIs(err, wasmvmtypes.UnsupportedRequest{Kind: "Custom queries are not allowed"})

				_, err = plugins.Stargate(sdk.Context{}, &wasmvmtypes.StargateQuery{})
				suite.Require().ErrorContains(err, "stargate querier error for TestNewKeeperWithOptions")
			},
		},
		{
			"success: both queriers",
			func() {
				querierOption := keeper.WithQueryPlugins(&keeper.QueryPlugins{
					Custom:   mockErrorCustomQuerier(),
					Stargate: mockErrorStargateQuerier(),
				})
				k = keeper.NewKeeperWithVM(
					GetSimApp(suite.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(suite.chainA).GetKey(types.StoreKey)),
					GetSimApp(suite.chainA).IBCKeeper.ClientKeeper,
					GetSimApp(suite.chainA).WasmClientKeeper.GetAuthority(),
					GetSimApp(suite.chainA).WasmClientKeeper.GetVM(),
					GetSimApp(suite.chainA).GRPCQueryRouter(),
					querierOption,
				)
			},
			func(k keeper.Keeper) {
				plugins := k.GetQueryPlugins()

				_, err := plugins.Custom(sdk.Context{}, nil)
				suite.Require().ErrorContains(err, "custom querier error for TestNewKeeperWithOptions")

				_, err = plugins.Stargate(sdk.Context{}, &wasmvmtypes.StargateQuery{})
				suite.Require().ErrorContains(err, "stargate querier error for TestNewKeeperWithOptions")
			},
		},
	}

	for _, tc := range testCases {
		suite.SetupTest()

		suite.Run(tc.name, func() {
			// make sure the default query plugins are set
			k.SetQueryPlugins(keeper.NewDefaultQueryPlugins(GetSimApp(suite.chainA).GRPCQueryRouter()))

			tc.malleate()
			tc.verifyFn(k)

			// reset query plugins after each test
			k.SetQueryPlugins(keeper.NewDefaultQueryPlugins(GetSimApp(suite.chainA).GRPCQueryRouter()))
		})
	}
}
