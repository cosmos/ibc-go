package wasm_test

import (
	"encoding/json"
	"time"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	storetypes "cosmossdk.io/store/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
)

const (
	tmClientID   = "07-tendermint-0"
	wasmClientID = "08-wasm-0"
	// Used for checks where look ups for valid client id should fail.
	unusedWasmClientID = "08-wasm-100"
)

func (suite *WasmTestSuite) TestStatus() {
	testCases := []struct {
		name      string
		malleate  func()
		expStatus exported.Status
	}{
		{
			"client is active",
			func() {},
			exported.Active,
		},
		{
			"client is frozen",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					resp, err := json.Marshal(types.StatusResult{Status: exported.Frozen.String()})
					suite.Require().NoError(err)
					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			exported.Frozen,
		},
		{
			"client status is expired",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					resp, err := json.Marshal(types.StatusResult{Status: exported.Expired.String()})
					suite.Require().NoError(err)
					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			exported.Expired,
		},
		{
			"client status is unknown: vm returns an error",
			func() {
				suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockContract
				})
			},
			exported.Unknown,
		},
		{
			"client status is unauthorized: checksum is not stored",
			func() {
				err := ibcwasm.Checksums.Remove(suite.chainA.GetContext(), suite.checksum)
				suite.Require().NoError(err)
			},
			exported.Unauthorized,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(endpoint.ClientID)
			suite.Require().True(found)

			tc.malleate()

			status := lightClientModule.Status(suite.chainA.GetContext(), endpoint.ClientID)
			suite.Require().Equal(tc.expStatus, status)
		})
	}
}

