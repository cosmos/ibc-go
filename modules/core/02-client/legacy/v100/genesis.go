package v100

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	smtypes "github.com/cosmos/ibc-go/modules/light-clients/06-solomachine/types"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
)

// Migrate accepts exported v0.39 x/auth and v0.38 x/bank genesis state and
// migrates it to v0.40 x/bank genesis state. The migration includes:
//
// - Moving balances from x/auth to x/bank genesis state.
// - Moving supply from x/supply to x/bank genesis state.
// - Re-encode in v0.40 GenesisState.
func Migrate(
	clientGenState *types.GenesisState,
) (*types.GenesisState, error) {
	for i, client := range clientGenState.Clients {
		clientType, _, err := types.ParseClientIdentifier(client.ClientId)
		if err != nil {
			return nil, err
		}

		if clientType == exported.Solomachine {
			// unpack any
			clientState, ok := client.ClientState.GetCachedValue().(*ClientState)
			if !ok {
				return nil, sdkerrors.Wrapf(sdkerrors.ErrUnpackAny, "cannot unpack Any into ClientState %T", client.ClientState)
			}

			isFrozen := clientState.FrozenSequence != 0
			consensusState := &smtypes.ConsensusState{
				PublicKey:   clientState.ConsensusState.PublicKey,
				Diversifier: clientState.ConsensusState.Diversifier,
				Timestamp:   clientState.ConsensusState.Timestamp,
			}

			newSolomachine := &smtypes.ClientState{
				Sequence:                 clientState.Sequence,
				IsFrozen:                 isFrozen,
				ConsensusState:           consensusState,
				AllowUpdateAfterProposal: clientState.AllowUpdateAfterProposal,
			}

			any, err := types.PackClientState(newSolomachine)
			if err != nil {
				return nil, err
			}

			clientGenState.Clients[i] = types.IdentifiedClientState{
				ClientId:    client.ClientId,
				ClientState: any,
			}
		}

		var smIndiciesToRemove []int
		for i, clientConsensusState := range clientGenState.ClientsConsensus {
			// found consensus state, prune as necessary
			if clientConsensusState.ClientId == client.ClientId {
				switch clientType {
				case exported.Solomachine:
					// remove all consensus states for the solo machine
					smIndiciesToRemove = append(smIndiciesToRemove, i)
				case exported.Tendermint:
					// prune expired consensus state
					tmClientState, ok := client.ClientState.GetCachedValue().(*ibctmtypes.ClientState)
					if !ok {
						return nil, clienttypes.Err
					}

					var consStateIndiciesToRemove []int
					for i, consState := range clientConsensusState.ConsensusStates {
						tmConsState := consState.ConsensusState.GetCachedValue().(*ibctmtypes.ConsensusState)
						if tmClientState.IsExpired(tmConsState.Timestamp, blockTime) {
							consStateIndiciesToRemove = append(consStateIndiciesToRemove, i)
						}
					}

					for _, index := range consStateIndiciesToRemove {
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

		for _, index := range smIndiciesToRemove {
			clientGenState.ClientsConsensus = append(clientGenState.ClientsConsensus[:index], clientGenState.ClientsConsensus[index+1:]...)
		}

	}

	return clientGenState, nil
}
