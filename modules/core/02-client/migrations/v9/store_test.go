package v9_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v8/modules/core/02-client/migrations/v9"
	"github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// numCreations is the number of clients/consensus states created for light clients clients
const numCreations = 10

type MigrationsV9TestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator
	chain       *ibctesting.TestChain
}

func (suite *MigrationsV9TestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1)

	suite.chain = suite.coordinator.GetChain(ibctesting.GetChainID(1))
}

func TestIBCTestSuite(t *testing.T) {
	testifysuite.Run(t, new(MigrationsV9TestSuite))
}

func (suite *MigrationsV9TestSuite) TestMigrateStore() {
	suite.createLocalhostClient()

	err := v9.MigrateStore(suite.chain.GetContext(), suite.chain.GetSimApp().IBCKeeper.ClientKeeper)
	suite.Require().NoError(err)

	suite.assertNoLocalhostClients()
}

func (suite *MigrationsV9TestSuite) TestMigrateStoreNoLocalhost() {
	err := v9.MigrateStore(suite.chain.GetContext(), suite.chain.GetSimApp().IBCKeeper.ClientKeeper)
	suite.Require().NoError(err)

	suite.assertNoLocalhostClients()
}

func (suite *MigrationsV9TestSuite) createLocalhostClient() {
	clientStore := suite.chain.GetSimApp().IBCKeeper.ClientKeeper.ClientStore(suite.chain.GetContext(), exported.LocalhostClientID)

	clientStore.Set(host.ClientStateKey(), []byte("clientState"))

	for i := 0; i < numCreations; i++ {
		clientStore.Set(host.ConsensusStateKey(types.NewHeight(1, uint64(i))), []byte("consensusState"))
	}
}

func (suite *MigrationsV9TestSuite) assertNoLocalhostClients() {
	for numClients := uint64(0); numClients < numCreations; numClients++ {
		clientStore := suite.chain.GetSimApp().IBCKeeper.ClientKeeper.ClientStore(suite.chain.GetContext(), exported.LocalhostClientID)

		suite.Require().False(clientStore.Has(host.ClientStateKey()))

		for i := uint64(0); i < numCreations; i++ {
			suite.Require().False(clientStore.Has(host.ConsensusStateKey(types.NewHeight(1, i))))
		}
	}
}
