package types_test

import (
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
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
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.Setup()

			tc.malleate() // malleate mutates test data

			portID, err := types.NewControllerPortID(owner)

			if tc.expErr == nil {
				s.Require().NoError(err, tc.name)
				s.Require().Equal(tc.expValue, portID)
			} else {
				s.Require().Error(err, tc.name)
				s.Require().Empty(portID)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
