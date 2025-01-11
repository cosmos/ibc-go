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
	govtypes "cosmossdk.io/x/gov/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing/simapp"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

type WasmTestSuite struct {
	testifysuite.Suite
	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	mockVM      *wasmtesting.MockWasmEngine

	checksum types.Checksum
}

func TestWasmTestSuite(t *testing.T) {
	testifysuite.Run(t, new(WasmTestSuite))
}

func (suite *WasmTestSuite) SetupTest() {
	ibctesting.DefaultTestingAppInit = setupTestingApp

	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
}

func init() {
	ibctesting.DefaultTestingAppInit = setupTestingApp
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
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.AppOptionsMap{}, nil)
	return app, app.DefaultGenesis()
}

// SetupWasmWithMockVM sets up mock cometbft chain with a mock vm.
func (suite *WasmTestSuite) SetupWasmWithMockVM() {
	ibctesting.DefaultTestingAppInit = suite.setupWasmWithMockVM

	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.checksum = storeWasmCode(suite, wasmtesting.Code)
}

func (suite *WasmTestSuite) setupWasmWithMockVM() (ibctesting.TestingApp, map[string]json.RawMessage) {
	suite.mockVM = wasmtesting.NewMockWasmEngine()

	suite.mockVM.InstantiateFn = func(checksum wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
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

		resp, err := json.Marshal(types.EmptyResult{})
		suite.Require().NoError(err)

		return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{Data: resp}}, 0, nil
	}

	suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(checksum wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.QueryResult, uint64, error) {
		resp, err := json.Marshal(types.StatusResult{Status: exported.Active.String()})
		suite.Require().NoError(err)
		return &wasmvmtypes.QueryResult{Ok: resp}, wasmtesting.DefaultGasUsed, nil
	})

	db := dbm.NewMemDB()
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.AppOptionsMap{}, suite.mockVM)

	// reset DefaultTestingAppInit to its original value
	ibctesting.DefaultTestingAppInit = setupTestingApp
	return app, app.DefaultGenesis()
}

// storeWasmCode stores the wasm code on chain and returns the checksum.
func storeWasmCode(suite *WasmTestSuite, wasmCode []byte) types.Checksum {
	ctx := suite.chainA.GetContext().WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	msg := types.NewMsgStoreCode(authtypes.NewModuleAddress(govtypes.ModuleName).String(), wasmCode)
	response, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response.Checksum)
	return response.Checksum
}
