package v2

import (
	"cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// MigrateStore migrates the rate-limiting store from v1 to v2 by:
// - Migrating whitelist entries from the incorrect "address-blacklist" prefix to the correct "address-whitelist" prefix
func MigrateStore(ctx sdk.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	kvStore := storeService.OpenKVStore(ctx)
	return migrateAddressWhitelistKeys(runtime.KVStoreAdapter(kvStore), cdc)
}

// migrateAddressWhitelistKeys migrates whitelist entries from the legacy key prefix to the correct key prefix
func migrateAddressWhitelistKeys(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// Get all entries with the legacy prefix
	iterator := storetypes.KVStorePrefixIterator(store, types.LegacyAddressWhitelistKeyPrefix)
	defer iterator.Close()

	// Collect all entries that need to be migrated
	var entries []types.WhitelistedAddressPair
	var keysToDelete [][]byte

	for ; iterator.Valid(); iterator.Next() {
		var whitelist types.WhitelistedAddressPair
		if err := cdc.Unmarshal(iterator.Value(), &whitelist); err != nil {
			return err
		}
		entries = append(entries, whitelist)
		keysToDelete = append(keysToDelete, iterator.Key())
	}

	// Set entries with the new prefix
	for _, whitelist := range entries {
		newKey := append(types.AddressWhitelistKeyPrefix, types.AddressWhitelistKey(whitelist.Sender, whitelist.Receiver)...)
		value := cdc.MustMarshal(&whitelist)
		store.Set(newKey, value)
	}

	// Delete old entries
	for _, key := range keysToDelete {
		store.Delete(key)
	}

	return nil
}
