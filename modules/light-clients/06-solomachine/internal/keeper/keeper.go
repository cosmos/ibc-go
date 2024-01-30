package keeper

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

// Keeper defines the 06-solomachine Keeper.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey
}

// Codec returns the keeper codec.
func (k Keeper) Codec() codec.Codec {
	return k.cdc
}

// ClientStore returns a namespaced prefix store for the provided IBC client identifier.
func (k Keeper) ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore {
	clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
	return prefix.NewStore(ctx.KVStore(k.storeKey), clientPrefix)
}
