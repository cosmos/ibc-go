package types_test

import (
	"fmt"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
)

func (suite *TypesTestSuite) TestValidateVersion() {
	testCases := []struct {
		name    string
		version string
		expPass bool
	}{
		{
			"success",
			fmt.Sprint(types.VersionPrefix, types.Delimiter, TestOwnerAddress),
			true,
		},
		{
			"success - version only",
			fmt.Sprint(types.VersionPrefix),
			true,
		},
		{
			"invalid version",
			fmt.Sprint("ics27-5", types.Delimiter, TestOwnerAddress),
			false,
		},
		{
			"invalid account address - empty",
			fmt.Sprint(types.VersionPrefix, types.Delimiter, ""),
			false,
		},
		{
			"invalid account address - exceeded character length",
			fmt.Sprint(types.VersionPrefix, types.Delimiter, "ofwafxhdmqcdbpzvrccxkidbunrwyyoboyctignpvthxbwxtmnzyfwhhywobaatltfwafxhdmqcdbpzvrccxkidbunrwyyoboyctignpvthxbwxtmnzyfwhhywobaatlt"),
			false,
		},
		{
			"invalid account address - non alphanumeric characters",
			fmt.Sprint(types.VersionPrefix, types.Delimiter, "-_-"),
			false,
		},
		{
			"invalid account address - address contains additional delimiter",
			fmt.Sprint(types.VersionPrefix, types.Delimiter, "cosmos17dtl0mjt3t77kpu|hg2edqzjpszulwhgzuj9ljs"),
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
