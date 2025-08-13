package wasm_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	errorsmod "cosmossdk.io/errors"

	internaltypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/internal/types"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v10/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

const (
	tmClientID   = "07-tendermint-0"
	wasmClientID = "08-wasm-0"
	// Used for checks where look ups for valid client id should fail.
	unusedWasmClientID = "08-wasm-100"
)

func (s *WasmTestSuite) TestStatus() {
	var clientID string

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
				s.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					resp, err := json.Marshal(types.StatusResult{Status: exported.Frozen.String()})
					s.Require().NoError(err)
					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			exported.Frozen,
		},
		{
			"client status is expired",
			func() {
				s.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					resp, err := json.Marshal(types.StatusResult{Status: exported.Expired.String()})
					s.Require().NoError(err)
					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			exported.Expired,
		},
		{
			"client status is unknown: vm returns an error",
			func() {
				s.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockContract
				})
			},
			exported.Unknown,
		},
		{
			"client status is unauthorized: checksum is not stored",
			func() {
				wasmClientKeeper := GetSimApp(s.chainA).WasmClientKeeper
				err := wasmClientKeeper.GetChecksums().Remove(s.chainA.GetContext(), s.checksum)
				s.Require().NoError(err)
			},
			exported.Unauthorized,
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedWasmClientID
			},
			exported.Unknown,
		},
		{
			"failure: response fails to unmarshal",
			func() {
				s.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return &wasmvmtypes.QueryResult{Ok: []byte("invalid json")}, wasmtesting.DefaultGasUsed, nil
				})
			},
			exported.Unknown,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)
			clientID = endpoint.ClientID

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			status := lightClientModule.Status(s.chainA.GetContext(), clientID)
			s.Require().Equal(tc.expStatus, status)
		})
	}
}

func (s *WasmTestSuite) TestTimestampAtHeight() {
	var (
		clientID string
		height   exported.Height
	)
	expectedTimestamp := uint64(time.Now().UnixNano())

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				s.mockVM.RegisterQueryCallback(types.TimestampAtHeightMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					var payload types.QueryMsg
					err := json.Unmarshal(queryMsg, &payload)
					s.Require().NoError(err)

					s.Require().NotNil(payload.TimestampAtHeight)
					s.Require().Nil(payload.CheckForMisbehaviour)
					s.Require().Nil(payload.Status)
					s.Require().Nil(payload.VerifyClientMessage)

					resp, err := json.Marshal(types.TimestampAtHeightResult{Timestamp: expectedTimestamp})
					s.Require().NoError(err)

					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: vm returns error",
			func() {
				s.mockVM.RegisterQueryCallback(types.TimestampAtHeightMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockVM
				})
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				s.mockVM.RegisterQueryCallback(types.TimestampAtHeightMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return &wasmvmtypes.QueryResult{Err: wasmtesting.ErrMockContract.Error()}, 0, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: error: invalid height",
			func() {
				height = ibcmock.Height{}
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"failure: response fails to unmarshal",
			func() {
				s.mockVM.RegisterQueryCallback(types.TimestampAtHeightMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return &wasmvmtypes.QueryResult{Ok: []byte("invalid json")}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmInvalidResponseData,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)
			clientID = endpoint.ClientID

			clientState, ok := endpoint.GetClientState().(*types.ClientState)
			s.Require().True(ok)

			height = clientState.LatestHeight

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			timestamp, err := lightClientModule.TimestampAtHeight(s.chainA.GetContext(), clientID, height)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().Equal(expectedTimestamp, timestamp)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
				s.Require().Equal(uint64(0), timestamp)
			}
		})
	}
}

