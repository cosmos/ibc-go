package types_test

import (
	"encoding/json"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

func (suite *TypesTestSuite) TestWasmInit() {
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					return nil, 0, nil
				}
			},
			nil,
		},
		{
			"failure: contract returns error",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					return nil, 0, wasmtesting.ErrMockContract
				}
			},
			types.ErrWasmContractCallFailed,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			tc.malleate()

			err := types.WasmInstantiate(suite.ctx, suite.store, &types.ClientState{}, types.InstantiateMessage{})

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *TypesTestSuite) TestWasmQuery() {
	var payload types.QueryMsg

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.StatusResult{Status: exported.Frozen.String()})
					suite.Require().NoError(err)
					return resp, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: contract returns error",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockContract
				})
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: response fails to unmarshal",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					return []byte("invalid json"), wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmInvalidResponseData,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			clientState := endpoint.GetClientState()

			wasmClientState, ok := clientState.(*types.ClientState)
			suite.Require().True(ok)

			payload = types.QueryMsg{Status: &types.StatusMsg{}}

			tc.malleate()

			res, err := types.WasmQuery[types.StatusResult](suite.ctx, suite.store, wasmClientState, payload)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *TypesTestSuite) TestWasmSudo() {
	var payload types.SudoMsg

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					resp, err := json.Marshal(types.UpdateStateResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.Response{Data: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: contract returns error",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockContract
				})
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: contract returns non-empty messages",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					resp := wasmvmtypes.Response{Messages: []wasmvmtypes.SubMsg{{}}}

					return &resp, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmSubMessagesNotAllowed,
		},
		{
			"failure: contract returns non-empty events",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					resp := wasmvmtypes.Response{Events: []wasmvmtypes.Event{{}}}

					return &resp, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmEventsNotAllowed,
		},
		{
			"failure: contract returns non-empty attributes",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					resp := wasmvmtypes.Response{Attributes: []wasmvmtypes.EventAttribute{{}}}

					return &resp, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmAttributesNotAllowed,
		},
		{
			"failure: response fails to unmarshal",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
					return &wasmvmtypes.Response{Data: []byte("invalid json")}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmInvalidResponseData,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			clientState := endpoint.GetClientState()

			wasmClientState, ok := clientState.(*types.ClientState)
			suite.Require().True(ok)

			payload = types.SudoMsg{UpdateState: &types.UpdateStateMsg{}}

			tc.malleate()

			res, err := types.WasmSudo[types.UpdateStateResult](suite.ctx, suite.store, wasmClientState, payload)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
