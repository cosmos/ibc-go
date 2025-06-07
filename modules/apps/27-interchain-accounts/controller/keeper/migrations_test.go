package keeper_test

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/runtime"

	icacontrollerkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
)

func (s *KeeperTestSuite) TestMigratorMigrateParams() {
	testCases := []struct {
		msg            string
		malleate       func()
		expectedParams icacontrollertypes.Params
	}{
		{
			"success: default params",
			func() {
				params := icacontrollertypes.DefaultParams()
				subspace := s.chainA.GetSimApp().GetSubspace(icacontrollertypes.SubModuleName) // get subspace
				subspace.SetParamSet(s.chainA.GetContext(), &params)                           // set params
			},
			icacontrollertypes.DefaultParams(),
		},
		{
			"success: no legacy params pre-migration",
			func() {
				s.chainA.GetSimApp().ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
					s.chainA.Codec,
					runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(icacontrollertypes.StoreKey)),
					nil, // assign a nil legacy param subspace
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
					s.chainA.GetSimApp().MsgServiceRouter(),
					s.chainA.GetSimApp().ICAControllerKeeper.GetAuthority(),
				)
			},
			icacontrollertypes.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate() // explicitly set params

			migrator := icacontrollerkeeper.NewMigrator(&s.chainA.GetSimApp().ICAControllerKeeper)
			err := migrator.MigrateParams(s.chainA.GetContext())
			s.Require().NoError(err)

			params := s.chainA.GetSimApp().ICAControllerKeeper.GetParams(s.chainA.GetContext())
			s.Require().Equal(tc.expectedParams, params)
		})
	}
}
