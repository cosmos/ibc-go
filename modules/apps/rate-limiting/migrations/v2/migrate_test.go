package v2_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2/prefix"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/migrations/v2"
	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
)

func TestMigrateClearsLegacyPendingPacketStores(t *testing.T) {
	coordinator := ibctesting.NewCoordinator(t, 1)
	chain := coordinator.GetChain(ibctesting.GetChainID(1))
	ctx := chain.GetContext()

	storeService := runtime.NewKVStoreService(chain.GetSimApp().GetKey(types.StoreKey))
	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	pendingSendStore := prefix.NewStore(adapter, types.PendingSendPacketPrefix)
	pendingReceiveStore := prefix.NewStore(adapter, types.PendingReceivePacketPrefix)

	pendingSendStore.Set([]byte("send-1"), []byte{1})
	pendingSendStore.Set([]byte("send-2"), []byte{2})
	pendingReceiveStore.Set([]byte("receive-1"), []byte{1})
	pendingReceiveStore.Set([]byte("receive-2"), []byte{2})

	require.NoError(t, v2.Migrate(ctx, storeService))
	require.Empty(t, collectKeys(t, pendingSendStore))
	require.Empty(t, collectKeys(t, pendingReceiveStore))
}

func collectKeys(t *testing.T, store prefix.Store) [][]byte {
	t.Helper()

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	var keys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, append([]byte(nil), iterator.Key()...))
	}

	return keys
}
