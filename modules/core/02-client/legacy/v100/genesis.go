package v100

import (
	"bytes"
	"time"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
)

// Migrate accepts exported v1.0.0 IBC client genesis file and migrates it to:
//
// - Update solo machine client state protobuf definition (v1 to v2)
// - Remove all solo machine consensus states
// - Remove all expired tendermint consensus states
func Migrate(clientGenState *types.GenesisState, genesisBlockTime time.Time) (*types.GenesisState, error) {

	for i, client := range clientGenState.Clients {
		clientType, _, err := types.ParseClientIdentifier(client.ClientId)
		if err != nil {
			return nil, err
		}

		// update solo machine client state defintions
		if clientType == exported.Solomachine {
			clientState, ok := client.ClientState.GetCachedValue().(*ClientState)
			if !ok {
				return nil, sdkerrors.Wrapf(sdkerrors.ErrUnpackAny, "cannot unpack Any into ClientState %T", client.ClientState)
			}

			updatedClientState := migrateSolomachine(clientState)

			any, err := types.PackClientState(updatedClientState)
			if err != nil {
				return nil, err
			}

			clientGenState.Clients[i] = types.IdentifiedClientState{
				ClientId:    client.ClientId,
				ClientState: any,
			}
		}

		// collect the client consensus state index for solo machine clients
		var smConsStateByIndex []int

		// iterate consensus states
		for i, clientConsensusState := range clientGenState.ClientsConsensus {
			// look for consensus states for the current client
			if clientConsensusState.ClientId == client.ClientId {
				switch clientType {
				case exported.Solomachine:
					// remove all consensus states for the solo machine
					smConsStateByIndex = append(smConsStateByIndex, i)
				case exported.Tendermint:
					// prune expired consensus state
					tmClientState, ok := client.ClientState.GetCachedValue().(*ibctmtypes.ClientState)
					if !ok {
						return nil, types.ErrInvalidClient
					}

					// collect the consensus state index for expired tendermint consensus states
					var tmConsStateByIndex []int

					for i, consState := range clientConsensusState.ConsensusStates {
						tmConsState := consState.ConsensusState.GetCachedValue().(*ibctmtypes.ConsensusState)
						if tmClientState.IsExpired(tmConsState.Timestamp, genesisBlockTime) {
							tmConsStateByIndex = append(tmConsStateByIndex, i)
						}
					}

					// remove all expired tendermint consensus states
					for _, index := range tmConsStateByIndex {
						for i, identifiedGenMetadata := range clientGenState.ClientsMetadata {
							// look for metadata for current client
							if identifiedGenMetadata.ClientId == client.ClientId {

								// collect the metadata indicies to be removed
								var tmConsMetadataByIndex []int

								// obtain height for consensus state being pruned
								height := clientConsensusState.ConsensusStates[index].Height

								// iterate throught metadata and find metadata which should be pruned
								for j, metadata := range identifiedGenMetadata.ClientMetadata {
									if bytes.Equal(metadata.Key, ibctmtypes.IterationKey(height)) ||
										bytes.Equal(metadata.Key, ibctmtypes.ProcessedTimeKey(height)) ||
										bytes.Equal(metadata.Key, ibctmtypes.ProcessedHeightKey(height)) {
										tmConsMetadataByIndex = append(tmConsMetadataByIndex, j)
									}
								}

								for _, metadataIndex := range tmConsMetadataByIndex {
									clientGenState.ClientsMetadata[i].ClientMetadata = append(clientGenState.ClientsMetadata[i].ClientMetadata[:metadataIndex], clientGenState.ClientsMetadata[i].ClientMetadata[metadataIndex+1:]...)
								}
							}
						}

						// remove client state
						clientGenState.ClientsConsensus[i] = types.ClientConsensusStates{
							ClientId:        clientConsensusState.ClientId,
							ConsensusStates: append(clientConsensusState.ConsensusStates[:index], clientConsensusState.ConsensusStates[index+1:]...),
						}
					}

				default:
					break
				}
			}
		}

		// remove all solo machine consensus states
		for _, index := range smConsStateByIndex {
			clientGenState.ClientsConsensus = append(clientGenState.ClientsConsensus[:index], clientGenState.ClientsConsensus[index+1:]...)
		}
	}

	return clientGenState, nil
}
