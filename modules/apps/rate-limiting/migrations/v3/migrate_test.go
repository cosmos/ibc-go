package v3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	corestore "cosmossdk.io/core/store"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/migrations/v3"
	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
)

// legacyRateLimitItemKey is the pre-v3 key layout: denom first, no separator.
func legacyRateLimitItemKey(denom, channelOrClientID string) []byte {
	return append([]byte(denom), channelOrClientID...)
}

// legacyAddressWhitelistKey is the pre-v3 key layout: sender||receiver, no
// separator.
func legacyAddressWhitelistKey(sender, receiver string) []byte {
	return append([]byte(sender), receiver...)
}

func rateLimit(denom, channelOrClientID string) types.RateLimit {
	return types.RateLimit{
		Path: &types.Path{Denom: denom, ChannelOrClientId: channelOrClientID},
		Quota: &types.Quota{
			MaxPercentSend: sdkmath.NewInt(10),
			MaxPercentRecv: sdkmath.NewInt(20),
			DurationHours:  24,
		},
		Flow: &types.Flow{
			Inflow:       sdkmath.NewInt(1),
			Outflow:      sdkmath.NewInt(2),
			ChannelValue: sdkmath.NewInt(100),
		},
	}
}

func setupMigrationTest(t *testing.T) (sdk.Context, corestore.KVStoreService, codec.Codec) {
	t.Helper()

	coordinator := ibctesting.NewCoordinator(t, 1)
	chain := coordinator.GetChain(ibctesting.GetChainID(1))

	return chain.GetContext(), runtime.NewKVStoreService(chain.GetSimApp().GetKey(types.StoreKey)), chain.GetSimApp().AppCodec()
}

func TestMigrateReKeysRateLimits(t *testing.T) {
	ctx, storeService, cdc := setupMigrationTest(t)
	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.RateLimitKeyPrefix)

	rateLimits := []types.RateLimit{
		rateLimit("uatom", "channel-0"),
		rateLimit("uatom", "channel-1"),
		rateLimit("uosmo", "channel-1"),
		rateLimit("uosmo", "07-tendermint-0"),
	}

	for _, rl := range rateLimits {
		store.Set(legacyRateLimitItemKey(rl.Path.Denom, rl.Path.ChannelOrClientId), cdc.MustMarshal(&rl))
	}
	require.Len(t, collectKeys(t, store), len(rateLimits))

	require.NoError(t, v3.Migrate(ctx, storeService, cdc))

	// Every entry is readable under its new key with its value intact, and no
	// legacy key survives.
	for _, rl := range rateLimits {
		bz := store.Get(types.RateLimitItemKey(rl.Path.Denom, rl.Path.ChannelOrClientId))
		require.NotEmpty(t, bz, "rate limit %s/%s missing after migration", rl.Path.Denom, rl.Path.ChannelOrClientId)

		var migrated types.RateLimit
		require.NoError(t, cdc.Unmarshal(bz, &migrated))
		require.Equal(t, rl, migrated)

		require.Empty(t, store.Get(legacyRateLimitItemKey(rl.Path.Denom, rl.Path.ChannelOrClientId)),
			"legacy key for %s/%s still set", rl.Path.Denom, rl.Path.ChannelOrClientId)
	}
	require.Len(t, collectKeys(t, store), len(rateLimits))

	// Migrating an already-migrated store is a byte-for-byte no-op: the new
	// keys are derived from stored values alone, so a second run maps every
	// entry to the key it is already under.
	before := snapshot(t, store)
	require.NoError(t, v3.Migrate(ctx, storeService, cdc))
	require.Equal(t, before, snapshot(t, store))
}

// TestMigrateKeyCollision covers the case the two-pass delete-then-set design
// exists for: the new key of one entry equals the legacy key of another, so a
// migration that deletes and sets per entry overwrites an entry that has not
// been migrated yet.
//
// Such a collision needs a legacy denom that starts with a length byte, e.g.
//
//	newKey("\x01gold", "channel-1")        = 09 "channel-1" 01 "gold"
//	legacyKey("\x09channel-1", "\x01gold") = 09 "channel-1" 01 "gold"
//
// Iteration order matters, and this pair pins the order that actually fails:
// the colliding entry's legacy key (01 "gold" "channel-1") sorts before the
// victim's (09 "channel-1" 01 "gold"), so a per-entry migration reaches the
// colliding entry while the victim is still stored under the key it is about
// to write. With the entries the other way round the victim would already have
// been moved, and the bug would pass by luck.
func TestMigrateKeyCollision(t *testing.T) {
	ctx, storeService, cdc := setupMigrationTest(t)
	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.RateLimitKeyPrefix)

	colliding, victim := rateLimit("\x01gold", "channel-1"), rateLimit("\x09channel-1", "\x01gold")
	require.Equal(t,
		types.RateLimitItemKey(colliding.Path.Denom, colliding.Path.ChannelOrClientId),
		legacyRateLimitItemKey(victim.Path.Denom, victim.Path.ChannelOrClientId),
		"fixture does not reproduce the newKey(A) == legacyKey(B) collision",
	)

	rateLimits := []types.RateLimit{colliding, victim}
	for _, rl := range rateLimits {
		store.Set(legacyRateLimitItemKey(rl.Path.Denom, rl.Path.ChannelOrClientId), cdc.MustMarshal(&rl))
	}
	require.Len(t, collectKeys(t, store), len(rateLimits))

	require.NoError(t, v3.Migrate(ctx, storeService, cdc))

	// Every entry survives under its own new key with its own value.
	for _, rl := range rateLimits {
		bz := store.Get(types.RateLimitItemKey(rl.Path.Denom, rl.Path.ChannelOrClientId))
		require.NotEmpty(t, bz, "rate limit %q/%q lost to the key collision", rl.Path.Denom, rl.Path.ChannelOrClientId)

		var migrated types.RateLimit
		require.NoError(t, cdc.Unmarshal(bz, &migrated))
		require.Equal(t, rl, migrated)
	}
	require.Len(t, collectKeys(t, store), len(rateLimits))
}

