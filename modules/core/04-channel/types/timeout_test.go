package types_test

import (
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

func (suite *TypesTestSuite) TestTimeoutPassed() {
	var timeout types.Timeout
	var passed bool

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"client height is not after timeout height",
			func() {
				passed = timeout.AfterHeight(clienttypes.NewHeight(0, 75))
			},
			false,
		},
		{
			"client timestamp is not after timeout timestamp",
			func() {
				passed = timeout.AfterTimestamp(75)
			},
			false,
		},
		{
			"client height is after timeout height",
			func() {
				passed = timeout.AfterHeight(clienttypes.NewHeight(0, 25))
			},
			true,
		},
		{
			"client timestamp is after timeout timestamp",
			func() {
				passed = timeout.AfterTimestamp(25)
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			timeout = types.NewTimeout(
				clienttypes.NewHeight(0, 50),
				50,
			)

			tc.malleate()

			if tc.expPass {
				suite.Require().True(passed)
			} else {
				suite.Require().False(passed)
			}
		})
	}
}

func (suite *TypesTestSuite) TestTimeout() {
	var timeout types.Timeout
	var valid bool

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid timeout",
			func() {},
			true,
		},
		{
			"invalid timeout",
			func() {
				timeout.Height = clienttypes.ZeroHeight()
				timeout.Timestamp = 0
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			timeout = types.NewTimeout(
				clienttypes.NewHeight(0, 50),
				0,
			)

			tc.malleate()

			valid = timeout.IsValid()

			if tc.expPass {
				suite.Require().True(valid)
			} else {
				suite.Require().False(valid)
			}
		})
	}
}