func (suite *WasmTestSuite) TestGetTimestampAtHeight() {
	var height exported.Height

	expectedTimestamp := uint64(time.Now().UnixNano())

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				suite.mockVM.RegisterQueryCallback(types.TimestampAtHeightMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					var payload types.QueryMsg
					err := json.Unmarshal(queryMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().NotNil(payload.TimestampAtHeight)
					suite.Require().Nil(payload.CheckForMisbehaviour)
					suite.Require().Nil(payload.Status)
					suite.Require().Nil(payload.VerifyClientMessage)

					resp, err := json.Marshal(types.TimestampAtHeightResult{Timestamp: expectedTimestamp})
					suite.Require().NoError(err)

					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: vm returns error",
			func() {
				suite.mockVM.RegisterQueryCallback(types.TimestampAtHeightMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockVM
				})
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				suite.mockVM.RegisterQueryCallback(types.TimestampAtHeightMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return &wasmvmtypes.QueryResult{Err: wasmtesting.ErrMockContract.Error()}, 0, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"error: invalid height",
			func() {
				height = ibcmock.Height{}
			},
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			clientState, ok := endpoint.GetClientState().(*types.ClientState)
			suite.Require().True(ok)

			height = clientState.LatestHeight

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(endpoint.ClientID)
			suite.Require().True(found)

			tc.malleate()

			timestamp, err := lightClientModule.TimestampAtHeight(suite.chainA.GetContext(), endpoint.ClientID, height)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expectedTimestamp, timestamp)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *WasmTestSuite) TestInitialize() {
	var (
		consensusState exported.ConsensusState
		clientState    *types.ClientState
		clientStore    storetypes.KVStore
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: new mock client",
			func() {},
			nil,
		},
		{
			"success: validate contract address",
			func() {
				suite.mockVM.InstantiateFn = func(_ wasmvm.Checksum, env wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.InstantiateMessage
					err := json.Unmarshal(initMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().Equal(env.Contract.Address, wasmClientID)

					wrappedClientState := clienttypes.MustUnmarshalClientState(suite.chainA.App.AppCodec(), payload.ClientState).(*ibctm.ClientState)

					clientState := types.NewClientState(payload.ClientState, payload.Checksum, wrappedClientState.LatestHeight)
					clientStateBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), clientState)
					store.Set(host.ClientStateKey(), clientStateBz)

					consensusState := types.NewConsensusState(payload.ConsensusState)
					consensusStateBz := clienttypes.MustMarshalConsensusState(suite.chainA.App.AppCodec(), consensusState)
					store.Set(host.ConsensusStateKey(clientState.LatestHeight), consensusStateBz)

					resp, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, 0, nil
				}
			},
			nil,
		},
		{
			"failure: invalid consensus state",
			func() {
				// set upgraded consensus state to solomachine consensus state
				consensusState = &solomachine.ConsensusState{}
			},
			types.ErrInvalidData,
		},
		{
			"failure: checksum has not been stored.",
			func() {
				clientState = types.NewClientState([]byte{1}, []byte("unknown"), clienttypes.NewHeight(0, 1))
			},
			types.ErrInvalidChecksum,
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
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			wrappedClientStateBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), wasmtesting.MockTendermitClientState)
			wrappedClientConsensusStateBz := clienttypes.MustMarshalConsensusState(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientConsensusState)
			clientState = types.NewClientState(wrappedClientStateBz, suite.checksum, wasmtesting.MockTendermitClientState.LatestHeight)
			consensusState = types.NewConsensusState(wrappedClientConsensusStateBz)

			clientID := suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(suite.chainA.GetContext(), clientState.ClientType())
			clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)
			// Set client state in state
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(
				suite.chainA.GetContext(), clientID, clientState,
			)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			// Marshal client state and consensus state:
			clientStateBz := suite.chainA.Codec.MustMarshal(clientState)
			consensusStateBz := suite.chainA.Codec.MustMarshal(consensusState)

			err := lightClientModule.Initialize(suite.chainA.GetContext(), clientID, clientStateBz, consensusStateBz)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				expClientState := clienttypes.MustMarshalClientState(suite.chainA.Codec, clientState)
				suite.Require().Equal(expClientState, clientStore.Get(host.ClientStateKey()))

				expConsensusState := clienttypes.MustMarshalConsensusState(suite.chainA.Codec, consensusState)
				suite.Require().Equal(expConsensusState, clientStore.Get(host.ConsensusStateKey(clientState.LatestHeight)))
			} else {
				suite.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (suite *WasmTestSuite) TestVerifyMembership() {
	var (
		clientState      *types.ClientState
		expClientStateBz []byte
		path             exported.Path
		proof            []byte
		proofHeight      exported.Height
		value            []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				expClientStateBz = GetSimApp(suite.chainA).GetIBCKeeper().ClientKeeper.MustMarshalClientState(clientState)
				suite.mockVM.RegisterSudoCallback(types.VerifyMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().NotNil(payload.VerifyMembership)
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyNonMembership)
					suite.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					suite.Require().Equal(proofHeight, payload.VerifyMembership.Height)
					suite.Require().Equal(path, payload.VerifyMembership.Path)
					suite.Require().Equal(proof, payload.VerifyMembership.Proof)
					suite.Require().Equal(value, payload.VerifyMembership.Value)

					bz, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"success: with update client state",
			func() {
				suite.mockVM.RegisterSudoCallback(types.VerifyMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().NotNil(payload.VerifyMembership)
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyNonMembership)
					suite.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					suite.Require().Equal(proofHeight, payload.VerifyMembership.Height)
					suite.Require().Equal(path, payload.VerifyMembership.Path)
					suite.Require().Equal(proof, payload.VerifyMembership.Proof)
					suite.Require().Equal(value, payload.VerifyMembership.Value)

					bz, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					expClientStateBz = wasmtesting.CreateMockClientStateBz(suite.chainA.Codec, suite.checksum)
					store.Set(host.ClientStateKey(), expClientStateBz)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"contract returns invalid proof error",
			func() {
				proof = wasmtesting.MockInvalidProofBz

				suite.mockVM.RegisterSudoCallback(types.VerifyMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: commitmenttypes.ErrInvalidProof.Error()}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"proof height greater than client state latest height",
			func() {
				proofHeight = clienttypes.NewHeight(1, 100)
			},
			ibcerrors.ErrInvalidHeight,
		},
		{
			"invalid path argument",
			func() {
				path = ibcmock.KeyPath{}
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"proof height is invalid type",
			func() {
				proofHeight = ibcmock.Height{}
			},
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			var ok bool
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			path = commitmenttypes.NewMerklePath("/ibc/key/path")
			proof = wasmtesting.MockValidProofBz
			proofHeight = clienttypes.NewHeight(0, 1)
			value = []byte("value")

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
			clientState, ok = endpoint.GetClientState().(*types.ClientState)
			suite.Require().True(ok)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(endpoint.ClientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.VerifyMembership(suite.chainA.GetContext(), endpoint.ClientID, proofHeight, 0, 0, proof, path, value)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().Equal(expClientStateBz, clientStateBz)
			} else {
				suite.Require().ErrorIs(err, tc.expError, "unexpected error in VerifyMembership")
			}
		})
	}
}

func (suite *WasmTestSuite) TestVerifyNonMembership() {
	var (
		clientState      *types.ClientState
		expClientStateBz []byte
		path             exported.Path
		proof            []byte
		proofHeight      exported.Height
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				expClientStateBz = GetSimApp(suite.chainA).GetIBCKeeper().ClientKeeper.MustMarshalClientState(clientState)
				suite.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().NotNil(payload.VerifyNonMembership)
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyMembership)
					suite.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					suite.Require().Equal(proofHeight, payload.VerifyNonMembership.Height)
					suite.Require().Equal(path, payload.VerifyNonMembership.Path)
					suite.Require().Equal(proof, payload.VerifyNonMembership.Proof)

					bz, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"success: with update client state",
			func() {
				suite.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					suite.Require().NotNil(payload.VerifyNonMembership)
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyMembership)
					suite.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					suite.Require().Equal(proofHeight, payload.VerifyNonMembership.Height)
					suite.Require().Equal(path, payload.VerifyNonMembership.Path)
					suite.Require().Equal(proof, payload.VerifyNonMembership.Proof)

					bz, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					expClientStateBz = wasmtesting.CreateMockClientStateBz(suite.chainA.Codec, suite.checksum)
					store.Set(host.ClientStateKey(), expClientStateBz)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"wasm vm returns error",
			func() {
				proof = wasmtesting.MockInvalidProofBz

				suite.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockVM
				})
			},
			types.ErrVMError,
		},
		{
			"contract returns invalid proof error",
			func() {
				proof = wasmtesting.MockInvalidProofBz

				suite.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: commitmenttypes.ErrInvalidProof.Error()}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"proof height greater than client state latest height",
			func() {
				proofHeight = clienttypes.NewHeight(1, 100)
			},
			ibcerrors.ErrInvalidHeight,
		},
		{
			"invalid path argument",
			func() {
				path = ibcmock.KeyPath{}
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"proof height is invalid type",
			func() {
				proofHeight = ibcmock.Height{}
			},
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			var ok bool
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			path = commitmenttypes.NewMerklePath("/ibc/key/path")
			proof = wasmtesting.MockInvalidProofBz
			proofHeight = clienttypes.NewHeight(0, 1)

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
			clientState, ok = endpoint.GetClientState().(*types.ClientState)
			suite.Require().True(ok)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(endpoint.ClientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.VerifyNonMembership(suite.chainA.GetContext(), endpoint.ClientID, proofHeight, 0, 0, proof, path)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)

				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().Equal(expClientStateBz, clientStateBz)
			} else {
				suite.Require().ErrorIs(err, tc.expError, "unexpected error in VerifyNonMembership")
			}
		})
	}
}

