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
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
)

// MigrateStore performs in-place store migrations from SDK v0.40 of the IBC module to v1.0.0 of ibc-go.
// The migration includes:
//
// - Pruning solo machine clients
// - Pruning expired tendermint consensus states
func MigrateStore(ctx sdk.Context, storeKey sdk.StoreKey, cdc codec.BinaryCodec) (err error) {
	store := ctx.KVStore(storeKey)
	iterator := sdk.KVStorePrefixIterator(store, host.KeyClientStorePrefix)

	var clients []clienttypes.IdentifiedClientState

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")
		if keySplit[len(keySplit)-1] != host.KeyClientState {
			continue
		}

		// key is clients/{clientid}/clientState
		// Thus, keySplit[1] is clientID
		clientID := keySplit[1]
		clientState := types.MustUnmarshalClientState(cdc, iterator.Value())
		clients = append(clients, clienttypes.NewIdentifiedClientState(clientID, clientState))

	}

	for _, client := range clients {
		clientType, _, err := types.ParseClientIdentifier(client.ClientId)
		if err != nil {
			return err
		}

		clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, client.ClientId))
		clientStore := prefix.NewStore(ctx.KVStore(storeKey), clientPrefix)

		switch clientType {
		case exported.Solomachine:
			pruneSolomachine(clientStore)
			store.Delete([]byte(fmt.Sprintf("%s/%s", host.KeyClientStorePrefix, client.ClientId)))

		case exported.Tendermint:
			clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, client.ClientId))
			clientStore := prefix.NewStore(ctx.KVStore(storeKey), clientPrefix)
			clientState, err := types.UnpackClientState(client.ClientState)
			if err != nil {
				return err
			}

			// ensure client is tendermint type
			tmClientState, ok := clientState.(*ibctmtypes.ClientState)
			if !ok {
				panic("client with identifier '07-tendermint' is not tendermint type!")
			}
			if err = ibctmtypes.PruneAllExpiredConsensusStates(ctx, clientStore, cdc, tmClientState); err != nil {
				return err
			}

		default:
			continue
		}
	}

	return nil
}

// pruneSolomachine removes the client state and all consensus states
// stored in the provided clientStore
func pruneSolomachine(clientStore sdk.KVStore) {
	// delete client state
	clientStore.Delete(host.ClientStateKey())

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
}
