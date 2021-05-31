package v100

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/core/02-client/types"
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

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")
		if keySplit[len(keySplit)-1] != host.KeyClientState {
			continue
		}

		// key is ibc/{clientid}/clientState
		// Thus, keySplit[1] is clientID
		clientID := keySplit[1]
		clientState := types.MustUnmarshalClientState(cdc, iterator.Value())

		clientType, _, err := types.ParseClientIdentifier(clientID)
		if err != nil {
			return err
		}

		switch clientType {
		case exported.Solomachine:
			store.Delete([]byte(fmt.Sprintf("%s/%s", host.KeyClientStorePrefix, clientID)))

		case exported.Tendermint:
			clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
			clientStore := prefix.NewStore(ctx.KVStore(storeKey), clientPrefix)

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
