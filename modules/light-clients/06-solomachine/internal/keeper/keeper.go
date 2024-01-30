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
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
}

// NewKeeper creates and returns a new 06-solomachine keeper.
func NewKeeper(cdc codec.BinaryCodec, storeKey storetypes.StoreKey) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
	}
}

// Codec returns the keeper codec.
func (k Keeper) Codec() codec.BinaryCodec {
	return k.cdc
}

// ClientStore returns a namespaced prefix store for the provided IBC client identifier.
func (k Keeper) ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore {
	clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
	return prefix.NewStore(ctx.KVStore(k.storeKey), clientPrefix)
}
