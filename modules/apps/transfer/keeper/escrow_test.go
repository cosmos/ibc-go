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

	acc := suite.chainA.GetSimApp().TransferKeeper.GetEscrowAccount(
		suite.chainA.GetContext(),
		portID,
		channelID,
	)

	address := types.GetEscrowAddress(portID, channelID)
	suite.Require().Equal(address, acc.GetAddress())

	// Check if the created escrow address is a module account
	moduleAcc := suite.chainA.GetSimApp().AccountKeeper.GetAccount(suite.chainA.GetContext(), address)

	_, isModuleAccount := moduleAcc.(authtypes.ModuleAccountI)
	suite.Require().True(isModuleAccount)
}

func (suite *KeeperTestSuite) TestEscrowAccountMigration() {
	var (
		portID    = ibctesting.TransferPort
		channelID = ibctesting.GetChainID(2)
	)

	address := types.GetEscrowAddress(portID, channelID)

	// fund escrow address
	trace := types.ParseDenomTrace(sdk.DefaultBondDenom)
	coin := sdk.NewCoin(trace.IBCDenom(), sdk.NewInt(100))

	suite.Require().NoError(simapp.FundAccount(suite.chainA.GetSimApp(), suite.chainA.GetContext(), address, sdk.NewCoins(coin)))

	// check if the escrow account is standard account
	acc := suite.chainA.GetSimApp().AccountKeeper.GetAccount(suite.chainA.GetContext(), address)
	_, isModuleAccount := acc.(authtypes.ModuleAccountI)
	suite.Require().False(isModuleAccount)

	// migrate the escrow account to a ModuleAccount
	suite.chainA.GetSimApp().TransferKeeper.GetEscrowAccount(
		suite.chainA.GetContext(),
		portID,
		channelID,
	)

	// check if the escrow account type changed to ModuleAccount
	acc = suite.chainA.GetSimApp().AccountKeeper.GetAccount(suite.chainA.GetContext(), address)
	_, isModuleAccount = acc.(authtypes.ModuleAccountI)
	suite.Require().True(isModuleAccount)

	// check that the balance remained the same after the migration
	balance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), address, trace.IBCDenom())
	suite.Require().Equal(coin.Amount.Int64(), balance.Amount.Int64())
}
