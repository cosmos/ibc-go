package fee_test

import (
	"testing"

	fee "github.com/cosmos/ibc-go/modules/apps/29-fee"
	"github.com/cosmos/ibc-go/modules/apps/transfer"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
	"github.com/stretchr/testify/suite"
)

type FeeTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	path *ibctesting.Path

	moduleA fee.AppModule
}

func (suite *FeeTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	keeper := suite.chainA.GetSimApp().IBCFeeKeeper

	transferModule := transfer.NewAppModule(suite.chainA.GetSimApp().TransferKeeper)
	suite.moduleA = fee.NewAppModule(keeper, suite.chainA.GetSimApp().ScopedIBCFeeKeeper, transferModule)

	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	path.EndpointA.ChannelConfig.PortID = transfertypes.FeePortID
	path.EndpointB.ChannelConfig.PortID = transfertypes.FeePortID
	suite.path = path
}

func TestIBCFeeTestSuite(t *testing.T) {
	suite.Run(t, new(FeeTestSuite))
}
