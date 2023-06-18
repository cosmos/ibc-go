package v7_test

import (
	v7 "github.com/cosmos/ibc-go/v7/modules/core/02-client/migrations/v7"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

func (s *MigrationsV7TestSuite) TestMigrateLocalhostClient() {
	s.SetupTest()

	// note: explicitly remove the localhost client before running migration handler
	clientStore := s.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), exported.LocalhostClientID)
	clientStore.Delete(host.ClientStateKey())

	clientState, found := s.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), exported.LocalhostClientID)
	s.Require().False(found)
	s.Require().Nil(clientState)

	err := v7.MigrateLocalhostClient(s.chainA.GetContext(), s.chainA.GetSimApp().GetIBCKeeper().ClientKeeper)
	s.Require().NoError(err)

	clientState, found = s.chainA.GetSimApp().GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), exported.LocalhostClientID)
	s.Require().True(found)
	s.Require().NotNil(clientState)
}
