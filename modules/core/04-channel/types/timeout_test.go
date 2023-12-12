package types_test

import (
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

func (suite *TypesTestSuite) TestIsValid() {
	var timeout types.Timeout

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: valid timeout with height and timestamp",
			func() {
				timeout = types.NewTimeout(clienttypes.NewHeight(1, 100), 100)
			},
			true,
		},
		{
			"success: valid timeout with height and zero timestamp",
			func() {
				timeout = types.NewTimeout(clienttypes.NewHeight(1, 100), 0)
			},
			true,
		},
		{
			"success: valid timeout with timestamp and zero height",
			func() {
				timeout = types.NewTimeout(clienttypes.ZeroHeight(), 100)
			},
			true,
		},
		{
			"invalid timeout with zero height and zero timestamp",
			func() {
				timeout = types.NewTimeout(clienttypes.ZeroHeight(), 0)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			tc.malleate()

			isValid := timeout.IsValid()
			if tc.expPass {
				suite.Require().True(isValid)
			} else {
				suite.Require().False(isValid)
			}
		})
	}
}
