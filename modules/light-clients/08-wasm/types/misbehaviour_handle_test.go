package types_test

import (
	"encoding/json"
	"errors"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

func (suite *TypesTestSuite) TestVerifyClientMessage() {
	var clientMsg exported.ClientMessage
	contractError := errors.New("callbackFn error")

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: valid misbehaviour",
			func() {
				suite.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					var msg *types.QueryMsg

					err := json.Unmarshal(queryMsg, &msg)
					suite.Require().NoError(err)

					suite.Require().NotNil(msg.VerifyClientMessage)
					suite.Require().NotNil(msg.VerifyClientMessage.ClientMessage)
					suite.Require().Nil(msg.Status)
					suite.Require().Nil(msg.CheckForMisbehaviour)
					suite.Require().Nil(msg.TimestampAtHeight)
					suite.Require().Nil(msg.ExportMetadata)

					resp, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return resp, types.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: invalid client message",
			func() {
				clientMsg = &ibctmtypes.Header{}

				suite.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return resp, types.DefaultGasUsed, nil
				})
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: error return from contract vm",
			func() {
				suite.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					return nil, 0, contractError
				})
			},
			contractError,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			clientState := endpoint.GetClientState()

			clientMsg = &types.ClientMessage{
				Data: []byte{1},
			}

			tc.malleate()

			err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.App.AppCodec(), suite.store, clientMsg)

			if tc.expErr != nil {
				suite.Require().ErrorIs(err, tc.expErr)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestCheckForMisbehaviour() {
	var clientMessage exported.ClientMessage
	var foundMisbehaviour bool

	testCases := []struct {
		name                 string
		malleate             func()
		expFoundMisbehaviour bool
		expPanic             error
	}{
		{
			"success: no misbehaviour",
			func() {
				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.CheckForMisbehaviourResult{FoundMisbehaviour: false})
					suite.Require().NoError(err)
					return resp, types.DefaultGasUsed, nil
				})
			},
			false,
			nil,
		},
		{
			"success: misbehaviour found", func() {
				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp, err := json.Marshal(types.CheckForMisbehaviourResult{FoundMisbehaviour: true})
					suite.Require().NoError(err)
					return resp, types.DefaultGasUsed, nil
				})
			},
			true,
			nil,
		},
		{
			"success: contract error, resp cannot be marshalled", func() {
				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					resp := "cannot be unmarshalled"
					return []byte(resp), types.DefaultGasUsed, nil
				})
			},
			false,
			nil,
		},
		{
			"success: vm returns error, ", func() {
				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					return nil, 0, errors.New("invalid block ID")
				})
			},
			false,
			nil,
		},
		{
			"success: invalid client message", func() {
				clientMessage = &ibctmtypes.Header{}

				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					// this test case will not reach the VM
					return nil, 0, nil
				})
			},
			false,
			nil,
		},
		{
			"failure: contract panics, panic propogated", func() {
				suite.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) ([]byte, uint64, error) {
					panic(errors.New("panic in query to contract"))
				})
			},
			false,
			errors.New("panic in query to contract"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmWithMockVM()
			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			clientState := endpoint.GetClientState()
			clientMessage = &types.ClientMessage{
				Data: []byte{1},
			}

			tc.malleate()

			if tc.expPanic == nil {
				foundMisbehaviour = clientState.CheckForMisbehaviour(suite.ctx, suite.chainA.App.AppCodec(), suite.store, clientMessage)
				suite.Require().Equal(tc.expFoundMisbehaviour, foundMisbehaviour)
			} else {
				suite.PanicsWithError(
					tc.expPanic.Error(),
					func() {
						clientState.CheckForMisbehaviour(suite.ctx, suite.chainA.App.AppCodec(), suite.store, clientMessage)
					})
			}
		})
	}
}
