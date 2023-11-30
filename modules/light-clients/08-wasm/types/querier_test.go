package types_test

import (
	"encoding/json"
	"math"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
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

type CustomQueryHandler struct{}

func (*CustomQueryHandler) GasConsumed() uint64 {
	return 0
}

func (*CustomQueryHandler) Query(request wasmvmtypes.QueryRequest, gasLimit uint64) ([]byte, error) {
	var customQuery CustomQuery
	err := json.Unmarshal([]byte(request.Custom), &customQuery)
	if err != nil {
		return nil, wasmtesting.ErrMockContract
	}

	if customQuery.Echo != nil {
		data, err := json.Marshal(customQuery.Echo.Data)
		return data, err
	}

	return nil, wasmtesting.ErrMockContract
}

func (suite *TypesTestSuite) TestCustomQuery() {
	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			"success: custom query",
			func() {
				ibcwasm.SetQuerier(&CustomQueryHandler{})

				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
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
			"default query",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
					_, err := querier.Query(wasmvmtypes.QueryRequest{}, math.MaxUint64)
					suite.Require().Error(err)

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
		})
	}
}
