package keeper

import (
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// Adds an pair of sender and receiver addresses to the whitelist to allow all
// IBC transfers between those addresses to skip all flow calculations
func (k *Keeper) SetWhitelistedAddressPair(ctx sdk.Context, whitelist types.WhitelistedAddressPair) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.AddressWhitelistKeyPrefix)
	key := types.AddressWhitelistKey(whitelist.Sender, whitelist.Receiver)
	value := k.cdc.MustMarshal(&whitelist)
	store.Set(key, value)
}

// Removes a whitelisted address pair so that it's transfers are counted in the quota
func (k *Keeper) RemoveWhitelistedAddressPair(ctx sdk.Context, sender, receiver string) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.AddressWhitelistKeyPrefix)
	key := types.AddressWhitelistKey(sender, receiver)
	store.Delete(key)
}

// Check if a sender/receiver address pair is currently whitelisted
func (k *Keeper) IsAddressPairWhitelisted(ctx sdk.Context, sender, receiver string) bool {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.AddressWhitelistKeyPrefix)

	key := types.AddressWhitelistKey(sender, receiver)
	value := store.Get(key)
	found := len(value) != 0

	return found
}

// Get all the whitelisted addresses
func (k *Keeper) GetAllWhitelistedAddressPairs(ctx sdk.Context) []types.WhitelistedAddressPair {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.AddressWhitelistKeyPrefix)

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	allWhitelistedAddresses := []types.WhitelistedAddressPair{}
	for ; iterator.Valid(); iterator.Next() {
		whitelist := types.WhitelistedAddressPair{}
		k.cdc.MustUnmarshal(iterator.Value(), &whitelist)
		allWhitelistedAddresses = append(allWhitelistedAddresses, whitelist)
	}

	return allWhitelistedAddresses
}
