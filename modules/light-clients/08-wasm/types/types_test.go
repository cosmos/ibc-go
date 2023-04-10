package types_test

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	tmjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/libs/log"
	tmtypes "github.com/cometbft/cometbft/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	tmclient "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/keeper"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/simapp"
	"github.com/stretchr/testify/suite"
)

const (
	chainID                        = "gaia"
	chainIDRevision0               = "gaia-revision-0"
	chainIDRevision1               = "gaia-revision-1"
	clientID                       = "gaiamainnet"
	trustingPeriod   time.Duration = time.Hour * 24 * 7 * 2
	ubdPeriod        time.Duration = time.Hour * 24 * 7 * 3
	maxClockDrift    time.Duration = time.Second * 10
)

var (
	height          = clienttypes.NewHeight(0, 4)
	newClientHeight = clienttypes.NewHeight(1, 1)
	upgradePath     = []string{"upgrade", "upgradedIBCState"}
)

type WasmTestSuite struct {
	suite.Suite
	coordinator    *ibctesting.Coordinator
	chainA         *ibctesting.TestChain
	chainB         *ibctesting.TestChain
	ctx            sdk.Context
	now            time.Time
	store          sdk.KVStore
	clientState    exported.ClientState
	consensusState wasmtypes.ConsensusState
	codeID         []byte
	testData       map[string]string
	wasmKeeper     keeper.Keeper
}

func (suite *WasmTestSuite) SetupWasmTendermint() {
	ibctesting.DefaultTestingAppInit = ibctesting.SetupTestingApp
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainA.SetWasm(true)
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainB.SetWasm(true)

	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)

	suite.ctx = suite.chainA.GetContext().WithBlockGasMeter(sdk.NewInfiniteGasMeter())
	suite.store = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, "08-wasm-0")

	err := os.MkdirAll("tmp", 0o755)
	suite.Require().NoError(err)
	suite.wasmKeeper = suite.chainA.App.GetWasmKeeper()
	wasmContract, err := os.ReadFile("test_data/ics07_tendermint_cw.wasm.gz")
	suite.Require().NoError(err)

	msg := wasmtypes.NewMsgPushNewWasmCode(authtypes.NewModuleAddress(govtypes.ModuleName).String(), wasmContract)
	response, err := suite.wasmKeeper.PushNewWasmCode(suite.chainA.GetContext(), msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response.CodeId)
	suite.codeID = response.CodeId

	response, err = suite.chainB.App.GetWasmKeeper().PushNewWasmCode(suite.chainB.GetContext(), msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response.CodeId)
	suite.codeID = response.CodeId

	suite.coordinator.SetCodeID(suite.codeID)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
}

func SetupTestingWithChannel() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	encCdc := simapp.MakeTestEncodingConfig()
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, map[int64]bool{}, simapp.DefaultNodeHome, 5, encCdc, simtestutil.EmptyAppOptions{})
	genesisState := simapp.NewDefaultGenesisState(encCdc.Marshaler)

	bytes, err := os.ReadFile("test_data/genesis.json")
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

	return app, genesisState
}

func (suite *WasmTestSuite) SetupWithChannel() {
	ibctesting.DefaultTestingAppInit = SetupTestingWithChannel
	suite.CommonSetupTest()
	clientState, ok := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.ctx, "08-wasm-0")
	if ok {
		suite.clientState = clientState
		// Update with current contract hash
		suite.clientState.(*wasmtypes.ClientState).CodeId = suite.codeID
		suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, "08-wasm-0", suite.clientState)
	}
}

func (suite *WasmTestSuite) SetupWithEmptyClient() {
	ibctesting.DefaultTestingAppInit = ibctesting.SetupTestingApp
	suite.CommonSetupTest()

	clientStateData, err := base64.StdEncoding.DecodeString(suite.testData["client_state_data"])
	suite.Require().NoError(err)

	wasmClientState := wasmtypes.ClientState{
		Data:   clientStateData,
		CodeId: suite.codeID,
		LatestHeight: clienttypes.Height{
			RevisionNumber: 2000,
			RevisionHeight: 4,
		},
	}
	suite.clientState = &wasmClientState

	consensusStateData, err := base64.StdEncoding.DecodeString(suite.testData["consensus_state_data"])
	suite.Require().NoError(err)
	wasmConsensusState := wasmtypes.ConsensusState{
		Data:      consensusStateData,
		Timestamp: uint64(1678304292),
	}
	suite.consensusState = wasmConsensusState
}

