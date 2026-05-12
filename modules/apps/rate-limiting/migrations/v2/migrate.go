package v2

import (
	"fmt"
	"slices"

	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

// oldPendingSendPacketChannelLength is hard-coded so the migration stays
// correct even if types.PendingSendPacketChannelLength is bumped again later.
const oldPendingSendPacketChannelLength = 16

// Migrate rewrites entries under types.PendingSendPacketPrefix from the old
// [16-byte channelID][8-byte sequence] layout to the new [64-byte channelID]
// [8-byte sequence] layout so IBC v2 channel IDs (up to 64 bytes) fit. Entries
// already in the new layout are skipped, making the migration idempotent.
func Migrate(ctx sdk.Context, storeService corestore.KVStoreService) error {
	var (
		oldKeyLen = oldPendingSendPacketChannelLength + 8
		newKeyLen = types.PendingSendPacketChannelLength + 8
	)

	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)

	// get store entries that need to be migrated
	legacyEntries, err := collectLegacyEntries(store, oldKeyLen, newKeyLen)
	if err != nil {
		return fmt.Errorf("collecting legacy pending send packet entries from prefix store: %w", err)
	}

	// migrate store entries
	// old key layout: [16-byte channelID][8-byte sequence]
	// new key layout: [64-byte channelID][8-byte sequence]
	for _, entry := range legacyEntries {
		newKey := make([]byte, newKeyLen)

		// place 16 byte channel id from old key into first 64 bytes of new key
		copy(newKey, entry.key[:oldPendingSendPacketChannelLength])

		// put remaining 8 bytes sequence from old key into the final 8 bytes
		// sequence of the new key
		copy(newKey[types.PendingSendPacketChannelLength:], entry.key[oldPendingSendPacketChannelLength:])

		// remove old kv and set new kv
		store.Delete(entry.key)
		store.Set(newKey, entry.value)
	}

	return nil
}

type entry struct {
	key, value []byte
}

// collectLegacyEntries returns a list of entries in the prefix store that must
// be migrated from the oldKeyLen to the newKeyLen.
func collectLegacyEntries(store prefix.Store, oldKeyLen, newKeyLen int) ([]entry, error) {
	var legacy []entry

	iterator := store.Iterator(nil, nil)
	defer func() {
		_ = iterator.Close()
	}()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		switch len(key) {
		case newKeyLen:
			// already correct length, noting to do
			continue
		case oldKeyLen:
			legacy = append(legacy, entry{
				key:   slices.Clone(key),
				value: slices.Clone(iterator.Value()),
			})
		default:
			return nil, fmt.Errorf("unexpected pending-send-packet key length %d (want %d or %d)", len(key), oldKeyLen, newKeyLen)
		}
	}

	return legacy, nil
}
