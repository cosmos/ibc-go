package migrations_test

import (
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctmmigrations "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint/migrations"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type MigrationsTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (s *MigrationsTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestTendermintTestSuite(t *testing.T) {
	testifysuite.Run(t, new(MigrationsTestSuite))
}

// test pruning of multiple expired tendermint consensus states
func (s *MigrationsTestSuite) TestPruneExpiredConsensusStates() {
	// create multiple tendermint clients and a solo machine client
	// the solo machine is used to verify this pruning function only modifies
	// the tendermint store.

	numTMClients := 3
	paths := make([]*ibctesting.Path, numTMClients)

	for i := range numTMClients {
		path := ibctesting.NewPath(s.chainA, s.chainB)
		path.SetupClients()

		paths[i] = path
	}

	solomachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, ibctesting.DefaultSolomachineClientID, "testing", 1)
	smClientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), solomachine.ClientID)

	// set client state
	bz, err := s.chainA.App.AppCodec().MarshalInterface(solomachine.ClientState())
	s.Require().NoError(err)
	smClientStore.Set(host.ClientStateKey(), bz)

	bz, err = s.chainA.App.AppCodec().MarshalInterface(solomachine.ConsensusState())
	s.Require().NoError(err)
	smHeight := clienttypes.NewHeight(0, 1)
	smClientStore.Set(host.ConsensusStateKey(smHeight), bz)

	pruneHeightMap := make(map[*ibctesting.Path][]exported.Height)
	unexpiredHeightMap := make(map[*ibctesting.Path][]exported.Height)

	for _, path := range paths {
		// collect all heights expected to be pruned
		var pruneHeights []exported.Height
		pruneHeights = append(pruneHeights, path.EndpointA.GetClientLatestHeight())

		// these heights will be expired and also pruned
		for range 3 {
			err := path.EndpointA.UpdateClient()
			s.Require().NoError(err)

			pruneHeights = append(pruneHeights, path.EndpointA.GetClientLatestHeight())
		}

		// double chedck all information is currently stored
		for _, pruneHeight := range pruneHeights {
			consState, ok := s.chainA.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
			s.Require().True(ok)
			s.Require().NotNil(consState)

			ctx := s.chainA.GetContext()
			clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)

			processedTime, ok := ibctm.GetProcessedTime(clientStore, pruneHeight)
			s.Require().True(ok)
			s.Require().NotNil(processedTime)

			processedHeight, ok := ibctm.GetProcessedHeight(clientStore, pruneHeight)
			s.Require().True(ok)
			s.Require().NotNil(processedHeight)

			expectedConsKey := ibctm.GetIterationKey(clientStore, pruneHeight)
			s.Require().NotNil(expectedConsKey)
		}
		pruneHeightMap[path] = pruneHeights
	}

	// Increment the time by a week
	s.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

	for _, path := range paths {
		// create the consensus state that can be used as trusted height for next update
		var unexpiredHeights []exported.Height
		err := path.EndpointA.UpdateClient()
		s.Require().NoError(err)
		unexpiredHeights = append(unexpiredHeights, path.EndpointA.GetClientLatestHeight())

		err = path.EndpointA.UpdateClient()
		s.Require().NoError(err)
		unexpiredHeights = append(unexpiredHeights, path.EndpointA.GetClientLatestHeight())

		unexpiredHeightMap[path] = unexpiredHeights
	}

	// Increment the time by another week, then update the client.
	// This will cause the consensus states created before the first time increment
	// to be expired
	s.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
	totalPruned, err := ibctmmigrations.PruneExpiredConsensusStates(s.chainA.GetContext(), s.chainA.App.AppCodec(), s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
	s.Require().NoError(err)
	s.Require().NotZero(totalPruned)

	for _, path := range paths {
		ctx := s.chainA.GetContext()
		clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)

		// ensure everything has been pruned
		for i, pruneHeight := range pruneHeightMap[path] {
			consState, ok := s.chainA.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
			s.Require().False(ok, i)
			s.Require().Nil(consState, i)

			processedTime, ok := ibctm.GetProcessedTime(clientStore, pruneHeight)
			s.Require().False(ok, i)
			s.Require().Equal(uint64(0), processedTime, i)

			processedHeight, ok := ibctm.GetProcessedHeight(clientStore, pruneHeight)
			s.Require().False(ok, i)
			s.Require().Nil(processedHeight, i)

			expectedConsKey := ibctm.GetIterationKey(clientStore, pruneHeight)
			s.Require().Nil(expectedConsKey, i)
		}

		// ensure metadata is set for unexpired consensus state
		for _, height := range unexpiredHeightMap[path] {
			consState, ok := s.chainA.GetConsensusState(path.EndpointA.ClientID, height)
			s.Require().True(ok)
			s.Require().NotNil(consState)

			processedTime, ok := ibctm.GetProcessedTime(clientStore, height)
			s.Require().True(ok)
			s.Require().NotEqual(uint64(0), processedTime)

			processedHeight, ok := ibctm.GetProcessedHeight(clientStore, height)
			s.Require().True(ok)
			s.Require().NotEqual(clienttypes.ZeroHeight(), processedHeight)

			consKey := ibctm.GetIterationKey(clientStore, height)
			s.Require().Equal(host.ConsensusStateKey(height), consKey)
		}
	}

	// verify that solomachine client and consensus state were not removed
	smClientStore = s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), solomachine.ClientID)
	bz = smClientStore.Get(host.ClientStateKey())
	s.Require().NotEmpty(bz)

	bz = smClientStore.Get(host.ConsensusStateKey(smHeight))
	s.Require().NotEmpty(bz)
}
