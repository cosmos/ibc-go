package types

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

var _ exported.ClientStoreProvider = (*storeProvider)(nil)

// storeProvider implements the api.ClientStoreProvider interface and encapsulates the IBC core store key.
type storeProvider struct {
	storeKey storetypes.StoreKey
}

// NewStoreProvider creates and returns a new ClientStoreProvider.
func NewStoreProvider(storeKey storetypes.StoreKey) exported.ClientStoreProvider {
	return storeProvider{
		storeKey: storeKey,
	}
}

// ClientStore returns isolated prefix store for each client so they can read/write in separate namespaces.
func (s storeProvider) ClientStore(ctx sdk.Context, clientID string) storetypes.KVStore {
	clientPrefix := []byte(fmt.Sprintf("%s/%s/", host.KeyClientStorePrefix, clientID))
	return prefix.NewStore(ctx.KVStore(s.storeKey), clientPrefix)
}

// ClientModuleStore returns the module store for a provided client type.
func (s storeProvider) ClientModuleStore(ctx sdk.Context, clientType string) storetypes.KVStore {
	return prefix.NewStore(ctx.KVStore(s.storeKey), host.PrefixedClientStoreKey([]byte(clientType)))
}
