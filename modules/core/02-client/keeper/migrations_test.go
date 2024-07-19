package keeper_test

import (
	"github.com/cosmos/ibc-go/v9/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// TestMigrateParams tests the migration for the client params
func (suite *KeeperTestSuite) TestMigrateParams() {
	testCases := []struct {
		name           string
		malleate       func()
		expectedParams types.Params
	}{
		{
			"success: default params",
			func() {
				params := types.DefaultParams()
				subspace := suite.chainA.GetSimApp().GetSubspace(ibcexported.ModuleName)
				subspace.SetParamSet(suite.chainA.GetContext(), &params)
			},
			types.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()

			ctx := suite.chainA.GetContext()
			migrator := keeper.NewMigrator(suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
			err := migrator.MigrateParams(ctx)
			suite.Require().NoError(err)

			params := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetParams(ctx)
			suite.Require().Equal(tc.expectedParams, params)
		})
	}
}

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
