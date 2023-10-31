package keeper_test

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"

	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	dbm "github.com/cosmos/cosmos-db"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing/simapp"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

const (
	defaultWasmClientID = "08-wasm-0"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	mockVM *wasmtesting.MockWasmEngine
}

func init() {
	ibctesting.DefaultTestingAppInit = setupTestingApp
}

// setupTestingApp provides the duplicated simapp which is specific to the 08-wasm module on chain creation.
func setupTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{}, nil)
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

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	queryHelper := baseapp.NewQueryServerTestHelper(suite.chainA.GetContext(), GetSimApp(suite.chainA).InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, GetSimApp(suite.chainA).WasmClientKeeper)
}

// SetupWasmWithMockVM sets up mock cometbft chain with a mock vm.
func (suite *KeeperTestSuite) SetupWasmWithMockVM() {
	ibctesting.DefaultTestingAppInit = suite.setupWasmWithMockVM

	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	_ = storeWasmCode(suite, wasmtesting.Code)
}

func (suite *KeeperTestSuite) setupWasmWithMockVM() (ibctesting.TestingApp, map[string]json.RawMessage) {
	suite.mockVM = wasmtesting.NewMockWasmEngine()
	// TODO: move default functionality required for wasm client testing to the mock VM
	suite.mockVM.StoreCodeFn = func(code wasmvm.WasmCode) (wasmvm.Checksum, error) {
		hash := sha256.Sum256(code)
		return wasmvm.Checksum(hash[:]), nil
	}

	suite.mockVM.PinFn = func(codeID wasmvm.Checksum) error {
		return nil
	}

	suite.mockVM.InstantiateFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
		var payload types.InstantiateMessage
		err := json.Unmarshal(initMsg, &payload)
		suite.Require().NoError(err)

		store.Set(host.ClientStateKey(), clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), payload.ClientState))
		store.Set(host.ConsensusStateKey(payload.ClientState.LatestHeight), clienttypes.MustMarshalConsensusState(suite.chainA.App.AppCodec(), payload.ConsensusState))
		return nil, 0, nil
	}

	suite.mockVM.RegisterQueryCallback(types.StatusMsg{}, func(codeID wasmvm.Checksum, env wasmvmtypes.Env, queryMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) ([]byte, uint64, error) {
		resp, err := json.Marshal(types.StatusResult{Status: exported.Active.String()})
		suite.Require().NoError(err)
		return resp, wasmtesting.DefaultGasUsed, nil
	})

	db := dbm.NewMemDB()
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{}, suite.mockVM)

	// reset DefaultTestingAppInit to its original value
	ibctesting.DefaultTestingAppInit = setupTestingApp
	return app, app.DefaultGenesis()
}

// storeWasmCode stores the wasm code on chain and returns the code hash.
func storeWasmCode(suite *KeeperTestSuite, wasmCode []byte) []byte {
	ctx := suite.chainA.GetContext().WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	msg := types.NewMsgStoreCode(authtypes.NewModuleAddress(govtypes.ModuleName).String(), wasmCode)
	response, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(ctx, msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(response.Checksum)
	return response.Checksum
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestNewKeeper() {
	testCases := []struct {
		name          string
		instantiateFn func()
		expPass       bool
		expError      error
	}{
		{
			"success",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(suite.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(suite.chainA).GetKey(types.StoreKey)),
					GetSimApp(suite.chainA).IBCKeeper.ClientKeeper,
					GetSimApp(suite.chainA).WasmClientKeeper.GetAuthority(),
					ibcwasm.GetVM(),
				)
			},
			true,
			nil,
		},
		{
			"failure: empty authority",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(suite.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(suite.chainA).GetKey(types.StoreKey)),
					GetSimApp(suite.chainA).IBCKeeper.ClientKeeper,
					"", // authority
					ibcwasm.GetVM(),
				)
			},
			false,
			errors.New("authority must be non-empty"),
		},
		{
			"failure: nil client keeper",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(suite.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(suite.chainA).GetKey(types.StoreKey)),
					nil, // client keeper,
					GetSimApp(suite.chainA).WasmClientKeeper.GetAuthority(),
					ibcwasm.GetVM(),
				)
			},
			false,
			errors.New("client keeper must be not nil"),
		},
		{
			"failure: nil wasm VM",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(suite.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(suite.chainA).GetKey(types.StoreKey)),
					GetSimApp(suite.chainA).IBCKeeper.ClientKeeper,
					GetSimApp(suite.chainA).WasmClientKeeper.GetAuthority(),
					nil,
				)
			},
			false,
			errors.New("wasm VM must be not nil"),
		},
		{
			"failure: nil store service",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(suite.chainA).AppCodec(),
					nil,
					GetSimApp(suite.chainA).IBCKeeper.ClientKeeper,
					GetSimApp(suite.chainA).WasmClientKeeper.GetAuthority(),
					ibcwasm.GetVM(),
				)
			},
			false,
			errors.New("store service must be not nil"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.SetupTest()

		suite.Run(tc.name, func() {
			if tc.expPass {
				suite.Require().NotPanics(
					tc.instantiateFn,
				)
			} else {
				suite.Require().PanicsWithError(tc.expError.Error(), func() {
					tc.instantiateFn()
				})
			}
		})
	}
}
