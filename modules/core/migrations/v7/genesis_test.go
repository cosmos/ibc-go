package v7_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	ibcclient "github.com/cosmos/ibc-go/v10/modules/core/02-client"
	clientv7 "github.com/cosmos/ibc-go/v10/modules/core/02-client/migrations/v7"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/cosmos/ibc-go/v10/modules/core/migrations/v7"
	"github.com/cosmos/ibc-go/v10/modules/core/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type MigrationsV7TestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

// TestMigrationsV7TestSuite runs all the tests within this package.
func TestMigrationsV7TestSuite(t *testing.T) {
	testifysuite.Run(t, new(MigrationsV7TestSuite))
}

// SetupTest creates a coordinator with 2 test chains.
func (s *MigrationsV7TestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

// NOTE: this test is mainly copied from 02-client/migrations/v7/genesis_test.go
func (s *MigrationsV7TestSuite) TestMigrateGenesisSolomachine() {
	// create tendermint clients
	for range 3 {
		path := ibctesting.NewPath(s.chainA, s.chainB)

		path.SetupClients()

		err := path.EndpointA.UpdateClient()
		s.Require().NoError(err)

		// update a second time to add more state
		err = path.EndpointA.UpdateClient()
		s.Require().NoError(err)
	}

	// create multiple legacy solo machine clients
	solomachine := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, ibctesting.DefaultSolomachineClientID, "testing", 1)
	solomachineMulti := ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "06-solomachine-1", "testing", 4)

	clientGenState := ibcclient.ExportGenesis(s.chainA.GetContext(), s.chainA.App.GetIBCKeeper().ClientKeeper)

	// manually generate old proto buf definitions and set in genesis
	// NOTE: we cannot use 'ExportGenesis' for the solo machines since we are
	// using client states and consensus states which do not implement the exported.ClientState
	// and exported.ConsensusState interface
	var clients []clienttypes.IdentifiedClientState
	for _, sm := range []*ibctesting.Solomachine{solomachine, solomachineMulti} {
		clientState := sm.ClientState()

		// generate old client state proto definition
		legacyClientState := &clientv7.ClientState{
			Sequence: clientState.Sequence,
			ConsensusState: &clientv7.ConsensusState{
				PublicKey:   clientState.ConsensusState.PublicKey,
				Diversifier: clientState.ConsensusState.Diversifier,
				Timestamp:   clientState.ConsensusState.Timestamp,
			},
			AllowUpdateAfterProposal: true,
		}

		// set client state
		protoAny, err := codectypes.NewAnyWithValue(legacyClientState)
		s.Require().NoError(err)
		s.Require().NotNil(protoAny)

		clients = append(clients, clienttypes.IdentifiedClientState{
			ClientId:    sm.ClientID,
			ClientState: protoAny,
		})

		// set in store for ease of determining expected genesis
		clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), sm.ClientID)
		cdc, ok := s.chainA.App.AppCodec().(*codec.ProtoCodec)
		s.Require().True(ok)
		clientv7.RegisterInterfaces(cdc.InterfaceRegistry())

		bz, err := cdc.MarshalInterface(legacyClientState)
		s.Require().NoError(err)
		clientStore.Set(host.ClientStateKey(), bz)

		protoAny, err = codectypes.NewAnyWithValue(legacyClientState.ConsensusState)
		s.Require().NoError(err)
		s.Require().NotNil(protoAny)

		// obtain marshalled bytes to set in client store
		bz, err = cdc.MarshalInterface(legacyClientState.ConsensusState)
		s.Require().NoError(err)

		var consensusStates []clienttypes.ConsensusStateWithHeight

		// set consensus states in store and genesis
		for i := range uint64(10) {
			height := clienttypes.NewHeight(1, i)
			clientStore.Set(host.ConsensusStateKey(height), bz)
			consensusStates = append(consensusStates, clienttypes.ConsensusStateWithHeight{
				Height:         height,
				ConsensusState: protoAny,
			})
		}

		clientGenState.ClientsConsensus = append(clientGenState.ClientsConsensus, clienttypes.ClientConsensusStates{
			ClientId:        sm.ClientID,
			ConsensusStates: consensusStates,
		})
	}

	// solo machine clients must come before tendermint in expected
	clientGenState.Clients = append(clients, clientGenState.Clients...)

	// migrate store get expected genesis
	// store migration and genesis migration should produce identical results
	// NOTE: tendermint clients are not pruned in genesis so the test should not have expired tendermint clients
	err := clientv7.MigrateStore(s.chainA.GetContext(), runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ibcexported.StoreKey)), s.chainA.App.AppCodec(), s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
	s.Require().NoError(err)
	expectedClientGenState := ibcclient.ExportGenesis(s.chainA.GetContext(), s.chainA.App.GetIBCKeeper().ClientKeeper)

	cdc, ok := s.chainA.App.AppCodec().(*codec.ProtoCodec)
	s.Require().True(ok)

	// NOTE: these lines are added in comparison to 02-client/migrations/v7/genesis_test.go
	// generate appState with old ibc genesis state
	appState := genutiltypes.AppMap{}
	ibcGenState := types.DefaultGenesisState()
	ibcGenState.ClientGenesis = clientGenState

	// ensure tests pass even if the legacy solo machine is already registered
	clientv7.RegisterInterfaces(cdc.InterfaceRegistry())
	appState[ibcexported.ModuleName] = cdc.MustMarshalJSON(ibcGenState)

	// NOTE: genesis time isn't updated since we aren't testing for tendermint consensus state pruning
	migrated, err := v7.MigrateGenesis(appState, cdc)
	s.Require().NoError(err)

	expectedAppState := genutiltypes.AppMap{}
	expectedIBCGenState := types.DefaultGenesisState()
	expectedIBCGenState.ClientGenesis = expectedClientGenState

	bz, err := cdc.MarshalJSON(expectedIBCGenState)
	s.Require().NoError(err)
	expectedAppState[ibcexported.ModuleName] = bz

	s.Require().Equal(expectedAppState, migrated)
}
