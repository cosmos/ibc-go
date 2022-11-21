package migrations

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint"
)

// PruneTendermintConsensusStates prunes all expired tendermint consensus states. This function
// may optionally be called during in-place store migrations. The ibc store key must be provided.
func PruneTendermintConsensusStates(ctx sdk.Context, cdc codec.BinaryCodec, storeKey storetypes.StoreKey) error {
	store := ctx.KVStore(storeKey)

	// iterate over ibc store with prefix: clients/07-tendermint,
	tendermintClientPrefix := []byte(fmt.Sprintf("%s/%s", host.KeyClientStorePrefix, exported.Tendermint))
	iterator := sdk.KVStorePrefixIterator(store, tendermintClientPrefix)

	var clients []string

	// collect all clients to avoid performing store state changes during iteration
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		path := string(iterator.Key())
		if !strings.Contains(path, host.KeyClientState) {
			// skip non client state keys
			continue
		}

		clientID, err := host.ParseClientStatePath(path)
		if err != nil {
			return err
		}

		clients = append(clients, clientID)
	}

	var totalPruned int

	for _, clientID := range clients {
		clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
		clientStore := prefix.NewStore(ctx.KVStore(storeKey), clientPrefix)

		bz := clientStore.Get(host.ClientStateKey())
		if bz == nil {
			return types.ErrClientNotFound
		}

		var clientState exported.ClientState
		if err := cdc.UnmarshalInterface(bz, &clientState); err != nil {
			return sdkerrors.Wrap(err, "failed to unmarshal client state bytes into tendermint client state")
		}

		tmClientState, ok := clientState.(*ibctm.ClientState)
		if !ok {
			return sdkerrors.Wrap(types.ErrInvalidClient, "client state is not tendermint even though client id contains 07-tendermint")
		}

		amtPruned := ibctm.PruneAllExpiredConsensusStates(ctx, clientStore, cdc, tmClientState)
		totalPruned = totalPruned + amtPruned
	}

	clientLogger := ctx.Logger().With("module", "x/"+host.ModuleName+"/"+types.SubModuleName)
	clientLogger.Info("pruned expired tendermint consensus states", "total", totalPruned)

	return nil
}