func (s *WasmTestSuite) TestInitialize() {
	var (
		consensusState exported.ConsensusState
		clientState    exported.ClientState
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
				s.mockVM.InstantiateFn = func(_ wasmvm.Checksum, env wasmvmtypes.Env, _ wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.InstantiateMessage
					err := json.Unmarshal(initMsg, &payload)
					s.Require().NoError(err)

					s.Require().Equal(env.Contract.Address, wasmClientID)

					wrappedClientState, ok := clienttypes.MustUnmarshalClientState(s.chainA.App.AppCodec(), payload.ClientState).(*ibctm.ClientState)
					s.Require().True(ok)

					clientState := types.NewClientState(payload.ClientState, payload.Checksum, wrappedClientState.LatestHeight)
					clientStateBz := clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), clientState)
					store.Set(host.ClientStateKey(), clientStateBz)

					consensusState := types.NewConsensusState(payload.ConsensusState)
					consensusStateBz := clienttypes.MustMarshalConsensusState(s.chainA.App.AppCodec(), consensusState)
					store.Set(host.ConsensusStateKey(clientState.LatestHeight), consensusStateBz)

					resp, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, 0, nil
				}
			},
			nil,
		},
		{
			"failure: cannot unmarshal client state",
			func() {
				clientState = &solomachine.ClientState{Sequence: 20}
			},
			errors.New("proto: wrong wireType = 0 for field Data"),
		},
		{
			"failure: client state is invalid",
			func() {
				clientState = &types.ClientState{}
			},
			types.ErrInvalidData,
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
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()

			wrappedClientStateBz := clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), wasmtesting.MockTendermitClientState)
			wrappedClientConsensusStateBz := clienttypes.MustMarshalConsensusState(s.chainA.App.AppCodec(), wasmtesting.MockTendermintClientConsensusState)
			clientState = types.NewClientState(wrappedClientStateBz, s.checksum, wasmtesting.MockTendermitClientState.LatestHeight)
			consensusState = types.NewConsensusState(wrappedClientConsensusStateBz)

			clientID := s.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(s.chainA.GetContext(), clientState.ClientType())

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			// Marshal client state and consensus state:
			clientStateBz := s.chainA.Codec.MustMarshal(clientState)
			consensusStateBz := s.chainA.Codec.MustMarshal(consensusState)

			err = lightClientModule.Initialize(s.chainA.GetContext(), clientID, clientStateBz, consensusStateBz)

			if tc.expError == nil {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorContains(err, tc.expError.Error())
			}
		})
	}
}

