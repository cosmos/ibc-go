package types_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	dbm "github.com/cometbft/cometbft-db"
	tmjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/libs/log"
	tmtypes "github.com/cometbft/cometbft/types"

	simapp "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing/simapp"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

const (
	tmClientID      = "07-tendermint-0"
	grandpaClientID = "08-wasm-0"
	codeHash        = "01234567012345670123456701234567"
	// trustingPeriod  time.Duration = time.Hour * 24 * 7 * 2
	// ubdPeriod       time.Duration = time.Hour * 24 * 7 * 3
	// maxClockDrift   time.Duration = time.Second * 10
)

// var (
// 	height          = clienttypes.NewHeight(0, 4)
// 	newClientHeight = clienttypes.NewHeight(1, 1)
// 	upgradePath     = []string{"upgrade", "upgradedIBCState"}
// )

type TypesTestSuite struct {
	testifysuite.Suite
	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	// chainB      *ibctesting.TestChain

	ctx      sdk.Context
	store    sdk.KVStore
	codeHash []byte
	testData map[string]string
}

// func init() {
// 	ibctesting.DefaultTestingAppInit = setupTestingApp
// }

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
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{})
	return app, simapp.NewDefaultGenesisState(encCdc.Codec)
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
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{})
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

// func getAltSigners(altVal *tmtypes.Validator, altPrivVal tmtypes.PrivValidator) map[string]tmtypes.PrivValidator {
// 	return map[string]tmtypes.PrivValidator{altVal.Address.String(): altPrivVal}
// }
