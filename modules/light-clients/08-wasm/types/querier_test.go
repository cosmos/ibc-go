package types_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

type CustomQuery struct {
	Echo *QueryEcho `json:"echo,omitempty"`
}

type QueryEcho struct {
	Data string `json:"data"`
}

func MockCustomQuerier() func(sdk.Context, json.RawMessage) ([]byte, error) {
	return func(ctx sdk.Context, request json.RawMessage) ([]byte, error) {
		var customQuery CustomQuery
		err := json.Unmarshal([]byte(request), &customQuery)
		if err != nil {
			return nil, wasmtesting.ErrMockContract
		}

		if customQuery.Echo != nil {
			data, err := json.Marshal(customQuery.Echo.Data)
			return data, err
		}

		return nil, wasmtesting.ErrMockContract
	}
}

func (suite *TypesTestSuite) TestCustomQuery() {
	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"success: custom query",
			func() {
				querierPlugin := types.QueryPlugins{
					Custom: MockCustomQuerier(),
				}
				types.SetQueryPlugins(&querierPlugin)

				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, querier wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					echo := CustomQuery{
						Echo: &QueryEcho{
							Data: "hello world",
						},
					}
					echoJSON, err := json.Marshal(echo)
					suite.Require().NoError(err)

					resp, err := querier.Query(wasmvmtypes.QueryRequest{
						Custom: json.RawMessage(echoJSON),
					}, math.MaxUint64)
					suite.Require().NoError(err)

					var respData string
					err = json.Unmarshal(resp, &respData)
					suite.Require().NoError(err)
					suite.Require().Equal("hello world", respData)

					resp, err = json.Marshal(types.StatusResult{Status: exported.Active.String()})
					suite.Require().NoError(err)

					return resp, wasmtesting.DefaultGasUsed, nil
				})
			},
		},
		{
			"failure: default query",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, querier wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := querier.Query(wasmvmtypes.QueryRequest{Custom: json.RawMessage("{}")}, math.MaxUint64)
					suite.Require().ErrorIs(err, wasmvmtypes.UnsupportedRequest{Kind: "Custom queries are not allowed"})
					suite.Require().Nil(resp)

					return nil, wasmtesting.DefaultGasUsed, err
				})
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			// reset query plugins after each test
			types.SetQueryPlugins(types.NewDefaultQueryPlugins())

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
			clientState := endpoint.GetClientState()
			clientState.Status(suite.chainA.GetContext(), clientStore, suite.chainA.App.AppCodec())
		})
	}
}

func (suite *TypesTestSuite) TestStargateQuery() {
	typeURL := "/ibc.lightclients.wasm.v1.Query/Checksums"

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"success: custom query",
			func() {
				querierPlugin := types.QueryPlugins{
					Stargate: types.AcceptListStargateQuerier([]string{typeURL}),
				}

				types.SetQueryPlugins(&querierPlugin)

				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, querier wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					queryRequest := types.QueryChecksumsRequest{}
					bz, err := queryRequest.Marshal()
					suite.Require().NoError(err)

					resp, err := querier.Query(wasmvmtypes.QueryRequest{
						Stargate: &wasmvmtypes.StargateQuery{
							Path: typeURL,
							Data: bz,
						},
					}, math.MaxUint64)
					suite.Require().NoError(err)

					var respData types.QueryChecksumsResponse
					err = respData.Unmarshal(resp)
					suite.Require().NoError(err)

					expChecksum := hex.EncodeToString(suite.checksum)

					suite.Require().Len(respData.Checksums, 1)
					suite.Require().Equal(expChecksum, respData.Checksums[0])

					return resp, wasmtesting.DefaultGasUsed, nil
				})
			},
		},
		{
			"failure: default querier",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, querier wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					queryRequest := types.QueryChecksumsRequest{}
					bz, err := queryRequest.Marshal()
					suite.Require().NoError(err)

					resp, err := querier.Query(wasmvmtypes.QueryRequest{
						Stargate: &wasmvmtypes.StargateQuery{
							Path: typeURL,
							Data: bz,
						},
					}, math.MaxUint64)
					suite.Require().ErrorIs(err, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", typeURL)})
					suite.Require().Nil(resp)

					return nil, wasmtesting.DefaultGasUsed, err
				})
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
			clientState := endpoint.GetClientState()
			clientState.Status(suite.chainA.GetContext(), clientStore, suite.chainA.App.AppCodec())

			// reset query plugins after each test
			types.SetQueryPlugins(types.NewDefaultQueryPlugins())
		})
	}
}
