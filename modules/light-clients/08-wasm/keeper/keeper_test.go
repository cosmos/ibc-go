package keeper_test

import (
	"encoding/json"
	"errors"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/keeper"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing/simapp"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
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
					"", // authority
					ibcwasm.GetVM(),
				)
			},
			false,
			errors.New("authority must be non-empty"),
		},
		{
			"failure: nil wasm VM",
			func() {
				keeper.NewKeeperWithVM(
					GetSimApp(suite.chainA).AppCodec(),
					runtime.NewKVStoreService(GetSimApp(suite.chainA).GetKey(types.StoreKey)),
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
