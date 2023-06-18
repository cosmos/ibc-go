package keeper_test

import (
	"fmt"

	icahostkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
)

func (s *KeeperTestSuite) TestMigratorMigrateParams() {
	testCases := []struct {
		msg            string
		malleate       func()
		expectedParams icahosttypes.Params
	}{
		{
			"success: default params",
			func() {
				params := icahosttypes.DefaultParams()
				subspace := s.chainA.GetSimApp().GetSubspace(icahosttypes.SubModuleName) // get subspace
				subspace.SetParamSet(s.chainA.GetContext(), &params)                     // set params
			},
			icahosttypes.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate() // explicitly set params

			migrator := icahostkeeper.NewMigrator(&s.chainA.GetSimApp().ICAHostKeeper)
			err := migrator.MigrateParams(s.chainA.GetContext())
			s.Require().NoError(err)

			params := s.chainA.GetSimApp().ICAHostKeeper.GetParams(s.chainA.GetContext())
			s.Require().Equal(tc.expectedParams, params)
		})
	}
}
