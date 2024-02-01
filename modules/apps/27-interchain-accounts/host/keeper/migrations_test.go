package keeper_test

import (
	"fmt"

	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
)

func (suite *KeeperTestSuite) TestMigratorMigrateParams() {
	testCases := []struct {
		msg            string
		malleate       func()
		expectedParams icahosttypes.Params
		expectedError  error
	}{
		{
			msg:           "error: must migrate to ibc-go v8.x first",
			malleate:      func() {},
			expectedError: icatypes.ErrInvalidVersion,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate() // explicitly set params

			migrator := icahostkeeper.NewMigrator(&suite.chainA.GetSimApp().ICAHostKeeper)
			err := migrator.MigrateParams(suite.chainA.GetContext())

			if tc.expectedError != nil {
				suite.Require().ErrorIs(err, tc.expectedError)
			} else {
				suite.Require().NoError(err)

				params := suite.chainA.GetSimApp().ICAHostKeeper.GetParams(suite.chainA.GetContext())
				suite.Require().Equal(tc.expectedParams, params)
			}
		})
	}
}
