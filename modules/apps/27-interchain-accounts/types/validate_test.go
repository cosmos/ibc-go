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
			"invalid version",
			"ics27-5|abc123",
			false,
		},
		{
			"invalid account address - 31 chars",
			"ics27-1|xtignpvthxbwxtmnzyfwhhywobaatlt",
			false,
		},
		{
			"invalid account address - 65 chars",
			"ics27-1|ofwafxhdmqcdbpzvrccxkidbunrwyyoboyctignpvthxbwxtmnzyfwhhywobaatlt",
			false,
		},
		{
			"invalid account address - non alphanumeric characters",
			"ics27-1|abc_123",
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
