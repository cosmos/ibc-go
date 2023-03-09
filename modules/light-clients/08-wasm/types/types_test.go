package types_test

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/keeper"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/cosmos/ibc-go/v7/testing/simapp"
	"github.com/stretchr/testify/suite"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

type WasmTestSuite struct {
	suite.Suite
	coordinator    *ibctesting.Coordinator
	wasm           *ibctesting.Wasm // singlesig public key
	chainA         *ibctesting.TestChain
	chainB         *ibctesting.TestChain
	ctx            sdk.Context
	cdc            codec.Codec
	now            time.Time
	store          sdk.KVStore
	clientState    exported.ClientState
	consensusState wasmtypes.ConsensusState
	codeId         []byte
	testData       map[string]string
	wasmKeeper     keeper.Keeper
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
    suite.SetupTest()
	clientState, ok := suite.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(suite.chainA.GetContext(), "08-wasm-0")
	if ok {
		suite.clientState = clientState
	}
}

func (suite *WasmTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	suite.wasm = ibctesting.NewWasm(suite.T(), suite.chainA.Codec, "wasmsingle", "testing", 1)

	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)

	// TODO: deprecate usage in favor of testing package
	checkTx := false
	app := simapp.Setup(checkTx)
	suite.cdc = app.AppCodec()
	suite.now = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	createClientData, err := os.ReadFile("test_data/data.json")
	suite.Require().NoError(err)
	err = json.Unmarshal(createClientData, &suite.testData)
	suite.Require().NoError(err)
	
	suite.ctx = suite.chainA.App.GetBaseApp().NewContext(checkTx, tmproto.Header{Height: 1, Time: suite.now}).WithGasMeter(sdk.NewInfiniteGasMeter())
	suite.store = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), "08-wasm-0")

	os.MkdirAll("tmp", 0o755)
	suite.wasmKeeper = suite.chainA.App.GetWasmKeeper()
	wasmContract, err := os.ReadFile("test_data/ics10_grandpa_cw.wasm")
	suite.Require().NoError(err)

	msg := wasmtypes.NewMsgPushNewWasmCode(authtypes.NewModuleAddress(govtypes.ModuleName).String(), wasmContract)
	response, err := suite.wasmKeeper.PushNewWasmCode(suite.ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response.CodeId)
	suite.codeId = response.CodeId

	//clientStateData := make([]byte, base64.StdEncoding.DecodedLen(len(suite.testData["create_client_client_state_data"])))
	//_, err = base64.StdEncoding.Decode(clientStateData, []byte(suite.testData["create_client_client_state_data"]))
	//suite.Require().NoError(err)


	wasmClientState := wasmtypes.ClientState{
		Data:   []byte(suite.testData["create_client_client_state_data"]),//clientStateData,
		CodeId: response.CodeId,
		LatestHeight: clienttypes.Height{
			RevisionNumber: 2000,
			RevisionHeight: 5,
		},
	}
	suite.clientState = &wasmClientState

	//consensusStateData := make([]byte, base64.StdEncoding.DecodedLen(len(suite.testData["create_client_consensus_state_data"])))
	//_, err = base64.StdEncoding.Decode(consensusStateData, []byte(suite.testData["create_client_consensus_state_data"]))
	//suite.Require().NoError(err)
	/*wasmConsensusState := wasmtypes.ConsensusState{
		Data:      []byte(suite.testData["create_client_consensus_state_data"]),//consensusStateData,
		CodeId:    response.CodeId,
		Timestamp: uint64(suite.now.UnixNano()),
		Root: &commitmenttypes.MerkleRoot{
			Hash: []byte{0},
		},
	}
	suite.consensusState = wasmConsensusState*/
}

func (suite *WasmTestSuite) TestPushNewWasmCodeWithErrors() {
	signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	data, err := os.ReadFile("test_data/ics10_grandpa_cw.wasm")
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
	res, err := suite.wasmKeeper.WasmCode(suite.ctx, &wasmtypes.WasmCodeQuery{CodeId: hex.EncodeToString(suite.codeId)})
	suite.Require().NoError(err)
	suite.Require().NotNil(res.Code)
}

/*func (suite *WasmTestSuite) TestWasm() {
	suite.Run("Init contract", func() {
		suite.SetupTest()
	})
}*/

func TestWasmTestSuite(t *testing.T) {
	suite.Run(t, new(WasmTestSuite))
}

func (suite *WasmTestSuite) Initialize() {
	err := suite.clientState.Initialize(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, &suite.consensusState)
	suite.Require().NoError(err)
}