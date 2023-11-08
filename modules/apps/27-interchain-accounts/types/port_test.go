package types_test

import (
	"fmt"

	"github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v4/testing"
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
		expPass  bool
	}{
		{
			"success",
			func() {},
			fmt.Sprint(types.PortPrefix, TestOwnerAddress),
			true,
		},
		{
			"invalid owner address",
			func() {
				owner = "    "
			},
			"",
			false,
		},
		{
			"owner address is too long",
			func() {
				owner = ibctesting.GenerateString(types.MaximumOwnerLength + 100)
			},
			"",
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			tc.malleate() // malleate mutates test data

			// print owner
			fmt.Println("OWNER: ", owner)

			portID, err := types.NewControllerPortID(owner)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(tc.expValue, portID)
			} else {
				suite.Require().Error(err, tc.name)
				suite.Require().Empty(portID)
			}
		})
	}
}
