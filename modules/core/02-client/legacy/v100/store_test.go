package v100_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/modules/core/02-client/legacy/v100"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

type LegacyTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

// TestLegacyTestSuite runs all the tests within this package.
func TestLegacyTestSuite(t *testing.T) {
	suite.Run(t, new(LegacyTestSuite))
}

// SetupTest creates a coordinator with 2 test chains.
func (suite *LegacyTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
}

// only test migration for solo machines
// ensure all client states are migrated and all consensus states
// are removed
func (suite *LegacyTestSuite) TestMigrateStoreSolomachine() {

}

// only test migration for tendermint clients
// ensure all expired consensus states are removed from tendermint client stores
func (suite *LegacyTestSuite) TestMigrateStoreTendermint() {
	// create path and setup clients
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupClients(path)

	// collect all heights expected to be pruned
	var pruneHeights []exported.Height
	pruneHeights = append(pruneHeights, path.EndpointA.GetClientState().GetLatestHeight())

	// these heights will be expired and also pruned
	for i := 0; i < 3; i++ {
		path.EndpointA.UpdateClient()
		pruneHeights = append(pruneHeights, path.EndpointA.GetClientState().GetLatestHeight())
	}

	// double chedck all information is currently stored
	for _, pruneHeight := range pruneHeights {
		consState, ok := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
		suite.Require().True(ok)
		suite.Require().NotNil(consState)

		ctx := path.EndpointA.Chain.GetContext()
		clientStore := path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)

		processedTime, ok := ibctmtypes.GetProcessedTime(clientStore, pruneHeight)
		suite.Require().True(ok)
		suite.Require().NotNil(processedTime)

		processedHeight, ok := ibctmtypes.GetProcessedHeight(clientStore, pruneHeight)
		suite.Require().True(ok)
		suite.Require().NotNil(processedHeight)

		expectedConsKey := ibctmtypes.GetIterationKey(clientStore, pruneHeight)
		suite.Require().NotNil(expectedConsKey)
	}

	// Increment the time by a week
	suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

	// create the consensus state that can be used as trusted height for next update
	path.EndpointA.UpdateClient()

	// Increment the time by another week, then update the client.
	// This will cause the consensus states created before the first time increment
	// to be expired
	suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
	err := v100.MigrateStore(path.EndpointA.Chain.GetContext(), path.EndpointA.Chain.GetSimApp().GetKey(host.StoreKey), path.EndpointA.Chain.App.AppCodec())
	suite.Require().NoError(err)

	ctx := path.EndpointA.Chain.GetContext()
	clientStore := path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)

	// ensure everything has been pruned
	for i, pruneHeight := range pruneHeights {
		consState, ok := path.EndpointA.Chain.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
		suite.Require().False(ok, i)
		suite.Require().Nil(consState, i)

		processedTime, ok := ibctmtypes.GetProcessedTime(clientStore, pruneHeight)
		suite.Require().False(ok, i)
		suite.Require().Equal(uint64(0), processedTime, i)

		processedHeight, ok := ibctmtypes.GetProcessedHeight(clientStore, pruneHeight)
		suite.Require().False(ok, i)
		suite.Require().Nil(processedHeight, i)

		expectedConsKey := ibctmtypes.GetIterationKey(clientStore, pruneHeight)
		suite.Require().Nil(expectedConsKey, i)
	}
}
