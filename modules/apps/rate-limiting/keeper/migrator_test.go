package keeper_test

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/keeper"
	ratelimittypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

func (s *KeeperTestSuite) TestMigrate1to2() {
	const oldPendingSendPacketChannelLength = 16
	newKeyLen := ratelimittypes.PendingSendPacketChannelLength + 8

	writeLegacy := func(store prefix.Store, channelID string, sequence uint64) {
		key := make([]byte, oldPendingSendPacketChannelLength+8)
		copy(key, channelID)
		binary.BigEndian.PutUint64(key[oldPendingSendPacketChannelLength:], sequence)
		store.Set(key, []byte{1})
	}

	readAllKeys := func(store prefix.Store) [][]byte {
		it := store.Iterator(nil, nil)
		defer it.Close()

		var keys [][]byte
		for ; it.Valid(); it.Next() {
			keys = append(keys, append([]byte(nil), it.Key()...))
		}
		return keys
	}

	var (
		ctx      sdk.Context
		rlKeeper *keeper.Keeper
		migrator keeper.Migrator
	)

	tests := []struct {
		name      string
		malleate  func(store prefix.Store)
		expectAll []string
		postCheck func(store prefix.Store)
		expectErr string
	}{
		{
			name:     "empty store",
			malleate: func(_ prefix.Store) {},
		},
		{
			name: "legacy entries rewritten",
			malleate: func(store prefix.Store) {
				writeLegacy(store, "channel-1", 0)
				writeLegacy(store, "channel-1", 1)
				writeLegacy(store, "channel-11", 0)
				writeLegacy(store, "channel-11", 7)
			},
			expectAll: []string{
				"channel-1/0", "channel-1/1",
				"channel-11/0", "channel-11/7",
			},
		},
		{
			name: "mixed legacy and new-layout entries",
			malleate: func(store prefix.Store) {
				writeLegacy(store, "channel-1", 0)
				writeLegacy(store, "channel-11", 7)
				rlKeeper.SetPendingSendPacket(ctx, "channel-99", 5)
			},
			expectAll: []string{
				"channel-1/0", "channel-11/7", "channel-99/5",
			},
		},
		{
			name: "idempotency",
			malleate: func(store prefix.Store) {
				writeLegacy(store, "channel-1", 0)
				writeLegacy(store, "channel-1", 1)
			},
			expectAll: []string{"channel-1/0", "channel-1/1"},
			postCheck: func(store prefix.Store) {
				before := readAllKeys(store)
				s.Require().NoError(migrator.Migrate1to2(ctx))
				s.Require().ElementsMatch(before, readAllKeys(store))
			},
		},
		{
			name: "long IBC v2 keys properly round trips post migration",
			malleate: func(store prefix.Store) {
				writeLegacy(store, "channel-1", 3)
			},
			expectAll: []string{"channel-1/3"},
			postCheck: func(store prefix.Store) {
				longChannelID := fmt.Sprintf("channel-%050d", 1)
				s.Require().Greater(len(longChannelID), oldPendingSendPacketChannelLength)
				rlKeeper.SetPendingSendPacket(ctx, longChannelID, 9)
				s.Require().True(rlKeeper.CheckPacketSentDuringCurrentQuota(ctx, longChannelID, 9))
				s.Require().True(rlKeeper.CheckPacketSentDuringCurrentQuota(ctx, "channel-1", 3))
			},
		},
		{
			name:     "long channel keys sharing first 16 bytes dont collide post migration",
			malleate: func(_ prefix.Store) {},
			postCheck: func(store prefix.Store) {
				// Two keys whose first 16 bytes are identical but full strings differ.
				shared := strings.Repeat("a", oldPendingSendPacketChannelLength)
				idA := shared + "-suffix-A"
				idB := shared + "-suffix-B"

				rlKeeper.SetPendingSendPacket(ctx, idA, 1)
				rlKeeper.SetPendingSendPacket(ctx, idB, 2)

				s.Require().True(rlKeeper.CheckPacketSentDuringCurrentQuota(ctx, idA, 1))
				s.Require().True(rlKeeper.CheckPacketSentDuringCurrentQuota(ctx, idB, 2))
				s.Require().Len(readAllKeys(store), 2, "two distinct keys despite shared first 16 bytes")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.SetupTest()

			ctx = s.chainA.GetContext()
			rlKeeper = s.chainA.GetSimApp().RateLimitKeeper
			migrator = keeper.NewMigrator(rlKeeper)

			storeService := runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ratelimittypes.StoreKey))
			adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
			prefixStore := prefix.NewStore(adapter, ratelimittypes.PendingSendPacketPrefix)

			// set initial store state
			tc.malleate(prefixStore)

			// perform migration
			err := migrator.Migrate1to2(ctx)

			if tc.expectErr != "" {
				s.Require().ErrorContains(err, tc.expectErr)
				return
			}
			s.Require().NoError(err)

			// assert that all keys match our expected keys, or there are no
			// keys at all
			for _, k := range readAllKeys(prefixStore) {
				s.Require().Len(k, newKeyLen, "every post-migration key must be in the new layout")
			}
			if tc.expectAll == nil {
				s.Require().Empty(rlKeeper.GetAllPendingSendPackets(ctx))
			} else {
				s.Require().ElementsMatch(tc.expectAll, rlKeeper.GetAllPendingSendPackets(ctx))
			}

			// do custom post checks
			if tc.postCheck != nil {
				tc.postCheck(prefixStore)
			}
		})
	}
}
