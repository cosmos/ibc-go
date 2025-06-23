package keeper_test

import (
	"encoding/json"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

func (s *KeeperTestSuite) TestWasmInstantiate() {
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				s.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					// Ensure GoAPI is set
					s.Require().NotNil(goapi.CanonicalizeAddress)
					s.Require().NotNil(goapi.HumanizeAddress)
					s.Require().NotNil(goapi.ValidateAddress)

					var payload types.InstantiateMessage
					err := json.Unmarshal(initMsg, &payload)
					s.Require().NoError(err)

					wrappedClientState, ok := clienttypes.MustUnmarshalClientState(s.chainA.App.AppCodec(), payload.ClientState).(*ibctm.ClientState)
					s.Require().True(ok)

					clientState := types.NewClientState(payload.ClientState, payload.Checksum, wrappedClientState.LatestHeight)
					clientStateBz := clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), clientState)
					store.Set(host.ClientStateKey(), clientStateBz)

					consensusState := types.NewConsensusState(payload.ConsensusState)
					consensusStateBz := clienttypes.MustMarshalConsensusState(s.chainA.App.AppCodec(), consensusState)
					store.Set(host.ConsensusStateKey(clientState.LatestHeight), consensusStateBz)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{}}, 0, nil
				}
			},
			nil,
		},
		{
			"failure: vm returns error",
			func() {
				s.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockVM
				}
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				s.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, 0, nil
				}
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: contract returns non-empty messages",
			func() {
				s.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Messages: []wasmvmtypes.SubMsg{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmSubMessagesNotAllowed,
		},
		{
			"failure: contract returns non-empty events",
			func() {
				s.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Events: []wasmvmtypes.Event{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmEventsNotAllowed,
		},
		{
			"failure: contract returns non-empty attributes",
			func() {
				s.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Attributes: []wasmvmtypes.EventAttribute{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmAttributesNotAllowed,
		},
		{
			"failure: change clientstate type",
			func() {
				s.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Set(host.ClientStateKey(), []byte("changed client state"))

					data, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: delete clientstate",
			func() {
				s.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Delete(host.ClientStateKey())
					data, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: unmarshallable clientstate",
			func() {
				s.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Set(host.ClientStateKey(), []byte("invalid json"))
					data, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: change checksum",
			func() {
				s.mockVM.InstantiateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.InstantiateMessage
					err := json.Unmarshal(initMsg, &payload)
					s.Require().NoError(err)

					// Change the checksum to something else.
					wrappedClientState, ok := clienttypes.MustUnmarshalClientState(s.chainA.App.AppCodec(), payload.ClientState).(*ibctm.ClientState)
					s.Require().True(ok)
					clientState := types.NewClientState(payload.ClientState, []byte("new checksum"), wrappedClientState.LatestHeight)
					store.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), clientState))

					resp, err := json.Marshal(types.UpdateStateResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()
			checksum := s.storeWasmCode(wasmtesting.Code)

			tc.malleate()

			initMsg := types.InstantiateMessage{
				ClientState:    clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), wasmtesting.MockTendermitClientState),
				ConsensusState: clienttypes.MustMarshalConsensusState(s.chainA.App.AppCodec(), wasmtesting.MockTendermintClientConsensusState),
				Checksum:       checksum,
			}

			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), defaultWasmClientID)
			wasmClientKeeper := GetSimApp(s.chainA).WasmClientKeeper
			err := wasmClientKeeper.WasmInstantiate(s.chainA.GetContext(), defaultWasmClientID, clientStore, &types.ClientState{Checksum: checksum}, initMsg)

			if tc.expError == nil {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestWasmMigrate() {
	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, goapi wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					// Ensure GoAPI is set
					s.Require().NotNil(goapi.CanonicalizeAddress)
					s.Require().NotNil(goapi.HumanizeAddress)
					s.Require().NotNil(goapi.ValidateAddress)

					resp, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, 0, nil
				}
			},
			nil,
		},
		{
			"failure: vm returns error",
			func() {
				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockVM
				}
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, 0, nil
				}
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: contract returns non-empty messages",
			func() {
				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Messages: []wasmvmtypes.SubMsg{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmSubMessagesNotAllowed,
		},
		{
			"failure: contract returns non-empty events",
			func() {
				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Events: []wasmvmtypes.Event{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmEventsNotAllowed,
		},
		{
			"failure: contract returns non-empty attributes",
			func() {
				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Attributes: []wasmvmtypes.EventAttribute{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmAttributesNotAllowed,
		},
		{
			"failure: change clientstate type",
			func() {
				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Set(host.ClientStateKey(), []byte("changed client state"))

					data, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: delete clientstate",
			func() {
				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Delete(host.ClientStateKey())
					data, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: unmarshallable clientstate",
			func() {
				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Set(host.ClientStateKey(), []byte("invalid json"))
					data, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmInvalidContractModification,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()
			_ = s.storeWasmCode(wasmtesting.Code)

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)

			tc.malleate()

			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), defaultWasmClientID)
			wasmClientKeeper := GetSimApp(s.chainA).WasmClientKeeper
			err = wasmClientKeeper.WasmMigrate(s.chainA.GetContext(), clientStore, &types.ClientState{}, defaultWasmClientID, []byte("{}"))

			if tc.expError == nil {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestWasmQuery() {
	var payload types.QueryMsg

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				s.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, goapi wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					// Ensure GoAPI is set
					s.Require().NotNil(goapi.CanonicalizeAddress)
					s.Require().NotNil(goapi.HumanizeAddress)
					s.Require().NotNil(goapi.ValidateAddress)

					resp, err := json.Marshal(types.StatusResult{Status: exported.Frozen.String()})
					s.Require().NoError(err)

					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: vm returns error",
			func() {
				s.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockVM
				})
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				s.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return &wasmvmtypes.QueryResult{Err: wasmtesting.ErrMockContract.Error()}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()
			_ = s.storeWasmCode(wasmtesting.Code)

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)

			clientState := endpoint.GetClientState()
			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), endpoint.ClientID)

			wasmClientState, ok := clientState.(*types.ClientState)
			s.Require().True(ok)

			payload = types.QueryMsg{Status: &types.StatusMsg{}}

			tc.malleate()

			wasmClientKeeper := GetSimApp(s.chainA).WasmClientKeeper
			res, err := wasmClientKeeper.WasmQuery(s.chainA.GetContext(), endpoint.ClientID, clientStore, wasmClientState, payload)

			if tc.expError == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestWasmSudo() {
	var payload types.SudoMsg

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, goapi wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					// Ensure GoAPI is set
					s.Require().NotNil(goapi.CanonicalizeAddress)
					s.Require().NotNil(goapi.HumanizeAddress)
					s.Require().NotNil(goapi.ValidateAddress)

					resp, err := json.Marshal(types.UpdateStateResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: vm returns error",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockVM
				})
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: contract returns non-empty messages",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Messages: []wasmvmtypes.SubMsg{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmSubMessagesNotAllowed,
		},
		{
			"failure: contract returns non-empty events",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Events: []wasmvmtypes.Event{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmEventsNotAllowed,
		},
		{
			"failure: contract returns non-empty attributes",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					resp := wasmvmtypes.Response{Attributes: []wasmvmtypes.EventAttribute{{}}}

					return &wasmvmtypes.ContractResult{Ok: &resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmAttributesNotAllowed,
		},
		{
			"failure: unmarshallable clientstate bytes",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Set(host.ClientStateKey(), []byte("invalid json"))

					resp, err := json.Marshal(types.UpdateStateResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: delete clientstate",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					store.Delete(host.ClientStateKey())

					resp, err := json.Marshal(types.UpdateStateResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmInvalidContractModification,
		},
		{
			"failure: change checksum",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					clientState := s.chainA.GetClientState(defaultWasmClientID)
					clientState.(*types.ClientState).Checksum = []byte("new checksum")
					store.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), clientState))

					resp, err := json.Marshal(types.UpdateStateResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmInvalidContractModification,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()
			_ = s.storeWasmCode(wasmtesting.Code)

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)

			clientState := endpoint.GetClientState()
			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), endpoint.ClientID)

			wasmClientState, ok := clientState.(*types.ClientState)
			s.Require().True(ok)

			payload = types.SudoMsg{UpdateState: &types.UpdateStateMsg{}}

			tc.malleate()

			wasmClientKeeper := GetSimApp(s.chainA).WasmClientKeeper
			res, err := wasmClientKeeper.WasmSudo(s.chainA.GetContext(), endpoint.ClientID, clientStore, wasmClientState, payload)

			if tc.expError == nil {
				s.Require().NoError(err)
				s.Require().NotNil(res)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
