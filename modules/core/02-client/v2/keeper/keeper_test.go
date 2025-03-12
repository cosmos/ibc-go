package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	types2 "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/cosmos/ibc-go/v10/testing/simapp"
)

const (
	testClientID  = "tendermint-0"
	testClientID2 = "tendermint-1"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	cdc    codec.Codec
	ctx    sdk.Context
	keeper *keeper.Keeper
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	isCheckTx := false
	app := simapp.Setup(suite.T(), isCheckTx)

	suite.cdc = app.AppCodec()
	suite.ctx = app.BaseApp.NewContext(isCheckTx)
	suite.keeper = app.IBCKeeper.ClientV2Keeper
}

func (suite *KeeperTestSuite) TestSetClientCounterparty() {
	counterparty := types2.NewCounterpartyInfo([][]byte{[]byte("ibc"), []byte("channel-7")}, testClientID2)
	suite.keeper.SetClientCounterparty(suite.ctx, testClientID, counterparty)

	retrievedCounterparty, found := suite.keeper.GetClientCounterparty(suite.ctx, testClientID)
	suite.Require().True(found, "GetCounterparty failed")
	suite.Require().Equal(counterparty, retrievedCounterparty, "Counterparties are not equal")
}

func (suite *KeeperTestSuite) TestSetParams() {
	params := suite.keeper.GetParams(suite.ctx, testClientID)
	suite.Require().Equal(params, types.DefaultParams(), "did not return default params on initialization")

	newParams := types.NewParams(ibctesting.TestAccAddress)
	suite.keeper.SetParams(suite.ctx, testClientID, newParams)

	params = suite.keeper.GetParams(suite.ctx, testClientID)
	suite.Require().Equal(newParams, params, "params not set correctly")

	// params should be empty for a different clientID
	params = suite.keeper.GetParams(suite.ctx, testClientID2)
	suite.Require().Equal(types.DefaultParams(), params, "params should be empty for different clientID")

	// set params for a different clientID
	newParams2 := types.NewParams(ibctesting.TestAccAddress, suite.chainA.SenderAccount.GetAddress().String())
	suite.keeper.SetParams(suite.ctx, testClientID2, newParams2)

	params = suite.keeper.GetParams(suite.ctx, testClientID2)
	suite.Require().Equal(newParams2, params, "params not set correctly for different clientID")

	// params for original client unaffected
	params = suite.keeper.GetParams(suite.ctx, testClientID)
	suite.Require().Equal(newParams, params, "params not set correctly for original clientID")

}
