package types

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ exported.ClientStoreProvider = (*storeProvider)(nil)

type storeProvider struct {
	storeKey storetypes.StoreKey
}

func NewStoreProvider(storeKey storetypes.StoreKey) exported.ClientStoreProvider {
	return storeProvider{
		storeKey: storeKey,
	}
}

// ClientStore returns isolated prefix store for each client so they can read/write in separate
// namespace without being able to read/write other client's data
func (s storeProvider) ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore {
	clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
	return prefix.NewStore(ctx.KVStore(s.storeKey), clientPrefix)
}

// ModuleStore returns the 02-client module store
func (s storeProvider) ModuleStore(ctx sdk.Context, clientType string) storetypes.KVStore {
	return prefix.NewStore(ctx.KVStore(s.storeKey), host.KeyClientStorePrefix)
}
