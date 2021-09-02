package fee_test

import (
	"testing"

	fee "github.com/cosmos/ibc-go/modules/apps/29-fee"
	feekeeper "github.com/cosmos/ibc-go/modules/apps/29-fee/keeper"
	"github.com/cosmos/ibc-go/modules/apps/transfer"
	ibctesting "github.com/cosmos/ibc-go/testing"
	"github.com/stretchr/testify/suite"
)

type FeeTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	module fee.AppModule
	keeper feekeeper.Keeper
}

func (suite *FeeTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	suite.keeper = suite.chainA.GetSimApp().IBCFeeKeeper

	transferModule := transfer.NewAppModule(suite.chainA.GetSimApp().TransferKeeper)
	suite.module = fee.NewAppModule(suite.keeper, suite.chainA.GetSimApp().ScopedIBCFeeKeeper, transferModule)
}

func TestIBCFeeTestSuite(t *testing.T) {
	suite.Run(t, new(FeeTestSuite))
}
