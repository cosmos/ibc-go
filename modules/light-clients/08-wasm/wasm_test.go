package wasm_test

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

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing/simapp"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type WasmTestSuite struct {
	testifysuite.Suite
	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	mockVM      *wasmtesting.MockWasmEngine

	checksum types.Checksum
}

func TestWasmTests(t *testing.T) {
	testifysuite.Run(t, new(WasmTestSuite))
}

func (s *WasmTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCustomAppCoordinator(s.T(), 1, setupTestingApp)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
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

// setupTestingApp provides the duplicated simapp which is specific to the 08-wasm module on chain creation.
func setupTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	app := simapp.NewUnitTestSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{}, nil)
	return app, app.DefaultGenesis()
}

// SetupWasmWithMockVM sets up mock cometbft chain with a mock vm.
func (s *WasmTestSuite) SetupWasmWithMockVM() {
	s.coordinator = ibctesting.NewCustomAppCoordinator(s.T(), 1, s.setupWasmWithMockVM)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.checksum = storeWasmCode(s, wasmtesting.Code)
}

func (s *WasmTestSuite) setupWasmWithMockVM() (ibctesting.TestingApp, map[string]json.RawMessage) {
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
func storeWasmCode(s *WasmTestSuite, wasmCode []byte) types.Checksum {
	ctx := s.chainA.GetContext().WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	msg := types.NewMsgStoreCode(authtypes.NewModuleAddress(govtypes.ModuleName).String(), wasmCode)
	response, err := GetSimApp(s.chainA).WasmClientKeeper.StoreCode(ctx, msg)
	s.Require().NoError(err)
	s.Require().NotNil(response.Checksum)
	return response.Checksum
}
