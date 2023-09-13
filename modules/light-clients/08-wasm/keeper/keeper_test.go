package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	queryHelper := baseapp.NewQueryServerTestHelper(suite.chainA.GetContext(), suite.chainA.GetSimApp().InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.chainA.GetSimApp().WasmClientKeeper)
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}
