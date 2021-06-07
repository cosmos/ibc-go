package v100

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/core/02-client/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	smtypes "github.com/cosmos/ibc-go/modules/light-clients/06-solomachine/types"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
)

// MigrateStore performs in-place store migrations from SDK v0.40 of the IBC module to v1.0.0 of ibc-go.
// The migration includes:
//
// - Migrating solo machine client states from v1 to v2 protobuf definition
// - Pruning all solo machine consensus states from the client stores
// - Pruning expired tendermint consensus states
func MigrateStore(ctx sdk.Context, storeKey sdk.StoreKey, cdc codec.BinaryCodec) (err error) {
	store := ctx.KVStore(storeKey)
	iterator := sdk.KVStorePrefixIterator(store, host.KeyClientStorePrefix)

	var clients []string

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")
		if keySplit[len(keySplit)-1] != host.KeyClientState {
			continue
		}

		// key is clients/{clientid}/clientState
		// Thus, keySplit[1] is clientID
		clients = append(clients, keySplit[1])

	}

	for _, clientID := range clients {
		clientType, _, err := types.ParseClientIdentifier(clientID)
		if err != nil {
			return err
		}

		clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
		clientStore := prefix.NewStore(ctx.KVStore(storeKey), clientPrefix)

		switch clientType {
		case exported.Solomachine:
			migrateSolomachine(clientStore, cdc, clientID)

		case exported.Tendermint:
			clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
			clientStore := prefix.NewStore(ctx.KVStore(storeKey), clientPrefix)
			// get tendermint client state from client store
			bz := clientStore.Get(host.ClientStateKey())
			if bz == nil {
				return clienttypes.ErrClientNotFound
			}

			var clientState *ibctmtypes.ClientState
			if err := cdc.UnmarshalInterface(bz, &clientState); err != nil {
				return err
			}

			if err = ibctmtypes.PruneAllExpiredConsensusStates(ctx, clientStore, cdc, clientState); err != nil {
				return err
			}

		default:
			continue
		}
	}

	return nil
}

// migrateSolomachine migrates the solomachine from v1 to v2 solo machine protobuf defintion.
// It also deletes all consensus states stored in the client store as they are not necessary
// and reference the v1 consensus state type.
func migrateSolomachine(clientStore sdk.KVStore, cdc codec.BinaryCodec, clientID string) error {
	// get legacy solo machine from client store
	bz := clientStore.Get(host.ClientStateKey())
	if bz == nil {
		return clienttypes.ErrClientNotFound
	}

	var clientState *ClientState
	if err := cdc.UnmarshalInterface(bz, &clientState); err != nil {
		return err
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

	bz, err := clienttypes.MarshalClientState(cdc, newSolomachine)
	if err != nil {
		return err
	}

	// update solomachine in store
	clientStore.Set(host.ClientStateKey(), bz)

	// collect consensus states to be pruned
	iterator := sdk.KVStorePrefixIterator(clientStore, []byte(host.KeyConsensusStatePrefix))
	var heights []exported.Height

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")
		// key is in the format "clients/<clientID>/consensusStates/<height>"
		if len(keySplit) != 4 || keySplit[2] != string(host.KeyConsensusStatePrefix) {
			continue
		}
		heights = append(heights, types.MustParseHeight(keySplit[3]))
	}

	// delete all consensus states
	for _, height := range heights {
		clientStore.Delete(host.ConsensusStateKey(height))
	}

	return nil
}
