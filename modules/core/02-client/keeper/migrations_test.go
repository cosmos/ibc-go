package keeper_test

import (
	"github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

// TestMigrateParams tests the migration for the client params
func (suite *KeeperTestSuite) TestMigrateParams() {
	testCases := []struct {
		name           string
		malleate       func()
		expectedParams types.Params
		expectedError  error
	}{
		{
			name:          "error: must migrate to ibc-go v8.x first",
			malleate:      func() {},
			expectedError: ibcerrors.ErrInvalidVersion,
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

			if tc.expectedError != nil {
				suite.Require().ErrorIs(err, tc.expectedError)
			} else {
				suite.Require().NoError(err)

				params := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.GetParams(ctx)
				suite.Require().Equal(tc.expectedParams, params)
			}
		})
	}
}
