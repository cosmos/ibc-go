package v7_test

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	ibcclient "github.com/cosmos/ibc-go/v10/modules/core/02-client"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/migrations/v7"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

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
	var clients []types.IdentifiedClientState
	for _, sm := range []*ibctesting.Solomachine{solomachine, solomachineMulti} {
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

		// set client state
		protoAny, err := codectypes.NewAnyWithValue(legacyClientState)
		s.Require().NoError(err)
		s.Require().NotNil(protoAny)

		clients = append(clients, types.IdentifiedClientState{
			ClientId:    sm.ClientID,
			ClientState: protoAny,
		})

		// set in store for ease of determining expected genesis
		clientStore := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), sm.ClientID)

		cdc, ok := s.chainA.App.AppCodec().(*codec.ProtoCodec)
		s.Require().True(ok)
		v7.RegisterInterfaces(cdc.InterfaceRegistry())

		bz, err := cdc.MarshalInterface(legacyClientState)
		s.Require().NoError(err)
		clientStore.Set(host.ClientStateKey(), bz)

		protoAny, err = codectypes.NewAnyWithValue(legacyClientState.ConsensusState)
		s.Require().NoError(err)
		s.Require().NotNil(protoAny)

		// obtain marshalled bytes to set in client store
		bz, err = cdc.MarshalInterface(legacyClientState.ConsensusState)
		s.Require().NoError(err)

		var consensusStates []types.ConsensusStateWithHeight

		// set consensus states in store and genesis
		for i := range numCreations {
			height := types.NewHeight(1, uint64(i))
			clientStore.Set(host.ConsensusStateKey(height), bz)
			consensusStates = append(consensusStates, types.ConsensusStateWithHeight{
				Height:         height,
				ConsensusState: protoAny,
			})
		}

		clientGenState.ClientsConsensus = append(clientGenState.ClientsConsensus, types.ClientConsensusStates{
			ClientId:        sm.ClientID,
			ConsensusStates: consensusStates,
		})
	}

	// solo machine clients must come before tendermint in expected
	clientGenState.Clients = append(clients, clientGenState.Clients...)

	// migrate store get expected genesis
	// store migration and genesis migration should produce identical results
	// NOTE: tendermint clients are not pruned in genesis so the test should not have expired tendermint clients
	err := v7.MigrateStore(s.chainA.GetContext(), runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ibcexported.StoreKey)), s.chainA.App.AppCodec(), s.chainA.GetSimApp().IBCKeeper.ClientKeeper)
	s.Require().NoError(err)
	expectedClientGenState := ibcclient.ExportGenesis(s.chainA.GetContext(), s.chainA.App.GetIBCKeeper().ClientKeeper)

	cdc, ok := s.chainA.App.AppCodec().(codec.ProtoCodecMarshaler)
	s.Require().True(ok)

	migrated, err := v7.MigrateGenesis(&clientGenState, cdc)
	s.Require().NoError(err)

	bz, err := cdc.MarshalJSON(&expectedClientGenState)
	s.Require().NoError(err)

	// Indent the JSON bz correctly.
	var jsonObj map[string]any
	err = json.Unmarshal(bz, &jsonObj)
	s.Require().NoError(err)
	expectedIndentedBz, err := json.MarshalIndent(jsonObj, "", "\t")
	s.Require().NoError(err)

	bz, err = cdc.MarshalJSON(migrated)
	s.Require().NoError(err)

	// Indent the JSON bz correctly.
	err = json.Unmarshal(bz, &jsonObj)
	s.Require().NoError(err)
	indentedBz, err := json.MarshalIndent(jsonObj, "", "\t")
	s.Require().NoError(err)

	s.Require().Equal(string(expectedIndentedBz), string(indentedBz))
}
