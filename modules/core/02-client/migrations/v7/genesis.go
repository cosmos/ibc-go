package v7

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
)

// MigrateGenesis accepts an exported IBC client genesis file and migrates it to:
//
// - Update solo machine client state protobuf definition (v2 to v3)
// - Remove all solo machine consensus states
// - Remove localhost client
func MigrateGenesis(clientGenState *types.GenesisState, cdc codec.BinaryCodec) (*types.GenesisState, error) {
	// To prune the client and consensus states, we will create new slices to fill up
	// with information we want to keep.
	var (
		clientsConsensus []types.ClientConsensusStates
		clients          []types.IdentifiedClientState
	)

	for _, client := range clientGenState.Clients {
		clientType, _, err := types.ParseClientIdentifier(client.ClientId)
		if err != nil {
			return nil, err
		}

		// update solo machine client state defintions
		switch clientType {
		case exported.Solomachine:
			clientState := &ClientState{}
			if err := cdc.Unmarshal(client.ClientState.Value, clientState); err != nil {
				return nil, sdkerrors.Wrap(err, "failed to unmarshal client state bytes into solo machine client state")
			}

			updatedClientState := migrateSolomachine(clientState)

			any, err := types.PackClientState(updatedClientState)
			if err != nil {
				return nil, err
			}

			clients = append(clients, types.IdentifiedClientState{
				ClientId:    client.ClientId,
				ClientState: any,
			})

		case Localhost:
			// remove localhost client state by not adding client state

		default:
			// add all other client states
			clients = append(clients, client)
		}

		// iterate consensus states by client
		for _, clientConsensusStates := range clientGenState.ClientsConsensus {
			// look for consensus states for the current client
			if clientConsensusStates.ClientId == client.ClientId {
				switch clientType {
				case exported.Solomachine:
					// remove all consensus states for the solo machine
					// do not add to new clientsConsensus

				case Localhost:
					// remove all consensus states for the solo machine
					// do not add to new clientsConsensus

				default:
					// ensure all consensus states added for other client types
					clientsConsensus = append(clientsConsensus, clientConsensusStates)
				}
			}
		}
	}

	clientGenState.Clients = clients
	clientGenState.ClientsConsensus = clientsConsensus
	return clientGenState, nil
}
