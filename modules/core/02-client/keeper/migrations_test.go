package keeper_test

import (
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/keeper"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

func (suite *KeeperTestSuite) TestMigrateToStatelessLocalhost() {
	// set localhost in state
	clientStore := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), ibcexported.LocalhostClientID)
	clientStore.Set(host.ClientStateKey(), []byte("clientState"))

	m := keeper.NewMigrator(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
	err := m.MigrateToStatelessLocalhost(suite.chainA.GetContext())
	suite.Require().NoError(err)
	suite.Require().False(clientStore.Has(host.ClientStateKey()))

	// rerun migration on no localhost set
	err = m.MigrateToStatelessLocalhost(suite.chainA.GetContext())
	suite.Require().NoError(err)
	suite.Require().False(clientStore.Has(host.ClientStateKey()))
}
