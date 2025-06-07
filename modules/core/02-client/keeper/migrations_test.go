package keeper_test

import (
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/keeper"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

func (s *KeeperTestSuite) TestMigrateToStatelessLocalhost() {
	// set localhost in state
	clientStore := s.chainA.GetSimApp().IBCKeeper.ClientKeeper.ClientStore(s.chainA.GetContext(), ibcexported.LocalhostClientID)
	clientStore.Set(host.ClientStateKey(), []byte("clientState"))

	m := keeper.NewMigrator(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
	err := m.MigrateToStatelessLocalhost(s.chainA.GetContext())
	s.Require().NoError(err)
	s.Require().False(clientStore.Has(host.ClientStateKey()))

	// rerun migration on no localhost set
	err = m.MigrateToStatelessLocalhost(s.chainA.GetContext())
	s.Require().NoError(err)
	s.Require().False(clientStore.Has(host.ClientStateKey()))
}
