package keeper_test

import (
	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

// TestMigrateDefaultParams tests the migration for the channel params
func (suite *KeeperTestSuite) TestMigrateParams() {
	testCases := []struct {
		name           string
		expectedParams channeltypes.Params
		expectedError  error
	}{
		{
			name:          "error: must migrate to ibc-go v8.x first",
			expectedError: ibcerrors.ErrInvalidVersion,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			ctx := suite.chainA.GetContext()
			migrator := keeper.NewMigrator(suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper)
			err := migrator.MigrateParams(ctx)

			if tc.expectedError != nil {
				suite.Require().ErrorIs(err, tc.expectedError)
			} else {
				suite.Require().NoError(err)

				params := suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.GetParams(ctx)
				suite.Require().Equal(tc.expectedParams, params)
			}
		})
	}
}
