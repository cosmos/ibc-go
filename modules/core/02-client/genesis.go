package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// InitGenesis initializes the ibc client submodule's state from a provided genesis
// state.
func InitGenesis(ctx context.Context, k *keeper.Keeper, gs types.GenesisState) error {
	if err := gs.Params.Validate(); err != nil {
		panic(fmt.Errorf("invalid ibc client genesis state parameters: %v", err))
	}
	k.SetParams(ctx, gs.Params)

	// Set all client metadata first. This will allow client keeper to overwrite client and consensus state keys
	// if clients accidentally write to ClientKeeper reserved keys.
	if len(gs.ClientsMetadata) != 0 {
		k.SetAllClientMetadata(ctx, gs.ClientsMetadata)
	}

	for _, client := range gs.Clients {
		cs, ok := client.ClientState.GetCachedValue().(exported.ClientState)
		if !ok {
			return errors.New("invalid client state")
		}

		if !gs.Params.IsAllowedClient(cs.ClientType()) {
			return fmt.Errorf("client state type %s is not registered on the allowlist", cs.ClientType())
		}

		k.SetClientState(ctx, client.ClientId, cs)
	}

	for _, cs := range gs.ClientsConsensus {
		for _, consState := range cs.ConsensusStates {
			consensusState, ok := consState.ConsensusState.GetCachedValue().(exported.ConsensusState)
			if !ok {
				return fmt.Errorf("invalid consensus state with client ID %s at height %s", cs.ClientId, consState.Height)
			}

			k.SetClientConsensusState(ctx, cs.ClientId, consState.Height, consensusState)
		}
	}

	k.SetNextClientSequence(ctx, gs.NextClientSequence)
	return nil
}

// ExportGenesis returns the ibc client submodule's exported genesis.
// NOTE: the export process is not optimized, it will iterate three
// times over the 02-client sub-store.
func ExportGenesis(ctx context.Context, k *keeper.Keeper) (types.GenesisState, error) {
	genClients := k.GetAllGenesisClients(ctx)
	clientsMetadata, err := k.GetAllClientMetadata(ctx, genClients)
	if err != nil {
		return types.GenesisState{}, err
	}
	return types.GenesisState{
		Clients:          genClients,
		ClientsMetadata:  clientsMetadata,
		ClientsConsensus: k.GetAllConsensusStates(ctx),
		Params:           k.GetParams(ctx),
		// Warning: CreateLocalhost is deprecated
		CreateLocalhost:    false,
		NextClientSequence: k.GetNextClientSequence(ctx),
	}, nil
}
