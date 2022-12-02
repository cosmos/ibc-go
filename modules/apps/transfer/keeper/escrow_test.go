package keeper_test

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
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
