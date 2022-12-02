package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
	"github.com/cosmos/ibc-go/v6/testing/simapp"
)

func (suite *KeeperTestSuite) TestGetEscrowAccount() {
	var (
		portID    = ibctesting.TransferPort
		channelID = ibctesting.GetChainID(1)
	)

	escrowAcc := suite.chainA.GetSimApp().TransferKeeper.GetEscrowAccount(
		suite.chainA.GetContext(),
		portID,
		channelID,
	)

	expectedAddres := types.GetEscrowAddress(portID, channelID)
	suite.Require().Equal(expectedAddres, escrowAcc.GetAddress())

	// Check if the created escrow address is a module account
	acc := suite.chainA.GetSimApp().AccountKeeper.GetAccount(suite.chainA.GetContext(), expectedAddres)

	_, isModuleAccount := acc.(authtypes.ModuleAccountI)
	suite.Require().True(isModuleAccount)
}

func (suite *KeeperTestSuite) TestEscrowAccountMigration() {
	var (
		portID    = ibctesting.TransferPort
		channelID = ibctesting.GetChainID(2)
	)

	escrowAddress := types.GetEscrowAddress(portID, channelID)

	// fund escrow address
	trace := types.ParseDenomTrace(sdk.DefaultBondDenom)
	coin := sdk.NewCoin(trace.IBCDenom(), sdk.NewInt(100))

	suite.Require().NoError(simapp.FundAccount(suite.chainA.GetSimApp(), suite.chainA.GetContext(), escrowAddress, sdk.NewCoins(coin)))

	// check if the escrow account is standard account
	escrowAcc := suite.chainA.GetSimApp().AccountKeeper.GetAccount(suite.chainA.GetContext(), escrowAddress)
	_, isModuleAccount := escrowAcc.(authtypes.ModuleAccountI)
	suite.Require().False(isModuleAccount)

	// migrate the escrow account to a ModuleAccount
	suite.chainA.GetSimApp().TransferKeeper.GetEscrowAccount(
		suite.chainA.GetContext(),
		portID,
		channelID,
	)

	// check if the escrow account type changed to ModuleAccount
	escrowAcc = suite.chainA.GetSimApp().AccountKeeper.GetAccount(suite.chainA.GetContext(), escrowAddress)
	_, isModuleAccount = escrowAcc.(authtypes.ModuleAccountI)
	suite.Require().True(isModuleAccount)

	// check that the balance remained the same after the migration
	balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), escrowAddress, trace.IBCDenom())
	suite.Require().Equal(coin.Amount.Int64(), balance.Amount.Int64())
}
