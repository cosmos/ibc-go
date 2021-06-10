package v100_test

import (
	"bytes"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/stretchr/testify/suite"
	tmtypes "github.com/tendermint/tendermint/types"

	ibcclient "github.com/cosmos/ibc-go/modules/core/02-client"
	clientv100 "github.com/cosmos/ibc-go/modules/core/02-client/legacy/v100"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/legacy/v100"
	"github.com/cosmos/ibc-go/modules/core/types"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
	"github.com/cosmos/ibc-go/testing/simapp"
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

// NOTE: this test is mainly copied from 02-client/legacy/v100
func (suite *LegacyTestSuite) TestMigrateGenesisSolomachine() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	encodingConfig := simapp.MakeTestEncodingConfig()
	clientCtx := client.Context{}.
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithJSONCodec(encodingConfig.Marshaler)

	// create multiple legacy solo machine clients
	solomachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "06-solomachine-0", "testing", 1)
	solomachineMulti := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "06-solomachine-1", "testing", 4)

	// create tendermint clients
	suite.coordinator.SetupClients(path)
	clientGenState := ibcclient.ExportGenesis(path.EndpointA.Chain.GetContext(), path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper)

	// manually generate old proto buf definitions and set in genesis
	// NOTE: we cannot use 'ExportGenesis' for the solo machines since we are
	// using client states and consensus states which do not implement the exported.ClientState
	// and exported.ConsensusState interface
	var clients []clienttypes.IdentifiedClientState
	for _, sm := range []*ibctesting.Solomachine{solomachine, solomachineMulti} {
		clientState := sm.ClientState()

		var seq uint64
		if clientState.IsFrozen {
			seq = 1
		}

		// generate old client state proto defintion
		legacyClientState := &clientv100.ClientState{
			Sequence:       clientState.Sequence,
			FrozenSequence: seq,
			ConsensusState: &clientv100.ConsensusState{
				PublicKey:   clientState.ConsensusState.PublicKey,
				Diversifier: clientState.ConsensusState.Diversifier,
				Timestamp:   clientState.ConsensusState.Timestamp,
			},
			AllowUpdateAfterProposal: clientState.AllowUpdateAfterProposal,
		}

		// set client state
		any, err := codectypes.NewAnyWithValue(legacyClientState)
		suite.Require().NoError(err)
		suite.Require().NotNil(any)
		client := clienttypes.IdentifiedClientState{
			ClientId:    sm.ClientID,
			ClientState: any,
		}
		clients = append(clients, client)

		// set in store for ease of determining expected genesis
		clientStore := path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper.ClientStore(path.EndpointA.Chain.GetContext(), sm.ClientID)
		bz, err := path.EndpointA.Chain.App.AppCodec().MarshalInterface(legacyClientState)
		suite.Require().NoError(err)
		clientStore.Set(host.ClientStateKey(), bz)

		// set some consensus states
		height1 := clienttypes.NewHeight(0, 1)
		height2 := clienttypes.NewHeight(1, 2)
		height3 := clienttypes.NewHeight(0, 123)

		any, err = codectypes.NewAnyWithValue(legacyClientState.ConsensusState)
		suite.Require().NoError(err)
		suite.Require().NotNil(any)
		consensusState1 := clienttypes.ConsensusStateWithHeight{
			Height:         height1,
			ConsensusState: any,
		}
		consensusState2 := clienttypes.ConsensusStateWithHeight{
			Height:         height2,
			ConsensusState: any,
		}
		consensusState3 := clienttypes.ConsensusStateWithHeight{
			Height:         height3,
			ConsensusState: any,
		}

		clientConsensusState := clienttypes.ClientConsensusStates{
			ClientId:        sm.ClientID,
			ConsensusStates: []clienttypes.ConsensusStateWithHeight{consensusState1, consensusState2, consensusState3},
		}

		clientGenState.ClientsConsensus = append(clientGenState.ClientsConsensus, clientConsensusState)

		// set in store for ease of determining expected genesis
		bz, err = path.EndpointA.Chain.App.AppCodec().MarshalInterface(legacyClientState.ConsensusState)
		suite.Require().NoError(err)
		clientStore.Set(host.ConsensusStateKey(height1), bz)
		clientStore.Set(host.ConsensusStateKey(height2), bz)
		clientStore.Set(host.ConsensusStateKey(height3), bz)
	}
	// solo machine clients must come before tendermint in expected
	clientGenState.Clients = append(clients, clientGenState.Clients...)

	// migrate store get expected genesis
	// store migration and genesis migration should produce identical results
	err := clientv100.MigrateStore(path.EndpointA.Chain.GetContext(), path.EndpointA.Chain.GetSimApp().GetKey(host.StoreKey), path.EndpointA.Chain.App.AppCodec())
	suite.Require().NoError(err)
	expectedClientGenState := ibcclient.ExportGenesis(path.EndpointA.Chain.GetContext(), path.EndpointA.Chain.App.GetIBCKeeper().ClientKeeper)

	// 'ExportGenesis' order metadata keys by processedheight, processedtime for all heights, then it appends all iteration keys
	// In order to match the genesis migration with export genesis we must reorder the iteration keys to be last
	// This isn't ideal, but it is better than modifying the genesis migration from a previous version to match the export genesis of a new version
	// which provides no benefit except nicer testing
	for i, clientMetadata := range expectedClientGenState.ClientsMetadata {
		var updatedMetadata []clienttypes.GenesisMetadata
		var iterationKeys []clienttypes.GenesisMetadata
		for _, metadata := range clientMetadata.ClientMetadata {
			if bytes.HasPrefix(metadata.Key, []byte(ibctmtypes.KeyIterateConsensusStatePrefix)) {
				iterationKeys = append(iterationKeys, metadata)
			} else {
				updatedMetadata = append(updatedMetadata, metadata)
			}
		}
		updatedMetadata = append(updatedMetadata, iterationKeys...)
		expectedClientGenState.ClientsMetadata[i] = clienttypes.IdentifiedGenesisMetadata{
			ClientId:       clientMetadata.ClientId,
			ClientMetadata: updatedMetadata,
		}
	}

	// NOTE: these lines are added in comparison to 02-client/legacy/v100
	// generate appState with old ibc genesis state
	appState := genutiltypes.AppMap{}
	ibcGenState := types.DefaultGenesisState()
	ibcGenState.ClientGenesis = clientGenState
	clientv100.RegisterInterfaces(clientCtx.InterfaceRegistry)
	appState[host.ModuleName] = clientCtx.JSONCodec.MustMarshalJSON(ibcGenState)
	genDoc := tmtypes.GenesisDoc{
		ChainID:       suite.chainA.ChainID,
		GenesisTime:   suite.coordinator.CurrentTime,
		InitialHeight: suite.chainA.GetContext().BlockHeight(),
	}

	// NOTE: genesis time isn't updated since we aren't testing for tendermint consensus state pruning
	migrated, err := v100.MigrateGenesis(appState, clientCtx, genDoc)
	suite.Require().NoError(err)

	expectedAppState := genutiltypes.AppMap{}
	expectedIBCGenState := types.DefaultGenesisState()
	expectedIBCGenState.ClientGenesis = expectedClientGenState

	bz, err := clientCtx.JSONCodec.MarshalJSON(expectedIBCGenState)
	suite.Require().NoError(err)
	expectedAppState[host.ModuleName] = bz

	suite.Require().Equal(expectedAppState, migrated)
}
