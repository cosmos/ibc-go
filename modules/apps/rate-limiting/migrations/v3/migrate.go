package v3

import (
	"errors"
	"fmt"

	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

// Migrate re-keys every rate limit from the legacy denom||channelOrClientID
// layout to the channel-first, length-prefixed layout of
// types.RateLimitItemKey, and every whitelisted address pair from the legacy
// sender||receiver layout to the length-prefixed layout of
// types.AddressWhitelistKey.
//
// The legacy keys have no separator and are therefore ambiguous, so the new
// keys are derived from the stored values rather than parsed out of the old
// keys.
func Migrate(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec) error {
	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))

	err := rekey(prefix.NewStore(adapter, types.RateLimitKeyPrefix), func(value []byte) ([]byte, error) {
		var rateLimit types.RateLimit
		if err := cdc.Unmarshal(value, &rateLimit); err != nil {
			return nil, fmt.Errorf("unmarshalling rate limit: %w", err)
		}
		if rateLimit.Path == nil {
			return nil, errors.New("rate limit has no path")
		}
		return types.RateLimitItemKey(rateLimit.Path.Denom, rateLimit.Path.ChannelOrClientId), nil
	})
	if err != nil {
		return fmt.Errorf("re-keying rate limits: %w", err)
	}

	err = rekey(prefix.NewStore(adapter, types.AddressWhitelistKeyPrefix), func(value []byte) ([]byte, error) {
		var pair types.WhitelistedAddressPair
		if err := cdc.Unmarshal(value, &pair); err != nil {
			return nil, fmt.Errorf("unmarshalling whitelisted address pair: %w", err)
		}
		return types.AddressWhitelistKey(pair.Sender, pair.Receiver), nil
	})
	if err != nil {
		return fmt.Errorf("re-keying address whitelist: %w", err)
	}

	return nil
}

// rekey rewrites every entry of store under newKey(value). All old keys are
// deleted before any new key is written, since a new key may equal a
// not-yet-migrated old key.
func rekey(store prefix.Store, newKey func(value []byte) ([]byte, error)) error {
	type entry struct {
		oldKey, newKey, value []byte
	}

	iterator := store.Iterator(nil, nil)
	entries := make([]entry, 0)
	for ; iterator.Valid(); iterator.Next() {
		// The buffers behind Key()/Value() may be reused by Next() or released
		// by Close(), so copy up front: the copies are what is retained in
		// entries and what the error path below formats after closing the
		// iterator. The copies must stay non-nil even when zero-length
		// (append to []byte{}, not to a nil slice): store.Set panics on a nil
		// value, and a zero-value WhitelistedAddressPair marshals to zero
		// bytes.
		oldKey := append([]byte{}, iterator.Key()...)
		value := append([]byte{}, iterator.Value()...)

		key, err := newKey(value)
		if err != nil {
			iterator.Close()
			return fmt.Errorf("entry at key %X: %w", oldKey, err)
		}

		entries = append(entries, entry{oldKey: oldKey, newKey: key, value: value})
	}
	if err := iterator.Close(); err != nil {
		return err
	}

	for _, e := range entries {
		store.Delete(e.oldKey)
	}
	for _, e := range entries {
		store.Set(e.newKey, e.value)
	}

	return nil
}
