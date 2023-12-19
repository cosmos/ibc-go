package types_test

import (
	"errors"

	errorsmod "cosmossdk.io/errors"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/cosmos/ibc-go/v8/testing/mock"
)

func (suite *TypesTestSuite) TestUpgradeValidateBasic() {
	var upgrade types.Upgrade

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"invalid ordering",
			func() {
				upgrade.Fields.Ordering = types.NONE
			},
			false,
		},
		{
			"connection hops length not equal to 1",
			func() {
				upgrade.Fields.ConnectionHops = []string{"connection-0", "connection-1"}
			},
			false,
		},
		{
			"empty version",
			func() {
				upgrade.Fields.Version = "  "
			},
			false,
		},
		{
			"invalid timeout",
			func() {
				upgrade.Timeout.Height = clienttypes.ZeroHeight()
				upgrade.Timeout.Timestamp = 0
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			upgrade = types.NewUpgrade(
				types.NewUpgradeFields(types.ORDERED, []string{ibctesting.FirstConnectionID}, mock.Version),
				types.NewTimeout(clienttypes.NewHeight(0, 100), 0),
				0,
			)

			tc.malleate()

			err := upgrade.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TypesTestSuite) TestUpgradeErrorIsOf() {
	var (
		upgradeError *types.UpgradeError
		inputErr     error
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
			msg: "input is upgrade error",
			malleate: func() {
				inputErr = types.NewUpgradeError(1, types.ErrInvalidChannel)
			},
			expPass: true,
		},
		{
			msg: "input has wrapped upgrade error",
			malleate: func() {
				wrappedErr := errorsmod.Wrap(types.ErrInvalidChannel, "wrapped upgrade error")
				inputErr = types.NewUpgradeError(1, wrappedErr)
			},
			expPass: true,
		},
		{
			msg: "not equal to nil error",
			malleate: func() {
				upgradeError = &types.UpgradeError{}
			},
			expPass: false,
		},
		{
			msg: "wrapped upgrade error",
			malleate: func() {
				wrappedErr := errorsmod.Wrap(upgradeError, "wrapped upgrade error")
				upgradeError = types.NewUpgradeError(1, wrappedErr)
			},
			expPass: true,
		},
		{
			msg: "empty upgrade and non nil target",
			malleate: func() {
				upgradeError = &types.UpgradeError{}
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.msg, func() {
			upgradeError = types.NewUpgradeError(1, types.ErrInvalidChannel)
			inputErr = types.ErrInvalidChannel

			tc.malleate()

			res := errorsmod.IsOf(upgradeError, inputErr)
			suite.Require().Equal(tc.expPass, res)
		})
	}
}

// TestGetErrorReceipt tests that the error receipt message is the same for both wrapped and unwrapped errors.
func (suite *TypesTestSuite) TestGetErrorReceipt() {
	upgradeError := types.NewUpgradeError(1, types.ErrInvalidChannel)

	wrappedErr := errorsmod.Wrap(upgradeError, "wrapped upgrade error")
	suite.Require().True(errorsmod.IsOf(wrappedErr, types.ErrInvalidChannel))

	upgradeError2 := types.NewUpgradeError(1, wrappedErr)

	suite.Require().Equal(upgradeError2.GetErrorReceipt().Message, upgradeError.GetErrorReceipt().Message)
}

// TestUpgradeErrorUnwrap tests that the underlying error is not modified when Unwrap is called.
func (suite *TypesTestSuite) TestUpgradeErrorUnwrap() {
	baseUnderlyingError := errorsmod.Wrap(types.ErrInvalidChannel, "base error")
	wrappedErr := errorsmod.Wrap(baseUnderlyingError, "wrapped error")
	upgradeError := types.NewUpgradeError(1, wrappedErr)

	originalUpgradeError := upgradeError.Error()
	unWrapped := errors.Unwrap(upgradeError)
	postUnwrapUpgradeError := upgradeError.Error()

	suite.Require().Equal(types.ErrInvalidChannel, unWrapped, "unwrapped error was not equal to base underlying error")
	suite.Require().Equal(originalUpgradeError, postUnwrapUpgradeError, "original error was modified when unwrapped")
}

func (suite *TypesTestSuite) TestIsUpgradeError() {
	var err error

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"true",
			func() {},
			true,
		},
		{
			"false with non upgrade error",
			func() {
				err = errors.New("error")
			},
			false,
		},
		{
			"false with nil error",
			func() {
				err = nil
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.msg, func() {
			err = types.NewUpgradeError(1, types.ErrInvalidChannel)

			tc.malleate()

			res := types.IsUpgradeError(err)
			suite.Require().Equal(tc.expPass, res)
		})
	}
}
