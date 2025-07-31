package types_test

import (
	"encoding/json"
	"errors"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/log"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing/simapp"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const (
	tmClientID          = "07-tendermint-0"
	defaultWasmClientID = "08-wasm-0"
)

type TypesTestSuite struct {
	testifysuite.Suite
	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
}

func TestWasmTestSuite(t *testing.T) {
	testifysuite.Run(t, new(TypesTestSuite))
}

func (s *TypesTestSuite) SetupTest() {
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
