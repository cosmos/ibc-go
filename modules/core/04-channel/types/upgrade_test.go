package types_test

import (
	errorsmod "cosmossdk.io/errors"

	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

func (suite *TypesTestSuite) TestUpgradeErrorIsOf() {
	var (
		upgradeError *channeltypes.UpgradeError
		intputErr    error
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			msg:      "standard sdk error",
			malleate: func() {},
			expPass:  true,
		},
		{
			msg: "not equal to nil error",
			malleate: func() {
				upgradeError = &channeltypes.UpgradeError{}
			},
			expPass: false,
		},
		{
			msg: "wrapped upgrade error",
			malleate: func() {
				wrappedErr := errorsmod.Wrap(upgradeError, "wrapped upgrade error")
				upgradeError = channeltypes.NewUpgradeError(1, wrappedErr)
			},
			expPass: true,
		},
		{
			msg: "empty upgrade and non nil target",
			malleate: func() {
				upgradeError = &channeltypes.UpgradeError{}
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.msg, func() {
			upgradeError = channeltypes.NewUpgradeError(1, channeltypes.ErrInvalidChannel)
			intputErr = channeltypes.ErrInvalidChannel

			tc.malleate()

			res := errorsmod.IsOf(upgradeError, intputErr)
			suite.Require().Equal(tc.expPass, res)
		})
	}
}

// TestGetErrorReceipt tests that the error receipt message is the same for both wrapped and unwrapped errors.
func (suite *TypesTestSuite) TestGetErrorReceipt() {
	upgradeError := channeltypes.NewUpgradeError(1, channeltypes.ErrInvalidChannel)

	wrappedErr := errorsmod.Wrap(upgradeError, "wrapped upgrade error")
	suite.Require().True(errorsmod.IsOf(wrappedErr, channeltypes.ErrInvalidChannel))

	upgradeError2 := channeltypes.NewUpgradeError(1, wrappedErr)

	suite.Require().Equal(upgradeError2.GetErrorReceipt().Message, upgradeError.GetErrorReceipt().Message)
}
