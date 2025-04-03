package types_test

import (
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (suite *TypesTestSuite) TestNewControllerPortID() {
	var (
		path  *ibctesting.Path
		owner = TestOwnerAddress
	)

	testCases := []struct {
		name     string
		malleate func()
		expValue string
		expErr   error
	}{
		{
			"success",
			func() {},
			types.ControllerPortPrefix + TestOwnerAddress,
			nil,
		},
		{
			"invalid owner address",
			func() {
				owner = "    "
			},
			"",
			types.ErrInvalidAccountAddress,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.Setup()

			tc.malleate() // malleate mutates test data

			portID, err := types.NewControllerPortID(owner)

			if tc.expErr == nil {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(tc.expValue, portID)
			} else {
				suite.Require().Error(err, tc.name)
				suite.Require().Empty(portID)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