func (s *WasmTestSuite) TestVerifyMembership() {
	var (
		clientState      *types.ClientState
		expClientStateBz []byte
		path             exported.Path
		proof            []byte
		proofHeight      exported.Height
		value            []byte
		clientID         string
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				expClientStateBz = clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), clientState)
				s.mockVM.RegisterSudoCallback(types.VerifyMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					s.Require().NoError(err)

					bz, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"success: with update client state",
			func() {
				s.mockVM.RegisterSudoCallback(types.VerifyMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					s.Require().NoError(err)

					s.Require().NotNil(payload.VerifyMembership)
					s.Require().Nil(payload.UpdateState)
					s.Require().Nil(payload.UpdateStateOnMisbehaviour)
					s.Require().Nil(payload.VerifyNonMembership)
					s.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					s.Require().Equal(proofHeight, payload.VerifyMembership.Height)
					s.Require().Equal(proof, payload.VerifyMembership.Proof)
					s.Require().Equal(path, payload.VerifyMembership.Path)
					s.Require().Equal(value, payload.VerifyMembership.Value)

					bz, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					expClientStateBz = wasmtesting.CreateMockClientStateBz(s.chainA.Codec, s.checksum)
					store.Set(host.ClientStateKey(), expClientStateBz)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"failure: contract returns invalid proof error",
			func() {
				proof = wasmtesting.MockInvalidProofBz

				s.mockVM.RegisterSudoCallback(types.VerifyMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: commitmenttypes.ErrInvalidProof.Error()}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: proof height greater than client state latest height",
			func() {
				proofHeight = clienttypes.NewHeight(1, 100)
			},
			ibcerrors.ErrInvalidHeight,
		},
		{
			"failure: invalid path argument",
			func() {
				path = ibcmock.KeyPath{}
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: proof height is invalid type",
			func() {
				proofHeight = ibcmock.Height{}
			},
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			var ok bool
			s.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)
			clientID = endpoint.ClientID

			path = commitmenttypes.NewMerklePath([]byte("/ibc/key/path"))
			proof = wasmtesting.MockValidProofBz
			proofHeight = clienttypes.NewHeight(0, 1)
			value = []byte("value")

			clientState, ok = endpoint.GetClientState().(*types.ClientState)
			s.Require().True(ok)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			err = lightClientModule.VerifyMembership(s.chainA.GetContext(), clientID, proofHeight, 0, 0, proof, path, value)

			if tc.expError == nil {
				clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), clientID)

				s.Require().NoError(err)
				s.Require().Equal(expClientStateBz, clientStore.Get(host.ClientStateKey()))
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *WasmTestSuite) TestVerifyNonMembership() {
	var (
		clientState      *types.ClientState
		expClientStateBz []byte
		path             exported.Path
		proof            []byte
		proofHeight      exported.Height
		clientID         string
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				expClientStateBz = clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), clientState)
				s.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					s.Require().NoError(err)

					s.Require().NotNil(payload.VerifyNonMembership)
					s.Require().Nil(payload.UpdateState)
					s.Require().Nil(payload.UpdateStateOnMisbehaviour)
					s.Require().Nil(payload.VerifyMembership)
					s.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					s.Require().Equal(proofHeight, payload.VerifyNonMembership.Height)
					s.Require().Equal(proof, payload.VerifyNonMembership.Proof)
					s.Require().Equal(path, payload.VerifyNonMembership.Path)

					bz, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"success: with update client state",
			func() {
				s.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg
					err := json.Unmarshal(sudoMsg, &payload)
					s.Require().NoError(err)

					s.Require().NotNil(payload.VerifyNonMembership)
					s.Require().Nil(payload.UpdateState)
					s.Require().Nil(payload.UpdateStateOnMisbehaviour)
					s.Require().Nil(payload.VerifyMembership)
					s.Require().Nil(payload.VerifyUpgradeAndUpdateState)
					s.Require().Equal(proofHeight, payload.VerifyNonMembership.Height)
					s.Require().Equal(proof, payload.VerifyNonMembership.Proof)
					s.Require().Equal(path, payload.VerifyNonMembership.Path)

					bz, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					expClientStateBz = wasmtesting.CreateMockClientStateBz(s.chainA.Codec, s.checksum)
					store.Set(host.ClientStateKey(), expClientStateBz)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"failure: wasm vm returns error",
			func() {
				proof = wasmtesting.MockInvalidProofBz

				s.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockVM
				})
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns invalid proof error",
			func() {
				proof = wasmtesting.MockInvalidProofBz

				s.mockVM.RegisterSudoCallback(types.VerifyNonMembershipMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore,
					_ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction,
				) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: commitmenttypes.ErrInvalidProof.Error()}, wasmtesting.DefaultGasUsed, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
		{
			"failure: proof height greater than client state latest height",
			func() {
				proofHeight = clienttypes.NewHeight(1, 100)
			},
			ibcerrors.ErrInvalidHeight,
		},
		{
			"failure: invalid path argument",
			func() {
				path = ibcmock.KeyPath{}
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: proof height is invalid type",
			func() {
				proofHeight = ibcmock.Height{}
			},
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			var ok bool
			s.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)
			clientID = endpoint.ClientID

			path = commitmenttypes.NewMerklePath([]byte("/ibc/key/path"))
			proof = wasmtesting.MockInvalidProofBz
			proofHeight = clienttypes.NewHeight(0, 1)

			clientState, ok = endpoint.GetClientState().(*types.ClientState)
			s.Require().True(ok)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			err = lightClientModule.VerifyNonMembership(s.chainA.GetContext(), clientID, proofHeight, 0, 0, proof, path)

			if tc.expError == nil {
				clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), clientID)

				s.Require().NoError(err)
				s.Require().Equal(expClientStateBz, clientStore.Get(host.ClientStateKey()))
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *WasmTestSuite) TestVerifyClientMessage() {
	var (
		clientMsg exported.ClientMessage
		clientID  string
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: valid misbehaviour",
			func() {
				s.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(_ wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					var msg *types.QueryMsg

					err := json.Unmarshal(queryMsg, &msg)
					s.Require().NoError(err)

					s.Require().NotNil(msg.VerifyClientMessage)
					s.Require().NotNil(msg.VerifyClientMessage.ClientMessage)
					s.Require().Nil(msg.Status)
					s.Require().Nil(msg.CheckForMisbehaviour)
					s.Require().Nil(msg.TimestampAtHeight)

					s.Require().Equal(env.Contract.Address, wasmClientID)

					resp, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"failure: invalid client message",
			func() {
				clientMsg = &ibctm.Header{}

				s.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					resp, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: error return from vm",
			func() {
				s.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockVM
				})
			},
			types.ErrVMError,
		},
		{
			"failure: error return from contract",
			func() {
				s.mockVM.RegisterQueryCallback(types.VerifyClientMessageMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, queryMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return &wasmvmtypes.QueryResult{Err: wasmtesting.ErrMockContract.Error()}, 0, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// reset suite to create fresh application state
			s.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)
			clientID = endpoint.ClientID

			clientMsg = &types.ClientMessage{
				Data: clienttypes.MustMarshalClientMessage(s.chainA.App.AppCodec(), wasmtesting.MockTendermintClientHeader),
			}

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			err = lightClientModule.VerifyClientMessage(s.chainA.GetContext(), clientID, clientMsg)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *WasmTestSuite) TestVerifyUpgradeAndUpdateState() {
	var (
		upgradedClient              exported.ClientState
		upgradedConsState           exported.ConsensusState
		upgradedClientProof         []byte
		upgradedConsensusStateProof []byte
		clientID                    string
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: successful upgrade",
			func() {
				s.mockVM.RegisterSudoCallback(types.VerifyUpgradeAndUpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var payload types.SudoMsg

					err := json.Unmarshal(sudoMsg, &payload)
					s.Require().NoError(err)

					expectedUpgradedClient, ok := upgradedClient.(*types.ClientState)
					s.Require().True(ok)
					expectedUpgradedConsensus, ok := upgradedConsState.(*types.ConsensusState)
					s.Require().True(ok)

					// verify payload values
					s.Require().Equal(expectedUpgradedClient.Data, payload.VerifyUpgradeAndUpdateState.UpgradeClientState)
					s.Require().Equal(expectedUpgradedConsensus.Data, payload.VerifyUpgradeAndUpdateState.UpgradeConsensusState)
					s.Require().Equal(upgradedClientProof, payload.VerifyUpgradeAndUpdateState.ProofUpgradeClient)
					s.Require().Equal(upgradedConsensusStateProof, payload.VerifyUpgradeAndUpdateState.ProofUpgradeConsensusState)

					// verify other Sudo fields are nil
					s.Require().Nil(payload.UpdateState)
					s.Require().Nil(payload.UpdateStateOnMisbehaviour)
					s.Require().Nil(payload.VerifyMembership)
					s.Require().Nil(payload.VerifyNonMembership)

					data, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					// set new client state and consensus state
					wrappedUpgradedClient, ok := clienttypes.MustUnmarshalClientState(s.chainA.App.AppCodec(), expectedUpgradedClient.Data).(*ibctm.ClientState)
					s.Require().True(ok)
					store.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), upgradedClient))
					store.Set(host.ConsensusStateKey(wrappedUpgradedClient.LatestHeight), clienttypes.MustMarshalConsensusState(s.chainA.App.AppCodec(), upgradedConsState))

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
		},
		{
			"failure: invalid client state",
			func() {
				upgradedClient = &solomachine.ClientState{Sequence: 20}
			},
			clienttypes.ErrInvalidClient,
		},
		{
			"failure: invalid height",
			func() {
				upgradedClient = &types.ClientState{LatestHeight: clienttypes.ZeroHeight()}
			},
			ibcerrors.ErrInvalidHeight,
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		/* NOTE(jim): This can't fail on unmarshalling, it appears. Any consensus type
					  we attempt to unmarshal just creates a Wasm ConsensusState that has a
					  Data field empty.
		{
			"failure: upgraded consensus state is not wasm consensus state",
			func() {
				// set upgraded consensus state to solomachine consensus state
				upgradedConsState = &solomachine.ConsensusState{}
			},
			clienttypes.ErrInvalidConsensus,
		},
		*/
		{
			"failure: vm returns error",
			func() {
				s.mockVM.RegisterSudoCallback(types.VerifyUpgradeAndUpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockVM
				})
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				s.mockVM.RegisterSudoCallback(types.VerifyUpgradeAndUpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, 0, nil
				})
			},
			types.ErrWasmContractCallFailed,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// reset suite
			s.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)
			clientID = endpoint.ClientID

			clientState, ok := endpoint.GetClientState().(*types.ClientState)
			s.Require().True(ok)

			newLatestHeight := clienttypes.NewHeight(2, 10)
			wrappedUpgradedClient := wasmtesting.CreateMockTendermintClientState(newLatestHeight)
			wrappedUpgradedClientBz := clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), wrappedUpgradedClient)
			upgradedClient = types.NewClientState(wrappedUpgradedClientBz, clientState.Checksum, newLatestHeight)

			wrappedUpgradedConsensus := ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("new-hash")), []byte("new-nextValsHash"))
			wrappedUpgradedConsensusBz := clienttypes.MustMarshalConsensusState(s.chainA.App.AppCodec(), wrappedUpgradedConsensus)
			upgradedConsState = types.NewConsensusState(wrappedUpgradedConsensusBz)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), wasmClientID)

			upgradedClientProof = wasmtesting.MockUpgradedClientStateProofBz
			upgradedConsensusStateProof = wasmtesting.MockUpgradedConsensusStateProofBz

			newClient := s.chainA.Codec.MustMarshal(upgradedClient)
			newConsensusState := s.chainA.Codec.MustMarshal(upgradedConsState)

			err = lightClientModule.VerifyUpgradeAndUpdateState(
				s.chainA.GetContext(),
				clientID,
				newClient,
				newConsensusState,
				upgradedClientProof,
				upgradedConsensusStateProof,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)

				// verify new client state and consensus state
				clientStateBz := clientStore.Get(host.ClientStateKey())
				s.Require().NotEmpty(clientStateBz)
				s.Require().Equal(upgradedClient, clienttypes.MustUnmarshalClientState(s.chainA.Codec, clientStateBz))

				consStateBz := clientStore.Get(host.ConsensusStateKey(upgradedClient.(*types.ClientState).LatestHeight))
				s.Require().NotEmpty(consStateBz)
				s.Require().Equal(upgradedConsState, clienttypes.MustUnmarshalConsensusState(s.chainA.Codec, consStateBz))
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *WasmTestSuite) TestCheckForMisbehaviour() {
	var (
		clientMessage exported.ClientMessage
		clientID      string
	)

	testCases := []struct {
		name              string
		malleate          func()
		foundMisbehaviour bool
		expPanic          error
	}{
		{
			"success: no misbehaviour",
			func() {
				s.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					resp, err := json.Marshal(types.CheckForMisbehaviourResult{FoundMisbehaviour: false})
					s.Require().NoError(err)
					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			false,
			nil,
		},
		{
			"success: misbehaviour found", func() {
				s.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					resp, err := json.Marshal(types.CheckForMisbehaviourResult{FoundMisbehaviour: true})
					s.Require().NoError(err)
					return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			true,
			nil,
		},
		{
			"success: contract error, resp cannot be marshalled", func() {
				s.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					resp := "cannot be unmarshalled"
					return &wasmvmtypes.QueryResult{Ok: []byte(resp)}, wasmtesting.DefaultGasUsed, nil
				})
			},
			false,
			nil,
		},
		{
			"success: contract returns error", func() {
				s.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return &wasmvmtypes.QueryResult{Err: wasmtesting.ErrMockContract.Error()}, wasmtesting.DefaultGasUsed, nil
				})
			},
			false,
			nil,
		},
		{
			"success: vm returns error, ", func() {
				s.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return nil, 0, errors.New("invalid block ID")
				})
			},
			false,
			nil,
		},
		{
			"success: invalid client message", func() {
				clientMessage = &ibctm.Header{}
				// we will not register the callback here because this test case does not reach the VM
			},
			false,
			nil,
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedWasmClientID
			},
			false, // not applicable
			fmt.Errorf("%s: %w", unusedWasmClientID, clienttypes.ErrClientNotFound),
		},
		{
			"failure: response fails to unmarshal",
			func() {
				s.mockVM.RegisterQueryCallback(types.CheckForMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
					return &wasmvmtypes.QueryResult{Ok: []byte("invalid json")}, wasmtesting.DefaultGasUsed, nil
				})
			},
			false,
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// reset suite to create fresh application state
			s.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)
			clientID = endpoint.ClientID

			clientMessage = &types.ClientMessage{
				Data: clienttypes.MustMarshalClientMessage(s.chainA.App.AppCodec(), wasmtesting.MockTendermintClientMisbehaviour),
			}

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			var foundMisbehaviour bool
			foundMisbehaviourFunc := func() {
				foundMisbehaviour = lightClientModule.CheckForMisbehaviour(s.chainA.GetContext(), clientID, clientMessage)
			}

			if tc.expPanic == nil {
				foundMisbehaviourFunc()
				s.Require().Equal(tc.foundMisbehaviour, foundMisbehaviour)
			} else {
				s.Require().PanicsWithError(tc.expPanic.Error(), foundMisbehaviourFunc)
			}
		})
	}
}

