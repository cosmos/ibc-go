package keeper_test

import (
	"encoding/json"
	"errors"
	"testing"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	dbm "github.com/cosmos/cosmos-db"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/keeper"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing/simapp"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const (
	defaultWasmClientID = "08-wasm-0"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// mockVM is a mock wasm VM that implements the WasmEngine interface
	mockVM *wasmtesting.MockWasmEngine
	chainA *ibctesting.TestChain
}

// setupTestingApp provides the duplicated simapp which is specific to the 08-wasm module on chain creation.
func setupTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	app := simapp.NewUnitTestSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{}, nil)
	return app, app.DefaultGenesis()
}

// GetSimApp returns the duplicated SimApp from within the 08-wasm directory.
// This must be used instead of chain.GetSimApp() for tests within this directory.
func GetSimApp(chain *ibctesting.TestChain) *simapp.SimApp {
	app, ok := chain.App.(*simapp.SimApp)
	if !ok {
		panic(errors.New("chain is not a simapp.SimApp"))
	}
	return app
}

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCustomAppCoordinator(s.T(), 1, setupTestingApp)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))

	queryHelper := baseapp.NewQueryServerTestHelper(s.chainA.GetContext(), GetSimApp(s.chainA).InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, &GetSimApp(s.chainA).WasmClientKeeper)
}

// SetupWasmWithMockVM sets up mock cometbft chain with a mock vm.
func (s *KeeperTestSuite) SetupWasmWithMockVM() {
	s.coordinator = ibctesting.NewCustomAppCoordinator(s.T(), 1, s.setupWasmWithMockVM)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
}

func (s *KeeperTestSuite) setupWasmWithMockVM() (ibctesting.TestingApp, map[string]json.RawMessage) {
	s.mockVM = wasmtesting.NewMockWasmEngine()

	s.mockVM.InstantiateFn = func(checksum wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
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

		resp, err := json.Marshal(types.EmptyResult{})
		s.Require().NoError(err)

		return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, 0, nil
	}

	s.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
		resp, err := json.Marshal(types.StatusResult{Status: exported.Active.String()})
		s.Require().NoError(err)
		return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
	})

	db := dbm.NewMemDB()
	app := simapp.NewUnitTestSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{}, s.mockVM)

	return app, app.DefaultGenesis()
}

// storeWasmCode stores the wasm code on chain and returns the checksum.
func (s *KeeperTestSuite) storeWasmCode(wasmCode []byte) []byte {
	ctx := s.chainA.GetContext().WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	msg := types.NewMsgStoreCode(authtypes.NewModuleAddress(govtypes.ModuleName).String(), wasmCode)
	response, err := GetSimApp(s.chainA).WasmClientKeeper.StoreCode(ctx, msg)
	s.Require().NoError(err)
	s.Require().NotNil(response.Checksum)
	return response.Checksum
}

func (s *KeeperTestSuite) SetupSnapshotterWithMockVM() *simapp.SimApp {
	s.mockVM = wasmtesting.NewMockWasmEngine()

	return simapp.SetupWithSnapshotter(s.T(), s.mockVM)
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestNewKeeper() {
	testCases := []struct {
		name          string
		instantiateFn func()
		expError      error
	}{
		{
			"success",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(s.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(s.chainA).GetKey(types.StoreKey)),
					GetSimApp(s.chainA).IBCKeeper.ClientKeeper,
					GetSimApp(s.chainA).WasmClientKeeper.GetAuthority(),
					GetSimApp(s.chainA).WasmClientKeeper.GetVM(),
					GetSimApp(s.chainA).GRPCQueryRouter(),
				)
			},
			nil,
		},
		{
			"failure: empty authority",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(s.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(s.chainA).GetKey(types.StoreKey)),
					GetSimApp(s.chainA).IBCKeeper.ClientKeeper,
					"", // authority
					GetSimApp(s.chainA).WasmClientKeeper.GetVM(),
					GetSimApp(s.chainA).GRPCQueryRouter(),
				)
			},
			errors.New("authority must be non-empty"),
		},
		{
			"failure: nil client keeper",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(s.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(s.chainA).GetKey(types.StoreKey)),
					nil, // client keeper,
					GetSimApp(s.chainA).WasmClientKeeper.GetAuthority(),
					GetSimApp(s.chainA).WasmClientKeeper.GetVM(),
					GetSimApp(s.chainA).GRPCQueryRouter(),
				)
			},
			errors.New("client keeper must not be nil"),
		},
		{
			"failure: nil wasm VM",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(s.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(s.chainA).GetKey(types.StoreKey)),
					GetSimApp(s.chainA).IBCKeeper.ClientKeeper,
					GetSimApp(s.chainA).WasmClientKeeper.GetAuthority(),
					nil,
					GetSimApp(s.chainA).GRPCQueryRouter(),
				)
			},
			errors.New("wasm VM must not be nil"),
		},
		{
			"failure: nil store service",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(s.chainA).AppCodec(),
					nil,
					GetSimApp(s.chainA).IBCKeeper.ClientKeeper,
					GetSimApp(s.chainA).WasmClientKeeper.GetAuthority(),
					GetSimApp(s.chainA).WasmClientKeeper.GetVM(),
					GetSimApp(s.chainA).GRPCQueryRouter(),
				)
			},
			errors.New("store service must not be nil"),
		},
		{
			"failure: nil query router",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(s.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(s.chainA).GetKey(types.StoreKey)),
					GetSimApp(s.chainA).IBCKeeper.ClientKeeper,
					GetSimApp(s.chainA).WasmClientKeeper.GetAuthority(),
					GetSimApp(s.chainA).WasmClientKeeper.GetVM(),
					nil,
				)
			},
			errors.New("query router must not be nil"),
		},
	}

	for _, tc := range testCases {
		s.SetupTest()

		s.Run(tc.name, func() {
			if tc.expError == nil {
				s.Require().NotPanics(
					tc.instantiateFn,
				)
			} else {
				s.Require().PanicsWithError(tc.expError.Error(), func() {
					tc.instantiateFn()
				})
			}
		})
	}
}

