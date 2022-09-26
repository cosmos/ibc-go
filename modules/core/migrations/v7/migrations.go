package v7

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clientkeeper "github.com/cosmos/ibc-go/v6/modules/core/02-client/keeper"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
)

// Localhost is the client type for a localhost client. It is also used as the clientID
// for the localhost client.
const Localhost string = "09-localhost"

// MigrateToV7 prunes the 09-Localhost client and associated consensus states from the ibc store
func MigrateToV7(ctx sdk.Context, clientKeeper clientkeeper.Keeper) {
	clientStore := clientKeeper.ClientStore(ctx, Localhost)

	iterator := sdk.KVStorePrefixIterator(clientStore, []byte(host.KeyConsensusStatePrefix))
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

	// delete the client state
	clientStore.Delete(host.ClientStateKey())
}
