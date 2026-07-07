package v2

import (
	"fmt"

	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

// Migrate clears legacy pending packet markers. The old markers were keyed by
// (channelOrClientID, sequence) and did not contain the denom, so they cannot be
// migrated into the new denom-scoped collections state.
func Migrate(ctx sdk.Context, storeService corestore.KVStoreService) error {
	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))

	if err := clearPrefixStore(prefix.NewStore(adapter, types.PendingSendPacketPrefix)); err != nil {
		return fmt.Errorf("clearing legacy pending send packet entries: %w", err)
	}
	if err := clearPrefixStore(prefix.NewStore(adapter, types.PendingReceivePacketPrefix)); err != nil {
		return fmt.Errorf("clearing legacy pending receive packet entries: %w", err)
	}

	return nil
}

func clearPrefixStore(store prefix.Store) error {
	iterator := store.Iterator(nil, nil)

	keys := make([][]byte, 0)
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, append([]byte(nil), iterator.Key()...))
	}
	if err := iterator.Close(); err != nil {
		return err
	}

	for _, key := range keys {
		store.Delete(key)
	}

	return nil
}