func (s *WasmTestSuite) TestUpdateState() {
	mockHeight := clienttypes.NewHeight(1, 50)

	var (
		clientMsg             exported.ClientMessage
		expectedClientStateBz []byte
		clientID              string
	)

	testCases := []struct {
		name       string
		malleate   func()
		expPanic   error
		expHeights []exported.Height
	}{
		{
			"success: no update",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, env wasmvmtypes.Env, sudoMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var msg types.SudoMsg
					err := json.Unmarshal(sudoMsg, &msg)
					s.Require().NoError(err)

					s.Require().NotNil(msg.UpdateState)
					s.Require().NotNil(msg.UpdateState.ClientMessage)
					s.Require().Equal(msg.UpdateState.ClientMessage, clienttypes.MustMarshalClientMessage(s.chainA.App.AppCodec(), wasmtesting.MockTendermintClientHeader))
					s.Require().Nil(msg.VerifyMembership)
					s.Require().Nil(msg.VerifyNonMembership)
					s.Require().Nil(msg.UpdateStateOnMisbehaviour)
					s.Require().Nil(msg.VerifyUpgradeAndUpdateState)

					s.Require().Equal(env.Contract.Address, wasmClientID)

					updateStateResp := types.UpdateStateResult{
						Heights: []clienttypes.Height{},
					}

					resp, err := json.Marshal(updateStateResp)
					if err != nil {
						return nil, 0, err
					}

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			[]exported.Height{},
		},
		{
			"success: update client",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var msg types.SudoMsg
					err := json.Unmarshal(sudoMsg, &msg)
					s.Require().NoError(err)

					bz := store.Get(host.ClientStateKey())
					s.Require().NotEmpty(bz)
					clientState, ok := clienttypes.MustUnmarshalClientState(s.chainA.Codec, bz).(*types.ClientState)
					s.Require().True(ok)
					clientState.LatestHeight = mockHeight
					expectedClientStateBz = clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), clientState)
					store.Set(host.ClientStateKey(), expectedClientStateBz)

					updateStateResp := types.UpdateStateResult{
						Heights: []clienttypes.Height{mockHeight},
					}

					resp, err := json.Marshal(updateStateResp)
					if err != nil {
						return nil, 0, err
					}

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			[]exported.Height{mockHeight},
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedWasmClientID
			},
			fmt.Errorf("08-wasm-100: %w", clienttypes.ErrClientNotFound),
			nil,
		},
		{
			"failure: invalid ClientMessage type",
			func() {
				// SudoCallback left nil because clientMsg is checked by 08-wasm before callbackFn is called.
				clientMsg = &ibctm.Misbehaviour{}
			},
			fmt.Errorf("expected type %T, got %T", (*types.ClientMessage)(nil), (*ibctm.Misbehaviour)(nil)),
			nil,
		},
		{
			"failure: VM returns error",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockVM
				})
			},
			errorsmod.Wrap(types.ErrVMError, wasmtesting.ErrMockVM.Error()),
			nil,
		},
		{
			"failure: response fails to unmarshal",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: []byte("invalid json")}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			fmt.Errorf("invalid character 'i' looking for beginning of value: %w", types.ErrWasmInvalidResponseData),
			nil,
		},
		{
			"failure: callbackFn returns error",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, 0, nil
				})
			},
			errorsmod.Wrap(types.ErrWasmContractCallFailed, wasmtesting.ErrMockContract.Error()),
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM() // reset

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)
			clientID = endpoint.ClientID

			expectedClientStateBz = nil

			clientMsg = &types.ClientMessage{
				Data: clienttypes.MustMarshalClientMessage(s.chainA.App.AppCodec(), wasmtesting.MockTendermintClientHeader),
			}

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			var heights []exported.Height
			updateState := func() {
				heights = lightClientModule.UpdateState(s.chainA.GetContext(), clientID, clientMsg)
			}

			if tc.expPanic == nil {
				updateState()
				s.Require().Equal(tc.expHeights, heights)

				if expectedClientStateBz != nil {
					clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), endpoint.ClientID)

					clientStateBz := clientStore.Get(host.ClientStateKey())
					s.Require().Equal(expectedClientStateBz, clientStateBz)
				}
			} else {
				s.Require().PanicsWithError(tc.expPanic.Error(), updateState)
			}
		})
	}
}

