package types_test

import (
	"fmt"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *TypesTestSuite) TestNewControllerPortID() {
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
			fmt.Sprint(types.ControllerPortPrefix, TestOwnerAddress),
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
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			s.coordinator.Setup(path)

			tc.malleate() // malleate mutates test data

			portID, err := types.NewControllerPortID(owner)

			if tc.expPass {
				s.Require().NoError(err, tc.name)
				s.Require().Equal(tc.expValue, portID)
			} else {
				s.Require().Error(err, tc.name)
				s.Require().Empty(portID)
			}
		})
	}
}
