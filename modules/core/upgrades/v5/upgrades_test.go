package v5_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	v5 "github.com/cosmos/ibc-go/v3/modules/core/upgrades/v5"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

type UpgradeV5TestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *UpgradeV5TestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestIBCTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeV5TestSuite))
}

func (suite *UpgradeV5TestSuite) TestUpgradeLocalhostClients() {
	var clientStore sdk.KVStore

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success: prune localhost client state",
			func() {
				clientStore.Set(host.ClientStateKey(), []byte("clientState"))
			},
			true,
		},
		{
			"success: prune localhost client state and consensus states",
			func() {
				clientStore.Set(host.ClientStateKey(), []byte("clientState"))

				for i := 0; i < 10; i++ {
					clientStore.Set(host.ConsensusStateKey(clienttypes.NewHeight(1, uint64(i))), []byte("consensusState"))
				}
			},
			true,
		},
		{
			"07-tendermint client state and consensus states remain in client store",
			func() {
				clientStore = suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), clienttypes.FormatClientIdentifier(exported.Tendermint, 0))
				clientStore.Set(host.ClientStateKey(), []byte("clientState"))

				for i := 0; i < 10; i++ {
					clientStore.Set(host.ConsensusStateKey(clienttypes.NewHeight(1, uint64(i))), []byte("consensusState"))
				}
			},
			false,
		},
		{
			"06-solomachine client state and consensus states remain in client store",
			func() {
				clientStore = suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), clienttypes.FormatClientIdentifier(exported.Solomachine, 0))
				clientStore.Set(host.ClientStateKey(), []byte("clientState"))

				for i := 0; i < 10; i++ {
					clientStore.Set(host.ConsensusStateKey(clienttypes.NewHeight(1, uint64(i))), []byte("consensusState"))
				}
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			ctx := suite.chainA.GetContext()
			clientStore = suite.chainA.GetSimApp().IBCKeeper.ClientKeeper.ClientStore(suite.chainA.GetContext(), v5.Localhost)

			tc.malleate()

			v5.UpgradeToV5(ctx, suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)

			if tc.expPass {
				suite.Require().False(clientStore.Has(host.ClientStateKey()))

				for i := 0; i < 10; i++ {
					suite.Require().False(clientStore.Has(host.ConsensusStateKey(clienttypes.NewHeight(1, uint64(i)))))
				}
			} else {
				suite.Require().True(clientStore.Has(host.ClientStateKey()))

				for i := 0; i < 10; i++ {
					suite.Require().True(clientStore.Has(host.ConsensusStateKey(clienttypes.NewHeight(1, uint64(i)))))
				}
			}
		})
	}
}
