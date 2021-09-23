package types_test

import (
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
)

func (suite *TypesTestSuite) TestValidateVersion() {
	testCases := []struct {
		name    string
		version string
		expPass bool
	}{
		{
			"success",
			types.NewAppVersion(types.VersionPrefix, TestOwnerAddress),
			true,
		},
		{
			"success - version prefix only",
			types.VersionPrefix,
			true,
		},
		{
			"invalid version",
			types.NewAppVersion("ics27-5", TestOwnerAddress),
			false,
		},
		{
			"invalid account address - empty",
			types.NewAppVersion(types.VersionPrefix, ""),
			false,
		},
		{
			"invalid account address - exceeded character length",
			types.NewAppVersion(types.VersionPrefix, "ofwafxhdmqcdbpzvrccxkidbunrwyyoboyctignpvthxbwxtmnzyfwhhywobaatltfwafxhdmqcdbpzvrccxkidbunrwyyoboyctignpvthxbwxtmnzyfwhhywobaatlt"),
			false,
		},
		{
			"invalid account address - non alphanumeric characters",
			types.NewAppVersion(types.VersionPrefix, "-_-"),
			false,
		},
		{
			"invalid account address - address contains additional delimiter",
			types.NewAppVersion(types.VersionPrefix, "cosmos17dtl0mjt3t77kpu|hg2edqzjpszulwhgzuj9ljs"),
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := types.ValidateVersion(tc.version)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}
