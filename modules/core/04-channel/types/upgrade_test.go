package types_test

import (
	errorsmod "cosmossdk.io/errors"

	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

func (suite *TypesTestSuite) TestUpgradeErrorIsOf() {
	ue := channeltypes.UpgradeError{}
	suite.Require().True(errorsmod.IsOf(ue, channeltypes.UpgradeError{}))
	suite.Require().False(errorsmod.IsOf(ue, channeltypes.ErrInvalidChannel))

	wrappedErr := errorsmod.Wrap(ue, "wrapped upgrade error")
	suite.Require().True(errorsmod.IsOf(wrappedErr, channeltypes.UpgradeError{}))
}
