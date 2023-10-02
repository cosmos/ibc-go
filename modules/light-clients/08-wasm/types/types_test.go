package types_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	testifysuite "github.com/stretchr/testify/suite"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	dbm "github.com/cometbft/cometbft-db"
	tmjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/libs/log"
	tmtypes "github.com/cometbft/cometbft/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	simapp "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing/simapp"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

// TODO
// contractResult is the default implementation of the ContractResult interface and the default return type of any contract call
// that does not require a custom return type.
type contractResult struct {
	IsValid  bool   `json:"is_valid,omitempty"`
	ErrorMsg string `json:"error_msg,omitempty"`
	Data     []byte `json:"data,omitempty"`
}

// TODO
type mockStatusResult struct {
	contractResult
	Status exported.Status `json:"status"`
}

const (
	tmClientID                    = "07-tendermint-0"
	grandpaClientID               = "08-wasm-0"
	codeHash                      = "01234567012345670123456701234567"
	trustingPeriod  time.Duration = time.Hour * 24 * 7 * 2
	ubdPeriod       time.Duration = time.Hour * 24 * 7 * 3
	maxClockDrift   time.Duration = time.Second * 10
)

var (
	height          = clienttypes.NewHeight(0, 4)
	newClientHeight = clienttypes.NewHeight(1, 1)
	upgradePath     = []string{"upgrade", "upgradedIBCState"}
)

type TypesTestSuite struct {
	testifysuite.Suite
	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
	mockVM      *wasmtesting.MockWasmEngine

	ctx      sdk.Context
	store    sdk.KVStore
	codeHash []byte
	testData map[string]string
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
	encCdc := simapp.MakeTestEncodingConfig()
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{}, nil)
	return app, simapp.NewDefaultGenesisState(encCdc.Codec)
}

// SetupWasmTendermint sets up 2 chains and stores the tendermint/cometbft light client wasm contract on both.
func (suite *TypesTestSuite) SetupWasmWithMockVM() {
	ibctesting.DefaultTestingAppInit = suite.setupWasmWithMockVM

	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainA.SetWasm(true)
	suite.coordinator.SetCodeHash(suite.codeHash)
}

func (suite *TypesTestSuite) setupWasmWithMockVM() (ibctesting.TestingApp, map[string]json.RawMessage) {
	suite.mockVM = &wasmtesting.MockWasmEngine{}
	// TODO: need a default instantiate function has clients need to be created for testing
	suite.mockVM.InstantiateFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
		// TODO
		var payload instantiateMessage
		err := json.Unmarsha(initMsg, &payload)
		suite.Require().NoError(err)
		// TODO:
		// - set client state in store
		// - set consensus state in store
		return nil, 0, nil
	}
	suite.mockVM.QueryFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
		resp, err := json.Marshal(&mockStatusResult{
			Status: exported.Active,
		})
		suite.Require().NoError(err)
		gasUsed := uint64(10) // TODO
		return resp, gasUsed, nil
	}

	db := dbm.NewMemDB()
	encCdc := simapp.MakeTestEncodingConfig()
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{}, suite.mockVM)
	return app, simapp.NewDefaultGenesisState(encCdc.Codec)
}

// SetupWasmTendermint sets up 2 chains and stores the tendermint/cometbft light client wasm contract on both.
func (suite *TypesTestSuite) SetupWasmTendermint() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainA.SetWasm(true)
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainB.SetWasm(true)

	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)

	suite.ctx = suite.chainA.GetContext().WithBlockGasMeter(sdk.NewInfiniteGasMeter())
	suite.store = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, grandpaClientID)

	wasmContract, err := os.ReadFile("../test_data/ics07_tendermint_cw.wasm.gz")
	suite.Require().NoError(err)

	msg := types.NewMsgStoreCode(authtypes.NewModuleAddress(govtypes.ModuleName).String(), wasmContract)
	response, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(suite.chainA.GetContext(), msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response.Checksum)
	suite.codeHash = response.Checksum

	response, err = GetSimApp(suite.chainB).WasmClientKeeper.StoreCode(suite.chainB.GetContext(), msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response.Checksum)
	suite.codeHash = response.Checksum

	suite.coordinator.SetCodeHash(suite.codeHash)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
}

// SetupWasmGrandpa sets up 1 chain and stores the grandpa light client wasm contract on chain.
func (suite *TypesTestSuite) SetupWasmGrandpa() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)

	testData, err := os.ReadFile("../test_data/data.json")
	suite.Require().NoError(err)
	err = json.Unmarshal(testData, &suite.testData)
	suite.Require().NoError(err)

	suite.ctx = suite.chainA.GetContext().WithBlockGasMeter(sdk.NewInfiniteGasMeter())
	suite.store = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, grandpaClientID)

	wasmContract, err := os.ReadFile("../test_data/ics10_grandpa_cw.wasm.gz")
	suite.Require().NoError(err)

	msg := types.NewMsgStoreCode(authtypes.NewModuleAddress(govtypes.ModuleName).String(), wasmContract)
	response, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response.Checksum)
	suite.codeHash = response.Checksum
}

func SetupTestingWithChannel() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	encCdc := simapp.MakeTestEncodingConfig()
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{}, nil)
	genesisState := simapp.NewDefaultGenesisState(encCdc.Codec)

	bytes, err := os.ReadFile("../test_data/genesis.json")
	if err != nil {
		panic(err)
	}

	var genesis tmtypes.GenesisDoc
	// NOTE: Tendermint uses a custom JSON decoder for GenesisDoc
	err = tmjson.Unmarshal(bytes, &genesis)
	if err != nil {
		panic(err)
	}

	var appState map[string]json.RawMessage
	err = json.Unmarshal(genesis.AppState, &appState)
	if err != nil {
		panic(err)
	}

	if appState[exported.ModuleName] != nil {
		genesisState[exported.ModuleName] = appState[exported.ModuleName]
	}

	// reset DefaultTestingAppInit to its original value
	ibctesting.DefaultTestingAppInit = setupTestingApp
	return app, genesisState
}

func (suite *TypesTestSuite) SetupWasmGrandpaWithChannel() {
	// Setup is assigned in init  and will be overwritten by this. SetupTestingWithChannel does use the same simapp
	// in 08-wasm directory so this should not affect what test app we use.
	ibctesting.DefaultTestingAppInit = SetupTestingWithChannel
	suite.SetupWasmGrandpa()
}

func TestWasmTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TypesTestSuite))
}

func getAltSigners(altVal *tmtypes.Validator, altPrivVal tmtypes.PrivValidator) map[string]tmtypes.PrivValidator {
	return map[string]tmtypes.PrivValidator{altVal.Address.String(): altPrivVal}
}
