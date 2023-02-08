package keeper

import (
	"fmt"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
)

// SetChainToTuple sets the tuple (channel, port) for a given chain ID
func (k Keeper) SetChainToTuple(ctx sdk.Context, chainId, channel, port string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ChainToTupleKeyPrefix)
	key := types.GetChainToTupleKey(chainId)

	bz := types.TupleToBytes(channel, port)
	store.Set(key, bz)
}

// GetChainToTuple gets the tuple (channel, port) for a given chain ID
func (k Keeper) GetChainToTuple(ctx sdk.Context, chainId string) (string, string, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ChainToTupleKeyPrefix)
	key := types.GetChainToTupleKey(chainId)

	bz := store.Get(key)
	if bz == nil {
		return "", "", fmt.Errorf("chain %s not found", chainId)
	}

	channel, port, err := types.GetChannelPortFromTuple(bz)
	if err != nil {
		return "", "", err
	}

	return channel, port, nil
}

// SetTupleToChain sets the chain ID for a given tuple (channel, port)
func (k Keeper) SetTupleToChain(ctx sdk.Context, chainId, channel, port string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.TupleToChainKeyPrefix)
	key := types.GetTupleToChainKey(channel, port)

	store.Set(key, []byte(chainId))
}

// GetTupleToChain gets the chain ID for a given tuple (channel, port)
func (k Keeper) GetTupleToChain(ctx sdk.Context, channel, port string) (string, error) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.TupleToChainKeyPrefix)
	key := types.GetTupleToChainKey(channel, port)

	bz := store.Get(key)
	if bz == nil {
		return "", fmt.Errorf("tuple (%s, %s) not found", channel, port)
	}

	return string(bz), nil
}