func (s *KeeperTestSuite) TestInitializedPinnedCodes() {
	var capturedChecksums []wasmvm.Checksum

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				s.mockVM.PinFn = func(checksum wasmvm.Checksum) error {
					capturedChecksums = append(capturedChecksums, checksum)
					return nil
				}
			},
			nil,
		},
		{
			"failure: pin error",
			func() {
				s.mockVM.PinFn = func(checksum wasmvm.Checksum) error {
					return wasmtesting.ErrMockVM
				}
			},
			wasmtesting.ErrMockVM,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()

			ctx := s.chainA.GetContext()
			wasmClientKeeper := GetSimApp(s.chainA).WasmClientKeeper

			contracts := [][]byte{wasmtesting.Code, wasmtesting.CreateMockContract([]byte("gzipped-contract"))}
			checksumIDs := make([]types.Checksum, len(contracts))
			signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()

			// store contract on chain
			for i, contract := range contracts {
				msg := types.NewMsgStoreCode(signer, contract)

				res, err := wasmClientKeeper.StoreCode(ctx, msg)
				s.Require().NoError(err)

				checksumIDs[i] = res.Checksum
			}

			// malleate after storing contracts
			tc.malleate()

			err := wasmClientKeeper.InitializePinnedCodes(ctx)

			if tc.expError == nil {
				s.Require().NoError(err)
				s.ElementsMatch(checksumIDs, capturedChecksums)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMigrateContract() {
	var (
		oldHash        []byte
		newHash        []byte
		payload        []byte
		expClientState *types.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: update client state",
			func() {
				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					expClientState = types.NewClientState([]byte{1}, newHash, clienttypes.NewHeight(2000, 2))
					store.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(s.chainA.App.AppCodec(), expClientState))

					data, err := json.Marshal(types.EmptyResult{})
					s.Require().NoError(err)

					return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: data}}, wasmtesting.DefaultGasUsed, nil
				}
			},
			nil,
		},
		{
			"failure: new and old checksum are the same",
			func() {
				newHash = oldHash
				// this should not be called
				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					panic("unreachable")
				}
			},
			types.ErrWasmCodeExists,
		},
		{
			"failure: checksum not found",
			func() {
				err := GetSimApp(s.chainA).WasmClientKeeper.GetChecksums().Remove(s.chainA.GetContext(), newHash)
				s.Require().NoError(err)
			},
			types.ErrWasmChecksumNotFound,
		},
		{
			"failure: vm returns error",
			func() {
				err := GetSimApp(s.chainA).WasmClientKeeper.GetChecksums().Set(s.chainA.GetContext(), newHash)
				s.Require().NoError(err)

				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return nil, wasmtesting.DefaultGasUsed, wasmtesting.ErrMockVM
				}
			},
			types.ErrVMError,
		},
		{
			"failure: contract returns error",
			func() {
				err := GetSimApp(s.chainA).WasmClientKeeper.GetChecksums().Set(s.chainA.GetContext(), newHash)
				s.Require().NoError(err)

				s.mockVM.MigrateFn = func(_ wasmvm.Checksum, _ wasmvmtypes.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
					return &wasmvmtypes.ContractResult{Err: wasmtesting.ErrMockContract.Error()}, wasmtesting.DefaultGasUsed, nil
				}
			},
			types.ErrWasmContractCallFailed,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()
			s.storeWasmCode(wasmtesting.Code)

			var err error
			oldHash, err = types.CreateChecksum(wasmtesting.Code)
			s.Require().NoError(err)
			newHash, err = types.CreateChecksum(wasmtesting.CreateMockContract([]byte{1, 2, 3}))
			s.Require().NoError(err)

			wasmClientKeeper := GetSimApp(s.chainA).WasmClientKeeper
			err = wasmClientKeeper.GetChecksums().Set(s.chainA.GetContext(), newHash)
			s.Require().NoError(err)

			endpointA := wasmtesting.NewWasmEndpoint(s.chainA)
			err = endpointA.CreateClient()
			s.Require().NoError(err)

			clientState, ok := endpointA.GetClientState().(*types.ClientState)
			s.Require().True(ok)

			expClientState = clientState

			tc.malleate()

			err = wasmClientKeeper.MigrateContractCode(s.chainA.GetContext(), endpointA.ClientID, newHash, payload)

			// updated client state
			clientState, ok = endpointA.GetClientState().(*types.ClientState)
			s.Require().True(ok)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().Equal(expClientState, clientState)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetChecksums() {
	testCases := []struct {
		name      string
		malleate  func()
		expResult func(checksums []types.Checksum)
	}{
		{
			"success: no contract stored.",
			func() {},
			func(checksums []types.Checksum) {
				s.Require().Empty(checksums)
			},
		},
		{
			"success: default mock vm contract stored.",
			func() {
				s.SetupWasmWithMockVM()
				s.storeWasmCode(wasmtesting.Code)
			},
			func(checksums []types.Checksum) {
				s.Require().Len(checksums, 1)
				expectedChecksum, err := types.CreateChecksum(wasmtesting.Code)
				s.Require().NoError(err)
				s.Require().Equal(expectedChecksum, checksums[0])
			},
		},
		{
			"success: non-empty checksums",
			func() {
				s.SetupWasmWithMockVM()
				s.storeWasmCode(wasmtesting.Code)

				err := GetSimApp(s.chainA).WasmClientKeeper.GetChecksums().Set(s.chainA.GetContext(), types.Checksum("checksum"))
				s.Require().NoError(err)
			},
			func(checksums []types.Checksum) {
				s.Require().Len(checksums, 2)
				s.Require().Contains(checksums, types.Checksum("checksum"))
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			tc.malleate()

			checksums, err := GetSimApp(s.chainA).WasmClientKeeper.GetAllChecksums(s.chainA.GetContext())
			s.Require().NoError(err)
			tc.expResult(checksums)
		})
	}
}

func (s *KeeperTestSuite) TestAddChecksum() {
	s.SetupWasmWithMockVM()
	s.storeWasmCode(wasmtesting.Code)

	wasmClientKeeper := GetSimApp(s.chainA).WasmClientKeeper

	checksums, err := wasmClientKeeper.GetAllChecksums(s.chainA.GetContext())
	s.Require().NoError(err)
	// default mock vm contract is stored
	s.Require().Len(checksums, 1)

	checksum1 := types.Checksum("checksum1")
	checksum2 := types.Checksum("checksum2")
	err = wasmClientKeeper.GetChecksums().Set(s.chainA.GetContext(), checksum1)
	s.Require().NoError(err)
	err = wasmClientKeeper.GetChecksums().Set(s.chainA.GetContext(), checksum2)
	s.Require().NoError(err)

	// Test adding the same checksum twice
	err = wasmClientKeeper.GetChecksums().Set(s.chainA.GetContext(), checksum1)
	s.Require().NoError(err)

	checksums, err = wasmClientKeeper.GetAllChecksums(s.chainA.GetContext())
	s.Require().NoError(err)
	s.Require().Len(checksums, 3)
	s.Require().Contains(checksums, checksum1)
	s.Require().Contains(checksums, checksum2)
}

func (s *KeeperTestSuite) TestHasChecksum() {
	var checksum types.Checksum

	testCases := []struct {
		name       string
		malleate   func()
		exprResult bool
	}{
		{
			"success: checksum exists",
			func() {
				checksum = types.Checksum("checksum")
				err := GetSimApp(s.chainA).WasmClientKeeper.GetChecksums().Set(s.chainA.GetContext(), checksum)
				s.Require().NoError(err)
			},
			true,
		},
		{
			"success: checksum does not exist",
			func() {
				checksum = types.Checksum("non-existent-checksum")
			},
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupWasmWithMockVM()

			tc.malleate()

			result := GetSimApp(s.chainA).WasmClientKeeper.HasChecksum(s.chainA.GetContext(), checksum)
			s.Require().Equal(tc.exprResult, result)
		})
	}
}
