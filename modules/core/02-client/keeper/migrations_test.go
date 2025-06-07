package keeper_test

import (
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// TestMigrateParams tests the migration for the client params
func (s *KeeperTestSuite) TestMigrateParams() {
	testCases := []struct {
		name           string
		malleate       func()
		expectedParams types.Params
	}{
		{
			"success: default params",
			func() {
				params := types.DefaultParams()
				subspace := s.chainA.GetSimApp().GetSubspace(ibcexported.ModuleName)
				subspace.SetParamSet(s.chainA.GetContext(), &params)
			},
			types.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			tc.malleate()

			ctx := s.chainA.GetContext()
			migrator := keeper.NewMigrator(s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			err := migrator.MigrateParams(ctx)
			s.Require().NoError(err)

			params := s.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetParams(ctx)
			s.Require().Equal(tc.expectedParams, params)
		})
	}
}

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