func (suite *WasmTestSuite) TestRecoverClient() {
	var (
		expectedClientStateBz               []byte
		subjectClientID, substituteClientID string
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		// TODO(02-client routing): add successful test when light client module does not call into 08-wasm ClientState
		// {
		// 	"success",
		// 	func() {
		// 	},
		// 	nil,
		// },
		{
			"cannot parse malformed substitute client ID",
			func() {
				substituteClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"substitute client ID does not contain 08-wasm prefix",
			func() {
				substituteClientID = tmClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot find subject client state",
			func() {
				subjectClientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"cannot find substitute client state",
			func() {
				substituteClientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()
			expectedClientStateBz = nil

			subjectEndpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := subjectEndpoint.CreateClient()
			suite.Require().NoError(err)
			subjectClientID = subjectEndpoint.ClientID

			subjectClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), subjectClientID)

			substituteEndpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err = substituteEndpoint.CreateClient()
			suite.Require().NoError(err)
			substituteClientID = substituteEndpoint.ClientID

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(subjectClientID)
			suite.Require().True(found)

			tc.malleate()

			err = lightClientModule.RecoverClient(suite.chainA.GetContext(), subjectClientID, substituteClientID)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				clientStateBz := subjectClientStore.Get(host.ClientStateKey())
				suite.Require().Equal(expectedClientStateBz, clientStateBz)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *WasmTestSuite) TestVerifyUpgradeAndUpdateState() {
	var (
		clientID                                              string
		clientState                                           *types.ClientState
		upgradedClientState                                   exported.ClientState
		upgradedConsensusState                                exported.ConsensusState
		upgradedClientStateAny, upgradedConsensusStateAny     *codectypes.Any
		upgradedClientStateProof, upgradedConsensusStateProof []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				suite.mockVM.RegisterSudoCallback(types.VerifyUpgradeAndUpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg

					err := json.Unmarshal(sudoMsg, &payload)
					suite.Require().NoError(err)

					expectedUpgradedClient, ok := upgradedClientState.(*types.ClientState)
					suite.Require().True(ok)
					expectedUpgradedConsensus, ok := upgradedConsensusState.(*types.ConsensusState)
					suite.Require().True(ok)

					// verify payload values
					suite.Require().Equal(expectedUpgradedClient.Data, payload.VerifyUpgradeAndUpdateState.UpgradeClientState)
					suite.Require().Equal(expectedUpgradedConsensus.Data, payload.VerifyUpgradeAndUpdateState.UpgradeConsensusState)
					suite.Require().Equal(upgradedClientStateProof, payload.VerifyUpgradeAndUpdateState.ProofUpgradeClient)
					suite.Require().Equal(upgradedConsensusStateProof, payload.VerifyUpgradeAndUpdateState.ProofUpgradeConsensusState)

					// verify other Sudo fields are nil
					suite.Require().Nil(payload.UpdateState)
					suite.Require().Nil(payload.UpdateStateOnMisbehaviour)
					suite.Require().Nil(payload.VerifyMembership)
					suite.Require().Nil(payload.VerifyNonMembership)

					data, err := json.Marshal(types.EmptyResult{})
					suite.Require().NoError(err)

					// set new client state and consensus state
					bz, err := suite.chainA.Codec.MarshalInterface(upgradedClientState)
					suite.Require().NoError(err)

					store.Set(host.ClientStateKey(), bz)

					bz, err = suite.chainA.Codec.MarshalInterface(upgradedConsensusState)
					suite.Require().NoError(err)

					store.Set(host.ConsensusStateKey(expectedUpgradedClient.LatestHeight), bz)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"cannot find client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"upgraded client state is not wasm client state",
			func() {
				upgradedClientStateAny = &codectypes.Any{
					Value: []byte("invalid client state bytes"),
				}
			},
			clienttypes.ErrInvalidClient,
		},
		{
			"upgraded consensus state is not wasm consensus sate",
			func() {
				upgradedConsensusStateAny = &codectypes.Any{
					Value: []byte("invalid consensus state bytes"),
				}
			},
			clienttypes.ErrInvalidConsensus,
		},
		{
			"upgraded client state height is not greater than current height",
			func() {
				var err error
				latestHeight := clientState.LatestHeight
				newLatestHeight := clienttypes.NewHeight(latestHeight.GetRevisionNumber(), latestHeight.GetRevisionHeight()-1)

				wrappedUpgradedClient := wasmtesting.CreateMockTendermintClientState(newLatestHeight)
				wrappedUpgradedClientBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), wrappedUpgradedClient)
				upgradedClientState = types.NewClientState(wrappedUpgradedClientBz, clientState.Checksum, newLatestHeight)
				upgradedClientStateAny, err = codectypes.NewAnyWithValue(upgradedClientState)
				suite.Require().NoError(err)
			},
			ibcerrors.ErrInvalidHeight,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM() // reset suite

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientID = endpoint.ClientID

			clientState = endpoint.GetClientState().(*types.ClientState)
			latestHeight := clientState.LatestHeight

			newLatestHeight := clienttypes.NewHeight(latestHeight.GetRevisionNumber(), latestHeight.GetRevisionHeight()+1)
			wrappedUpgradedClient := wasmtesting.CreateMockTendermintClientState(newLatestHeight)
			wrappedUpgradedClientBz := clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), wrappedUpgradedClient)
			upgradedClientState = types.NewClientState(wrappedUpgradedClientBz, clientState.Checksum, newLatestHeight)
			upgradedClientStateAny, err = codectypes.NewAnyWithValue(upgradedClientState)
			suite.Require().NoError(err)

			wrappedUpgradedConsensus := ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("new-hash")), []byte("new-nextValsHash"))
			wrappedUpgradedConsensusBz := clienttypes.MustMarshalConsensusState(suite.chainA.App.AppCodec(), wrappedUpgradedConsensus)
			upgradedConsensusState = types.NewConsensusState(wrappedUpgradedConsensusBz)
			upgradedConsensusStateAny, err = codectypes.NewAnyWithValue(upgradedConsensusState)
			suite.Require().NoError(err)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), clientID)

			upgradedClientStateProof = wasmtesting.MockUpgradedClientStateProofBz
			upgradedConsensusStateProof = wasmtesting.MockUpgradedConsensusStateProofBz

			err = lightClientModule.VerifyUpgradeAndUpdateState(
				suite.chainA.GetContext(),
				clientID,
				upgradedClientStateAny.Value,
				upgradedConsensusStateAny.Value,
				upgradedClientStateProof,
				upgradedConsensusStateProof,
			)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				// verify new client state and consensus state
				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				expClientStateBz, err := suite.chainA.Codec.MarshalInterface(upgradedClientState)
				suite.Require().NoError(err)
				suite.Require().Equal(expClientStateBz, clientStateBz)

				consensusStateBz := clientStore.Get(host.ConsensusStateKey(endpoint.GetClientLatestHeight()))
				suite.Require().NotEmpty(consensusStateBz)

				expConsensusStateBz, err := suite.chainA.Codec.MarshalInterface(upgradedConsensusState)
				suite.Require().NoError(err)
				suite.Require().Equal(expConsensusStateBz, consensusStateBz)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
