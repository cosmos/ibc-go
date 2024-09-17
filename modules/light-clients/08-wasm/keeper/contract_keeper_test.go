package keeper_test

import (
	"encoding/json"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
)

func (suite *KeeperTestSuite) TestWasmInstantiate() {
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					// Ensure GoAPI is set
					suite.Require().NotNil(goapi.CanonicalizeAddress)
					suite.Require().NotNil(goapi.HumanizeAddress)
					suite.Require().NotNil(goapi.ValidateAddress)

					var payload types.InstantiateMessage
					err := json.Unmarshal(initMsg, &payload)
					suite.Require().NoError(err)

					wrappedClientState, ok := clienttypes.MustUnmarshalClientState(suite.chainA.App.AppCodec(), payload.ClientState).(*ibctm.ClientState)
					suite.Require().True(ok)

					clientState := types.NewClientState(payload.ClientState, payload.Checksum, wrappedClientState.LatestHeight)
					clientStateBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), clientState)
					store.Set(host.ClientStateKey(), clientStateBz)

					consensusState := types.NewConsensusState(payload.ConsensusState)
					consensusStateBz := clienttypes.MustMarshalConsensusState(suite.chainA.App.AppCodec(), consensusState)
					store.Set(host.ConsensusStateKey(clientState.LatestHeight), consensusStateBz)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{}}, 0, nil
				}
			},
			nil,
		},
		{
			"failure: vm returns error",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockVM
				}
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, 0, nil
				}
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: contract returns non-empty messages",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Messages: []wasmvmtypes.SubMsg{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmSubMessagesNotAllowed,
		},
		{
			"failure: contract returns non-empty events",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Events: []wasmvmtypes.Event{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmEventsNotAllowed,
		},
		{
			"failure: contract returns non-empty attributes",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Attributes: []wasmvmtypes.EventAttribute{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmAttributesNotAllowed,
		},
		{
			"failure: change clientstate type",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Set(host.ClientStateKey(), []byte("changed client state"))

					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: delete clientstate",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Delete(host.ClientStateKey())
					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: unmarshallable clientstate",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Set(host.ClientStateKey(), []byte("invalid json"))
					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: change checksum",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.InstantiateMessage
					err := json.Unmarshal(initMsg, &payload)
					suite.Require().NoError(err)

					// Change the checksum to something else.
					wrappedClientState, ok := clienttypes.MustUnmarshalClientState(suite.chainA.App.AppCodec(), payload.ClientState).(*ibctm.ClientState)
					suite.Require().True(ok)
					clientState := types.NewClientState(payload.ClientState, []byte("new checksum"), wrappedClientState.LatestHeight)
					store.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), clientState))

					resp, err := json.Marshal(types.UpdateStateResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()
			checksum := suite.storeWasmCode(wasmtesting.Code)

			tc.malleate()

			initMsg := types.InstantiateMessage{
				ClientState:    clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), wasmtesting.MockTendermitClientState),
				ConsensusState: clienttypes.MustMarshalConsensusState(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientConsensusState),
				Checksum:       checksum,
			}

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), defaultWasmClientID)
			wasmClientKeeper := GetSimApp(suite.chainA).WasmClientKeeper
			err := wasmClientKeeper.WasmInstantiate(suite.chainA.GetContext(), defaultWasmClientID, clientStore, &types.ClientState{Checksum: checksum}, initMsg)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestWasmMigrate() {
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, goapi wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					// Ensure GoAPI is set
					suite.Require().NotNil(goapi.CanonicalizeAddress)
					suite.Require().NotNil(goapi.HumanizeAddress)
					suite.Require().NotNil(goapi.ValidateAddress)

					resp, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, 0, nil
				}
			},
			nil,
		},
		{
			"failure: vm returns error",
			func() {
				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockVM
				}
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, 0, nil
				}
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: contract returns non-empty messages",
			func() {
				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Messages: []wasmvmtypes.SubMsg{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmSubMessagesNotAllowed,
		},
		{
			"failure: contract returns non-empty events",
			func() {
				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Events: []wasmvmtypes.Event{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmEventsNotAllowed,
		},
		{
			"failure: contract returns non-empty attributes",
			func() {
				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Attributes: []wasmvmtypes.EventAttribute{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmAttributesNotAllowed,
		},
		{
			"failure: change clientstate type",
			func() {
				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Set(host.ClientStateKey(), []byte("changed client state"))

					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: delete clientstate",
			func() {
				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Delete(host.ClientStateKey())
					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: unmarshallable clientstate",
			func() {
				suite.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Set(host.ClientStateKey(), []byte("invalid json"))
					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()
			_ = suite.storeWasmCode(wasmtesting.Code)

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), defaultWasmClientID)
			wasmClientKeeper := GetSimApp(suite.chainA).WasmClientKeeper
			err = wasmClientKeeper.WasmMigrate(suite.chainA.GetContext(), clientStore, &types.ClientState{}, defaultWasmClientID, []byte("{}"))

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestWasmQuery() {
	var payload types.QueryMsg

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, goapi wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					// Ensure GoAPI is set
					suite.Require().NotNil(goapi.CanonicalizeAddress)
					suite.Require().NotNil(goapi.HumanizeAddress)
					suite.Require().NotNil(goapi.ValidateAddress)

					resp, err := json.Marshal(types.StatusResult{Status: exported.Frozen.String()})
					suite.Require().NoError(err)

					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: vm returns error",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockVM
				})
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return &wasmvmtypes.QueryResult{Err: wasmtesting.ErrMockContract.Error()}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()
			_ = suite.storeWasmCode(wasmtesting.Code)

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			clientState := endpoint.GetClientState()
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)

			wasmClientState, ok := clientState.(*types.ClientState)
			suite.Require().True(ok)

			payload = types.QueryMsg{Status: &types.StatusMsg{}}

			tc.malleate()

			wasmClientKeeper := GetSimApp(suite.chainA).WasmClientKeeper
			res, err := wasmClientKeeper.WasmQuery(suite.chainA.GetContext(), endpoint.ClientID, clientStore, wasmClientState, payload)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestWasmSudo() {
	var payload types.SudoMsg

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, goapi wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					// Ensure GoAPI is set
					suite.Require().NotNil(goapi.CanonicalizeAddress)
					suite.Require().NotNil(goapi.HumanizeAddress)
					suite.Require().NotNil(goapi.ValidateAddress)

					resp, err := json.Marshal(types.UpdateStateResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: vm returns error",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockVM
				})
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: contract returns non-empty messages",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Messages: []wasmvmtypes.SubMsg{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmSubMessagesNotAllowed,
		},
		{
			"failure: contract returns non-empty events",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Events: []wasmvmtypes.Event{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmEventsNotAllowed,
		},
		{
			"failure: contract returns non-empty attributes",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Attributes: []wasmvmtypes.EventAttribute{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmAttributesNotAllowed,
		},
		{
			"failure: unmarshallable clientstate bytes",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Set(host.ClientStateKey(), []byte("invalid json"))

					resp, err := json.Marshal(types.UpdateStateResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: delete clientstate",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Delete(host.ClientStateKey())

					resp, err := json.Marshal(types.UpdateStateResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: change checksum",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					clientState := suite.chainA.GetClientState(defaultWasmClientID)
					clientState.(*types.ClientState).Checksum = []byte("new checksum")
					store.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), clientState))

					resp, err := json.Marshal(types.UpdateStateResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmInvalidContractModification,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()
			_ = suite.storeWasmCode(wasmtesting.Code)

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			clientState := endpoint.GetClientState()
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)

			wasmClientState, ok := clientState.(*types.ClientState)
			suite.Require().True(ok)

			payload = types.SudoMsg{UpdateState: &types.UpdateStateMsg{}}

			tc.malleate()

			wasmClientKeeper := GetSimApp(suite.chainA).WasmClientKeeper
			res, err := wasmClientKeeper.WasmSudo(suite.chainA.GetContext(), endpoint.ClientID, clientStore, wasmClientState, payload)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
