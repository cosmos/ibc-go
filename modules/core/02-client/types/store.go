package types

import (
	"fmt"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

// StoreProvider encapsulates the IBC core store service and offers convenience methods for LightClientModules.
type StoreProvider struct {
	storeService corestore.KVStoreService
}

// NewStoreProvider creates and returns a new client StoreProvider.
func NewStoreProvider(storeService corestore.KVStoreService) StoreProvider {
	return StoreProvider{
		storeService: storeService,
	}
}

// ClientStore returns isolated prefix store for each client so they can read/write in separate namespaces.
func (s StoreProvider) ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore {
	clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
	return prefix.NewStore(runtime.KVStoreAdapter(s.storeService.OpenKVStore(ctx)), clientPrefix)
}

// ClientModuleStore returns the module store for a provided client type.
func (s StoreProvider) ClientModuleStore(ctx sdk.Context, clientType string) storetypes.KVStore {
	return prefix.NewStore(runtime.KVStoreAdapter(s.storeService.OpenKVStore(ctx)), host.PrefixedClientStoreKey([]byte(clientType)))
}