func (s *WasmTestSuite) TestUpdateStateOnMisbehaviour() {
	mockHeight := clienttypes.NewHeight(1, 50)

	var (
		clientMsg             exported.ClientMessage
		expectedClientStateBz []byte
		clientID              string
	)

	testCases := []struct {
		name               string
		malleate           func()
		panicErr           error
		updatedClientState []byte
	}{
		{
			"success: no update",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateOnMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var msg types.SudoMsg

					err := json.Unmarshal(sudoMsg, &msg)
					s.Require().NoError(err)

					s.Require().NotNil(msg.UpdateStateOnMisbehaviour)
					s.Require().NotNil(msg.UpdateStateOnMisbehaviour.ClientMessage)
					s.Require().Nil(msg.UpdateState)
					s.Require().Nil(msg.UpdateState)
					s.Require().Nil(msg.VerifyMembership)
					s.Require().Nil(msg.VerifyNonMembership)
					s.Require().Nil(msg.VerifyUpgradeAndUpdateState)

					resp, err := json.Marshal(types.EmptyResult{})
					if err != nil {
						return nil, 0, err
					}

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			nil,
		},
		{
			"success: client state updated on valid misbehaviour",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateOnMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					var msg types.SudoMsg
					err := json.Unmarshal(sudoMsg, &msg)
					s.Require().NoError(err)

					// set new client state in store
					bz := store.Get(host.ClientStateKey())
					s.Require().NotEmpty(bz)
					clientState, ok := clienttypes.MustUnmarshalClientState(s.chainA.App.AppCodec(), bz).(*types.ClientState)
					s.Require().True(ok)
					clientState.LatestHeight = mockHeight
					expectedClientStateBz = clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), clientState)
					store.Set(host.ClientStateKey(), expectedClientStateBz)

					resp, err := json.Marshal(types.EmptyResult{})
					if err != nil {
						return nil, 0, err
					}

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), wasmtesting.CreateMockTendermintClientState(mockHeight)),
		},
		{
			"failure: cannot find client state",
			func() {
				clientID = unusedWasmClientID
			},
			fmt.Errorf("%s: %w", unusedWasmClientID, clienttypes.ErrClientNotFound),
			nil,
		},
		{
			"failure: invalid client message",
			func() {
				clientMsg = &ibctm.Header{}
				// we will not register the callback here because this test case does not reach the VM
			},
			fmt.Errorf("expected type %T, got %T", (*types.ClientMessage)(nil), (*ibctm.Header)(nil)),
			nil,
		},
		{
			"failure: err return from vm",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateOnMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, 0, wasmtesting.ErrMockVM
				})
			},
			errorsmod.Wrap(types.ErrVMError, wasmtesting.ErrMockVM.Error()),
			nil,
		},
		{
			"failure: err return from contract",
			func() {
				s.mockVM.RegisterSudoCallback(types.UpdateStateOnMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, 0, nil
				})
			},
			errorsmod.Wrap(types.ErrWasmContractCallFailed, wasmtesting.ErrMockContract.Error()),
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// reset suite to create fresh application state
			s.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := endpoint.CreateClient()
			s.Require().NoError(err)
			clientID = endpoint.ClientID

			expectedClientStateBz = nil

			clientMsg = &types.ClientMessage{
				Data: clienttypes.MustMarshalClientMessage(s.chainA.App.AppCodec(), wasmtesting.MockTendermintClientMisbehaviour),
			}

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			updateFunc := func() {
				lightClientModule.UpdateStateOnMisbehaviour(s.chainA.GetContext(), clientID, clientMsg)
			}

			if tc.panicErr == nil {
				updateFunc()
				if expectedClientStateBz != nil {
					store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), endpoint.ClientID)
					s.Require().Equal(expectedClientStateBz, store.Get(host.ClientStateKey()))
				}
			} else {
				s.Require().PanicsWithError(tc.panicErr.Error(), updateFunc)
			}
		})
	}
}

