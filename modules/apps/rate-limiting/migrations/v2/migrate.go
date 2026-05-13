package v2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"strings"

	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	host "github.com/cosmos/ibc-go/v11/modules/core/24-host"
)

const (
	// oldPendingSendPacketChannelLength is hard-coded so the migration stays
	// correct even if types.PendingSendPacketChannelLength is bumped again
	// later.
	oldPendingSendPacketChannelLength = 16
	oldKeyLen                         = oldPendingSendPacketChannelLength + 8

	// newPendingSendPacketChannelLength is hard-coded so the migration stays
	// correct even if types.PendingSendPacketChannelLength is bumped again
	// later.
	newPendingSendPacketChannelLength = 64
	newKeyLen                         = newPendingSendPacketChannelLength + 8
)

// Migrate rewrites entries under types.PendingSendPacketPrefix from the old
// [16-byte channelID][8-byte sequence] layout to the new [64-byte channelID]
// [8-byte sequence] layout so IBC v2 channel IDs (up to 64 bytes) fit. Entries
// already in the new layout are skipped, making the migration idempotent.
func Migrate(ctx sdk.Context, storeService corestore.KVStoreService) error {
	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)

	// get store entries that need to be migrated
	legacyEntries, err := collectLegacyEntries(store)
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
		copy(newKey[newPendingSendPacketChannelLength:], entry.key[oldPendingSendPacketChannelLength:])

		if err := validateMigratedKey(newKey, entry.key); err != nil {
			return fmt.Errorf("validating migrated key %X against legacy key %X: %w", newKey, entry.key, err)
		}

		// remove old kv and set new kv
		store.Delete(entry.key[:])
		store.Set(newKey, entry.value)
	}

	return nil
}

type entry struct {
	key   [oldKeyLen]byte
	value []byte
}

// collectLegacyEntries returns a list of entries in the prefix store that must
// be migrated from the oldKeyLen to the newKeyLen.
func collectLegacyEntries(store prefix.Store) ([]entry, error) {
	var (
		legacy []entry
		err    error
	)

	iterator := store.Iterator(nil, nil)
	defer func() {
		err = errors.Join(err, iterator.Close())
	}()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		switch len(key) {
		case newKeyLen:
			// already correct length, nothing to do
			continue
		case oldKeyLen:
			legacy = append(legacy, entry{
				key:   [oldKeyLen]byte(key),
				value: slices.Clone(iterator.Value()),
			})
		default:
			err = fmt.Errorf("unexpected pending-send-packet key length %d (want %d or %d)", len(key), oldKeyLen, newKeyLen)
			return nil, err
		}
	}

	return legacy, nil
}

// validateMigratedKey ensures a legacy key transformation is valid, returns an
// error if not.
func validateMigratedKey(newKey []byte, oldKey [oldKeyLen]byte) error {
	// channelID is right-padded with null bytes in the key (see
	// types.PendingSendPacketKey), so trim them before validating.
	rawChannelID := string(newKey[:newPendingSendPacketChannelLength])
	channelID := strings.TrimRight(rawChannelID, "\x00")

	// validate channelID in the newKey.

	// we are using the client
	// validator here since in v1 these will be channelID's, and in v2 they
	// are clientID's, the client validator is slightly less strict and
	// will accept both.
	if err := host.ClientIdentifierValidator(channelID); err != nil {
		return fmt.Errorf("invalid channel or client ID %q in migrated key: %w", channelID, err)
	}

	// ensure we have not modified the existing value
	if !bytes.Equal(newKey[:oldPendingSendPacketChannelLength], oldKey[:oldPendingSendPacketChannelLength]) {
		return fmt.Errorf("first %d bytes of migrated key not do not match existing key", oldPendingSendPacketChannelLength)
	}
	// ensure after the oldPendingSendPacketChannelLength, we have 48 bytes of 0's
	padding := newKey[oldPendingSendPacketChannelLength:newPendingSendPacketChannelLength]
	if !bytes.Equal(padding, make([]byte, len(padding))) {
		return fmt.Errorf("expected all zero values after channel identifier in migrated key")
	}

	// validate sequence number in the newKey.
	sequenceRaw := newKey[newPendingSendPacketChannelLength:]
	sequence := binary.BigEndian.Uint64(sequenceRaw)
	if sequence == 0 {
		return fmt.Errorf("invalid sequence 0 for %q in migrated key", channelID)
	}

	// ensure we have not modified the existing value
	if !bytes.Equal(sequenceRaw, oldKey[oldPendingSendPacketChannelLength:]) {
		return fmt.Errorf("mismatch in sequence number bytes between migrated key and existing key")
	}

	return nil
}
