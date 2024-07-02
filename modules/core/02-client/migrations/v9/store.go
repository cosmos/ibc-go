package v9

import (
	"strings"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// MigrateStore performs in-place store migrations from ibc-go v8 to ibc-go v9.
// The migration includes:
//
// - Removing the localhost client state as it is now stateless
func MigrateStore(ctx sdk.Context, clientKeeper ClientKeeper) error {
	handleLocalhostMigration(ctx, clientKeeper)

	return nil
}

func handleLocalhostMigration(ctx sdk.Context, clientKeeper ClientKeeper) {
	clientStore := clientKeeper.ClientStore(ctx, exported.LocalhostClientID)

	// delete the client state
	clientStore.Delete(host.ClientStateKey())

	removeAllClientConsensusStates(clientStore)
}

// removeAllClientConsensusStates removes all client consensus states from the associated
// client store.
func removeAllClientConsensusStates(clientStore storetypes.KVStore) {
	iterator := storetypes.KVStorePrefixIterator(clientStore, []byte(host.KeyConsensusStatePrefix))
	var heights []exported.Height

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")
		// key is in the format "consensusStates/<height>"
		if len(keySplit) != 2 || keySplit[0] != string(host.KeyConsensusStatePrefix) {
			continue
		}

		// collect consensus states to be pruned
		heights = append(heights, clienttypes.MustParseHeight(keySplit[1]))
	}

	// delete all consensus states
	for _, height := range heights {
		clientStore.Delete(host.ConsensusStateKey(height))
	}
}
