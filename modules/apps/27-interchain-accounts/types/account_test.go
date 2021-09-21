package types_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

var (
	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"
)

type TypesTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *TypesTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

func (suite *TypesTestSuite) TestGenerateAddress() {
	addr := types.GenerateAddress([]byte{}, "test-port-id")
	accAddr, err := sdk.AccAddressFromBech32(addr.String())

	suite.Require().NoError(err, "TestGenerateAddress failed")
	suite.Require().NotEmpty(accAddr)
}

func (suite *TypesTestSuite) TestGeneratePortID() {
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
			fmt.Sprintf("ics-27-0-0-%s", TestOwnerAddress),
			true,
		},
		{
			"success with non matching connection sequences",
			func() {
				path.EndpointA.ConnectionID = "connection-1"
			},
			fmt.Sprintf("ics-27-1-0-%s", TestOwnerAddress),
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
			"invalid connectionID",
			func() {
				path.EndpointA.ConnectionID = "connection"
			},
			"",
			false,
		},
		{
			"invalid counterparty connectionID",
			func() {
				path.EndpointB.ConnectionID = "connection"
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

			tc.malleate()

			portID, err := types.GeneratePortID(owner, path.EndpointA.ConnectionID, path.EndpointB.ConnectionID)

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