func TestMigrateReKeysWhitelist(t *testing.T) {
	ctx, storeService, cdc := setupMigrationTest(t)
	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.AddressWhitelistKeyPrefix)

	pairs := []types.WhitelistedAddressPair{
		{Sender: "cosmos1sender", Receiver: "cosmos1receiver"},
		{Sender: "cosmos1abc", Receiver: "cosmos1def"},
	}

	for _, pair := range pairs {
		store.Set(legacyAddressWhitelistKey(pair.Sender, pair.Receiver), cdc.MustMarshal(&pair))
	}
	require.Len(t, collectKeys(t, store), len(pairs))

	require.NoError(t, v3.Migrate(ctx, storeService, cdc))

	// Every pair is readable under its new key with its value intact, and no
	// legacy key survives.
	for _, pair := range pairs {
		bz := store.Get(types.AddressWhitelistKey(pair.Sender, pair.Receiver))
		require.NotEmpty(t, bz, "whitelist pair %s/%s missing after migration", pair.Sender, pair.Receiver)

		var migrated types.WhitelistedAddressPair
		require.NoError(t, cdc.Unmarshal(bz, &migrated))
		require.Equal(t, pair, migrated)

		require.Empty(t, store.Get(legacyAddressWhitelistKey(pair.Sender, pair.Receiver)),
			"legacy key for %s/%s still set", pair.Sender, pair.Receiver)
	}
	require.Len(t, collectKeys(t, store), len(pairs))

	// Same no-op property as the rate limit store: a second run derives the
	// same keys from the same values.
	before := snapshot(t, store)
	require.NoError(t, v3.Migrate(ctx, storeService, cdc))
	require.Equal(t, before, snapshot(t, store))
}

// TestMigrateWhitelistKeyCollision pins the same newKey(A) == legacyKey(B)
// hazard for the whitelist store:
//
//	newKey("\x01gold", "recv-1")        = 05 01 "gold" "recv-1"
//	legacyKey("\x05\x01gold", "recv-1") = 05 01 "gold" "recv-1"
//
// As in TestMigrateKeyCollision, the pair pins the iteration order that
// actually fails: the colliding entry's legacy key (01 "gold" "recv-1") sorts
// before the victim's (05 01 "gold" "recv-1"), so a per-entry migration
// reaches the colliding entry while the victim is still stored under the key
// it is about to write.
func TestMigrateWhitelistKeyCollision(t *testing.T) {
	ctx, storeService, cdc := setupMigrationTest(t)
	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.AddressWhitelistKeyPrefix)

	colliding := types.WhitelistedAddressPair{Sender: "\x01gold", Receiver: "recv-1"}
	victim := types.WhitelistedAddressPair{Sender: "\x05\x01gold", Receiver: "recv-1"}
	require.Equal(t,
		types.AddressWhitelistKey(colliding.Sender, colliding.Receiver),
		legacyAddressWhitelistKey(victim.Sender, victim.Receiver),
		"fixture does not reproduce the newKey(A) == legacyKey(B) collision",
	)

	pairs := []types.WhitelistedAddressPair{colliding, victim}
	for _, pair := range pairs {
		store.Set(legacyAddressWhitelistKey(pair.Sender, pair.Receiver), cdc.MustMarshal(&pair))
	}
	require.Len(t, collectKeys(t, store), len(pairs))

	require.NoError(t, v3.Migrate(ctx, storeService, cdc))

	// Every pair survives under its own new key with its own value.
	for _, pair := range pairs {
		bz := store.Get(types.AddressWhitelistKey(pair.Sender, pair.Receiver))
		require.NotEmpty(t, bz, "whitelist pair %q/%q lost to the key collision", pair.Sender, pair.Receiver)

		var migrated types.WhitelistedAddressPair
		require.NoError(t, cdc.Unmarshal(bz, &migrated))
		require.Equal(t, pair, migrated)
	}
	require.Len(t, collectKeys(t, store), len(pairs))
}

// TestMigrateZeroLengthValue seeds an entry whose stored value is zero bytes
// (a zero-value WhitelistedAddressPair marshals to nothing): the migration's
// value copy must stay non-nil empty, since store.Set panics on a nil value.
func TestMigrateZeroLengthValue(t *testing.T) {
	ctx, storeService, cdc := setupMigrationTest(t)
	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.AddressWhitelistKeyPrefix)

	value := cdc.MustMarshal(&types.WhitelistedAddressPair{})
	require.Empty(t, value)
	store.Set([]byte("legacy-key"), value)

	require.NoError(t, v3.Migrate(ctx, storeService, cdc))

	keys := collectKeys(t, store)
	require.Len(t, keys, 1)
	require.Equal(t, types.AddressWhitelistKey("", ""), keys[0])
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

// snapshot returns the full key -> value contents of the store, so that a
// re-run can be compared byte for byte rather than by entry count.
func snapshot(t *testing.T, store prefix.Store) map[string][]byte {
	t.Helper()

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	contents := map[string][]byte{}
	for ; iterator.Valid(); iterator.Next() {
		contents[string(iterator.Key())] = append([]byte(nil), iterator.Value()...)
	}

	return contents
}
