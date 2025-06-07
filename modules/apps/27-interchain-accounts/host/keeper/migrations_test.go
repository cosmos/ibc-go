package keeper_test

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/runtime"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	icahostkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
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
		{
			"success: no legacy params pre-migration",
			func() {
				s.chainA.GetSimApp().ICAHostKeeper = icahostkeeper.NewKeeper(
					s.chainA.Codec,
					runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(icahosttypes.StoreKey)),
					nil, // assign a nil legacy param subspace
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
					s.chainA.GetSimApp().AccountKeeper,
					s.chainA.GetSimApp().MsgServiceRouter(),
					s.chainA.GetSimApp().GRPCQueryRouter(),
					authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				)
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
