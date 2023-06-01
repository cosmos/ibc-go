package types_test

import (
	errorsmod "cosmossdk.io/errors"

	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

func (suite *TypesTestSuite) TestUpgradeErrorIsOf() {
	ue := channeltypes.NewUpgradeError(1, channeltypes.ErrInvalidChannel)
	suite.Require().True(errorsmod.IsOf(ue, channeltypes.ErrInvalidChannel))
	suite.Require().False(errorsmod.IsOf(ue, channeltypes.UpgradeError{}))

	wrappedErr := errorsmod.Wrap(ue, "wrapped upgrade error")
	suite.Require().True(errorsmod.IsOf(wrappedErr, channeltypes.ErrInvalidChannel))
}

func (suite *TypesTestSuite) TestGetErrorReceipt() {
	ue := channeltypes.NewUpgradeError(1, channeltypes.ErrInvalidChannel)

	wrappedErr := errorsmod.Wrap(ue, "wrapped upgrade error")
	suite.Require().True(errorsmod.IsOf(wrappedErr, channeltypes.ErrInvalidChannel))

	ue2 := channeltypes.NewUpgradeError(1, wrappedErr)

	suite.Require().Equal(ue2.GetErrorReceipt().Message, ue.GetErrorReceipt().Message)
}
