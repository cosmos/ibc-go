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
			fmt.Sprint(types.Version, types.Delimiter, TestOwnerAddress),
			true,
		},
		{
			"success - version only",
			fmt.Sprint(types.Version),
			true,
		},
		{
			"invalid version",
			fmt.Sprint("ics27-5", types.Delimiter, TestOwnerAddress),
			false,
		},
		{
			"invalid account address - 31 chars",
			fmt.Sprint(types.Version, types.Delimiter, "xtignpvthxbwxtmnzyfwhhywobaatlt"),
			false,
		},
		{
			"invalid account address - 65 chars",
			fmt.Sprint(types.Version, types.Delimiter, "ofwafxhdmqcdbpzvrccxkidbunrwyyoboyctignpvthxbwxtmnzyfwhhywobaatlt"),
			false,
		},
		{
			"invalid account address - non alphanumeric characters",
			fmt.Sprint(types.Version, types.Delimiter, "-_-"),
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
