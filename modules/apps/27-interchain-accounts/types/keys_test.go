package types_test

import (
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
)

func (suite *TypesTestSuite) TestKeyActiveChannel() {
	key := types.KeyActiveChannel("port-id")
	suite.Require().Equal("activeChannel/port-id", string(key))
}

func (suite *TypesTestSuite) TestKeyOwnerAccount() {
	key := types.KeyOwnerAccount("port-id")
	suite.Require().Equal("owner/port-id", string(key))
}

func (suite *TypesTestSuite) TestParseControllerConnSequence() {

	testCases := []struct {
		name     string
		portID   string
		expValue uint64
		expPass  bool
	}{
		{
			"success",
			TestPortID,
			0,
			true,
		},
		{
			"failed to parse port identifier",
			"invalid-port-id",
			0,
			false,
		},
		{
			"failed to parse connection sequence",
			"ics27-1.x.y.cosmos1",
			0,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			connSeq, err := types.ParseControllerConnSequence(tc.portID)

			if tc.expPass {
				suite.Require().Equal(tc.expValue, connSeq)
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Zero(connSeq)
				suite.Require().Error(err, tc.name)
			}
		})
	}
}

func (suite *TypesTestSuite) TestParseHostConnSequence() {

	testCases := []struct {
		name     string
		portID   string
		expValue uint64
		expPass  bool
	}{
		{
			"success",
			TestPortID,
			0,
			true,
		},
		{
			"failed to parse port identifier",
			"invalid-port-id",
			0,
			false,
		},
		{
			"failed to parse connection sequence",
			"ics27-1.x.y.cosmos1",
			0,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			connSeq, err := types.ParseHostConnSequence(tc.portID)

			if tc.expPass {
				suite.Require().Equal(tc.expValue, connSeq)
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Zero(connSeq)
				suite.Require().Error(err, tc.name)
			}
		})
	}
}
