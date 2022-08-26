package types_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v5/modules/apps/icq/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
)

type TypesTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *TypesTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

func (suite *TypesTestSuite) TestValidateGenesisState() {
	var genesisState types.GenesisState

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
			"failed to validate - empty value",
			func() {
				genesisState = types.GenesisState{}
			},
			false,
		},
		{
			"failed to validate - invalid host port",
			func() {
				genesisState = *types.NewHostGenesisState("p", types.DefaultParams())
			},
			false,
		},
		{
			"failed to validate - invalid empty query path",
			func() {
				genesisState = *types.NewHostGenesisState("port", types.NewParams(true, []string{" "}))
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			genesisState = *types.DefaultGenesis()

			tc.malleate() // malleate mutates test data

			err := genesisState.Validate()

			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}