func (suite *WasmTestSuite) CommonSetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)

	testData, err := os.ReadFile("test_data/data.json")
	suite.Require().NoError(err)
	err = json.Unmarshal(testData, &suite.testData)
	suite.Require().NoError(err)

	suite.ctx = suite.chainA.GetContext().WithBlockGasMeter(sdk.NewInfiniteGasMeter())
	suite.store = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.ctx, "08-wasm-0")

	err = os.MkdirAll("tmp", 0o755)
	suite.Require().NoError(err)
	suite.wasmKeeper = suite.chainA.App.GetWasmKeeper()
	wasmContract, err := os.ReadFile("test_data/ics10_grandpa_cw.wasm.gz")
	suite.Require().NoError(err)

	msg := wasmtypes.NewMsgPushNewWasmCode(authtypes.NewModuleAddress(govtypes.ModuleName).String(), wasmContract)
	response, err := suite.wasmKeeper.PushNewWasmCode(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response.CodeId)
	suite.codeID = response.CodeId
}

func (suite *WasmTestSuite) TestPushNewWasmCodeWithErrors() {
	suite.SetupWithEmptyClient()
	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	data, err := os.ReadFile("test_data/ics10_grandpa_cw.wasm.gz")
	suite.Require().NoError(err)

	// test wasmcode duplication
	msg := wasmtypes.NewMsgPushNewWasmCode(signer, data)
	_, err = suite.wasmKeeper.PushNewWasmCode(suite.ctx, msg)
	suite.Require().Error(err)

	// test invalid wasm code
	msg = wasmtypes.NewMsgPushNewWasmCode(signer, []byte{})
	_, err = suite.wasmKeeper.PushNewWasmCode(suite.ctx, msg)
	suite.Require().Error(err)
}

func (suite *WasmTestSuite) TestQueryWasmCode() {
	// test invalid query request
	_, err := suite.wasmKeeper.WasmCode(suite.ctx, &wasmtypes.WasmCodeQuery{})
	suite.Require().Error(err)

	_, err = suite.wasmKeeper.WasmCode(suite.ctx, &wasmtypes.WasmCodeQuery{CodeId: "test"})
	suite.Require().Error(err)

	// test valid query request
	res, err := suite.wasmKeeper.WasmCode(suite.ctx, &wasmtypes.WasmCodeQuery{CodeId: hex.EncodeToString(suite.codeID)})
	suite.Require().NoError(err)
	suite.Require().NotNil(res.Code)
}

func (suite *WasmTestSuite) TestQueryAllWasmCode() {
	// test valid query request
	res, err := suite.wasmKeeper.AllWasmCodeID(suite.ctx, &wasmtypes.AllWasmCodeIDQuery{})
	suite.Require().NoError(err)
	suite.Require().NotNil(res.CodeIds)
}

func TestWasmTestSuite(t *testing.T) {
	suite.Run(t, new(WasmTestSuite))
}

func NewTendermintClientState(endpoint *ibctesting.Endpoint, height exported.Height) tmclient.ClientState {
	tmConfig, ok := endpoint.ClientConfig.(*ibctesting.TendermintConfig)
	if !ok {
		panic("Panic casting ClientConfig to TendermindConfig")
	}

	clientHeight := height.(clienttypes.Height)

	clientState := tmclient.NewClientState(
		endpoint.Counterparty.ClientID,
		tmConfig.TrustLevel,
		tmConfig.TrustingPeriod,
		tmConfig.UnbondingPeriod,
		tmConfig.MaxClockDrift,
		clientHeight,
		commitmenttypes.GetSDKSpecs(),
		ibctesting.UpgradePath,
	)

	return *clientState
}

func getAltSigners(altVal *tmtypes.Validator, altPrivVal tmtypes.PrivValidator) map[string]tmtypes.PrivValidator {
	return map[string]tmtypes.PrivValidator{altVal.Address.String(): altPrivVal}
}