func (s *WasmTestSuite) TestRecoverClient() {
	var (
		expectedClientStateBz               []byte
		subjectClientID, substituteClientID string
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				s.mockVM.RegisterSudoCallback(
					types.MigrateClientStoreMsg{},
					func(_ wasmvm.Checksum, _ wasmvmtypes.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
						var payload types.SudoMsg
						err := json.Unmarshal(sudoMsg, &payload)
						s.Require().NoError(err)

						s.Require().NotNil(payload.MigrateClientStore)
						s.Require().Nil(payload.UpdateState)
						s.Require().Nil(payload.UpdateStateOnMisbehaviour)
						s.Require().Nil(payload.VerifyMembership)
						s.Require().Nil(payload.VerifyNonMembership)
						s.Require().Nil(payload.VerifyUpgradeAndUpdateState)

						bz, err := json.Marshal(types.EmptyResult{})
						s.Require().NoError(err)

						prefixedKey := internaltypes.SubjectPrefix
						prefixedKey = append(prefixedKey, host.ClientStateKey()...)
						expectedClientStateBz = wasmtesting.CreateMockClientStateBz(s.chainA.Codec, s.checksum)
						store.Set(prefixedKey, expectedClientStateBz)

						return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: bz}}, wasmtesting.DefaultGasUsed, nil
					},
				)
			},
			nil,
		},
		{
			"failure: cannot parse malformed substitute client ID",
			func() {
				substituteClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"failure: substitute client ID does not contain 08-wasm prefix",
			func() {
				substituteClientID = tmClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"failure: cannot find subject client state",
			func() {
				subjectClientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"failure: cannot find substitute client state",
			func() {
				substituteClientID = unusedWasmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"failure: checksums do not match",
			func() {
				substituteClientState, found := GetSimApp(s.chainA).IBCKeeper.ClientKeeper.GetClientState(s.chainA.GetContext(), substituteClientID)
				s.Require().True(found)

				wasmSubstituteClientState, ok := substituteClientState.(*types.ClientState)
				s.Require().True(ok)

				wasmSubstituteClientState.Checksum = []byte("invalid")
				GetSimApp(s.chainA).IBCKeeper.ClientKeeper.SetClientState(s.chainA.GetContext(), substituteClientID, wasmSubstituteClientState)
			},
			clienttypes.ErrInvalidClient,
		},
		{
			"failure: vm returns error",
			func() {
				s.mockVM.RegisterSudoCallback(
					types.MigrateClientStoreMsg{},
					func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
						return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockVM
					},
				)
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				s.mockVM.RegisterSudoCallback(
					types.MigrateClientStoreMsg{},
					func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
						return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, wasmtesting.DefaultGasUsed, nil
					},
				)
			},
			types.ErrWasmContractCallFailed,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()

			subjectEndpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := subjectEndpoint.CreateClient()
			s.Require().NoError(err)
			subjectClientID = subjectEndpoint.ClientID

			substituteEndpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err = substituteEndpoint.CreateClient()
			s.Require().NoError(err)
			substituteClientID = substituteEndpoint.ClientID

			expectedClientStateBz = nil

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), subjectClientID)
			s.Require().NoError(err)

			tc.malleate()

			err = lightClientModule.RecoverClient(s.chainA.GetContext(), subjectClientID, substituteClientID)

			if tc.expErr == nil {
				s.Require().NoError(err)

				subjectClientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), subjectClientID)
				s.Require().Equal(expectedClientStateBz, subjectClientStore.Get(host.ClientStateKey()))
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *WasmTestSuite) TestLatestHeight() {
	var clientID string

	testCases := []struct {
		name      string
		malleate  func()
		expHeight clienttypes.Height
	}{
		{
			"success",
			func() {
			},
			clienttypes.NewHeight(1, 5),
		},
		{
			"failure: cannot find substitute client state",
			func() {
				clientID = unusedWasmClientID
			},
			clienttypes.ZeroHeight(),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()

			subjectEndpoint := wasmtesting.NewWasmEndpoint(s.chainA)
			err := subjectEndpoint.CreateClient()
			s.Require().NoError(err)
			clientID = subjectEndpoint.ClientID

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			height := lightClientModule.LatestHeight(s.chainA.GetContext(), clientID)

			s.Require().Equal(tc.expHeight, height)
		})
	}
}
