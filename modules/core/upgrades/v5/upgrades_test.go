package v5_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	v5 "github.com/cosmos/ibc-go/v3/modules/core/upgrades/v5"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

type UpgradeTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *UpgradeTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	// TODO: remove
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
}

func TestIBCTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeTestSuite))
}

func (suite *UpgradeTestSuite) TestUpgradeLocalhostClients() {

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
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			ctx := suite.chainA.GetContext()
			clientStore := suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), v5.Localhost)

			clientStore.Set(host.ClientStateKey(), []byte("val"))

			for i := 0; i < 100; i++ {
				clientStore.Set(host.ConsensusStateKey(clienttypes.NewHeight(1, uint64(i))), sdk.Uint64ToBigEndian(uint64(i)))
			}

			err := v5.UpgradeLocalhostClients(ctx, suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)

			if tc.expPass {
				suite.Require().NoError(err)

				suite.Require().False(clientStore.Has(host.ClientStateKey()))
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
