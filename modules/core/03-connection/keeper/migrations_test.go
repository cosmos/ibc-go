package keeper_test

import (
	"github.com/cosmos/ibc-go/v7/modules/core/03-connection/keeper"
	"github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// TestMigrateParams tests that the params for the connection are properly migrated
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
				subspace.SetParamSet(s.chainA.GetContext(), &params) // set params
			},
			types.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			tc.malleate()

			ctx := s.chainA.GetContext()
			migrator := keeper.NewMigrator(s.chainA.GetSimApp().IBCKeeper.ConnectionKeeper)
			err := migrator.MigrateParams(ctx)
			s.Require().NoError(err)

			params := s.chainA.GetSimApp().IBCKeeper.ConnectionKeeper.GetParams(ctx)
			s.Require().Equal(tc.expectedParams, params)
		})
	}
}
