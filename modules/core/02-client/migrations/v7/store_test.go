package v7_test

import (
	"fmt"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/migrations/v7"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// numCreations is the number of clients/consensus states created for
// solo machine and localhost clients
const numCreations = 10

type MigrationsV7TestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func TestIBCTestSuite(t *testing.T) {
	testifysuite.Run(t, new(MigrationsV7TestSuite))
}

func (s *MigrationsV7TestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

// create multiple solo machine clients, tendermint and localhost clients
// ensure that solo machine clients are migrated and their consensus states are removed
// ensure the localhost is deleted entirely.
func (s *MigrationsV7TestSuite) TestMigrateStore() {
	paths := []*ibctesting.Path{
		ibctesting.NewPath(s.chainA, s.chainB),
		ibctesting.NewPath(s.chainA, s.chainB),
	}

	// create tendermint clients
	for _, path := range paths {
		path.SetupClients()
	}

	solomachines := []*ibctesting.Solomachine{
		ibctesting.NewSolomachine(s.T(), s.chainA.Codec, ibctesting.DefaultSolomachineClientID, "testing", 1),
		ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "06-solomachine-1", "testing", 4),
	}

	s.createSolomachineClients(solomachines)
	s.createLocalhostClients()

	err := v7.MigrateStore(s.chainA.GetContext(), runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ibcexported.StoreKey)), s.chainA.App.AppCodec(), s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
	s.Require().NoError(err)

	s.assertSolomachineClients(solomachines)
	s.assertNoLocalhostClients()
}

func (s *MigrationsV7TestSuite) TestMigrateStoreNoTendermintClients() {
	solomachines := []*ibctesting.Solomachine{
		ibctesting.NewSolomachine(s.T(), s.chainA.Codec, ibctesting.DefaultSolomachineClientID, "testing", 1),
		ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "06-solomachine-1", "testing", 4),
	}

	s.createSolomachineClients(solomachines)
	s.createLocalhostClients()

	err := v7.MigrateStore(s.chainA.GetContext(), runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ibcexported.StoreKey)), s.chainA.App.AppCodec(), s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
	s.Require().NoError(err)

	s.assertSolomachineClients(solomachines)
	s.assertNoLocalhostClients()
}

func (s *MigrationsV7TestSuite) createSolomachineClients(solomachines []*ibctesting.Solomachine) {
	// manually generate old protobuf definitions and set in store
	// NOTE: we cannot use 'CreateClient' and 'UpdateClient' functions since we are
	// using client states and consensus states which do not implement the exported.ClientState
	// and exported.ConsensusState interface
	for _, sm := range solomachines {
		clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), sm.ClientID)
		clientState := sm.ClientState()

		// generate old client state proto definition
		legacyClientState := &v7.ClientState{
			Sequence: clientState.Sequence,
			ConsensusState: &v7.ConsensusState{
				PublicKey:   clientState.ConsensusState.PublicKey,
				Diversifier: clientState.ConsensusState.Diversifier,
				Timestamp:   clientState.ConsensusState.Timestamp,
			},
			AllowUpdateAfterProposal: true,
		}

		cdc, ok := s.chainA.App.AppCodec().(*codec.ProtoCodec)
		s.Require().True(ok)
		v7.RegisterInterfaces(cdc.InterfaceRegistry())

		bz, err := cdc.MarshalInterface(legacyClientState)
		s.Require().NoError(err)
		clientStore.Set(host.ClientStateKey(), bz)

		bz, err = cdc.MarshalInterface(legacyClientState.ConsensusState)
		s.Require().NoError(err)

		// set some consensus states
		for i := range numCreations {
			height := types.NewHeight(1, uint64(i))
			clientStore.Set(host.ConsensusStateKey(height), bz)
		}
	}
}

func (s *MigrationsV7TestSuite) assertSolomachineClients(solomachines []*ibctesting.Solomachine) {
	// verify client state has been migrated
	for _, sm := range solomachines {
		clientState, ok := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientState(s.chainA.GetContext(), sm.ClientID)
		s.Require().True(ok)
		s.Require().Equal(sm.ClientState(), clientState)

		for i := range numCreations {
			height := types.NewHeight(1, uint64(i))

			consState, ok := s.chainA.App.GetIBCKeeper().ClientKeeper.GetClientConsensusState(s.chainA.GetContext(), sm.ClientID, height)
			s.Require().False(ok)
			s.Require().Empty(consState)
		}
	}
}

// createLocalhostClients clients creates multiple localhost clients and multiple consensus states for each
func (s *MigrationsV7TestSuite) createLocalhostClients() {
	for numClients := range numCreations {
		clientID := fmt.Sprintf("%s-%d", v7.Localhost, numClients)
		clientStore := s.chainA.GetSimApp().IBCKeeper.ClientKeeper.ClientStore(s.chainA.GetContext(), clientID)

		clientStore.Set(host.ClientStateKey(), []byte("clientState"))

		for i := range numCreations {
			clientStore.Set(host.ConsensusStateKey(types.NewHeight(1, uint64(i))), []byte("consensusState"))
		}
	}
}

// assertNoLocalhostClients asserts that all localhost information has been deleted
func (s *MigrationsV7TestSuite) assertNoLocalhostClients() {
	for numClients := range numCreations {
		clientID := fmt.Sprintf("%s-%d", v7.Localhost, numClients)
		clientStore := s.chainA.GetSimApp().IBCKeeper.ClientKeeper.ClientStore(s.chainA.GetContext(), clientID)

		s.Require().False(clientStore.Has(host.ClientStateKey()))

		for i := range numCreations {
			s.Require().False(clientStore.Has(host.ConsensusStateKey(types.NewHeight(1, uint64(i)))))
		}
	}
}
