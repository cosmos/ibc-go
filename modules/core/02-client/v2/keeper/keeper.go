package keeper

import (
	"cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
)

type Keeper struct {
	cdc            codec.BinaryCodec
	kvStoreService store.KVStoreService
}

// NewKeeper creates a new client v2 keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	kvStoreService store.KVStoreService,
) *Keeper {
	return &Keeper{
		cdc:            cdc,
		kvStoreService: kvStoreService,
	}
}
