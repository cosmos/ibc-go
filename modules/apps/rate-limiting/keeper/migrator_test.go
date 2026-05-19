package keeper_test

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/keeper"
	ratelimittypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

func (s *KeeperTestSuite) TestMigrate1to2() {
	const oldPendingSendPacketChannelLength = 16
	const newPendingSendPacketChannelLength = 64
	const oldKeyLen = oldPendingSendPacketChannelLength + 8
	const newKeyLen = newPendingSendPacketChannelLength + 8

	writeLegacy := func(store prefix.Store, channelID string, sequence uint64) {
		key := make([]byte, oldKeyLen)
		copy(key, channelID)
		binary.BigEndian.PutUint64(key[oldPendingSendPacketChannelLength:], sequence)
		store.Set(key, []byte{1})
	}

	writeNewLayout := func(store prefix.Store, channelID string, sequence uint64) {
		key := make([]byte, newKeyLen)
		copy(key, channelID)
		binary.BigEndian.PutUint64(key[newPendingSendPacketChannelLength:], sequence)
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
		expectErr string
	}{
		{
			name:     "success: empty store",
			malleate: func(_ prefix.Store) {},
		},
		{
			name: "success: legacy entries rewritten",
			malleate: func(store prefix.Store) {
				writeLegacy(store, "channel-1", 1)
				writeLegacy(store, "channel-1", 2)
				writeLegacy(store, "channel-11", 1)
				writeLegacy(store, "channel-11", 7)
				writeLegacy(store, "07-tendermint-10", 500)
				// note this following channelID will get truncated when written to the store
				writeLegacy(store, "07-tendermint-5011", 600)
			},
			expectAll: []string{
				"channel-1/1", "channel-1/2",
				"channel-11/1", "channel-11/7",
				"07-tendermint-10/500", "07-tendermint-50/600",
			},
		},
		{
			name: "failure: already migrated entry",
			malleate: func(store prefix.Store) {
				writeLegacy(store, "channel-1", 1)
				writeNewLayout(store, "channel-99", 5)
			},
			expectErr: "unexpected pending-send-packet key length",
		},
		{
			name: "failure: unexpected key length",
			malleate: func(store prefix.Store) {
				// key length is does not match oldKeyLen
				store.Set(make([]byte, oldKeyLen+1), []byte{1})
			},
			expectErr: "unexpected pending-send-packet key length",
		},
		{
			name: "failure: existing channel ID too short",
			malleate: func(store prefix.Store) {
				writeLegacy(store, "abc", 1)
			},
			expectErr: "invalid channel or client ID",
		},
		{
			name: "failure: invalid existing channelID",
			malleate: func(store prefix.Store) {
				writeLegacy(store, "ch/1", 1)
			},
			expectErr: "invalid channel or client ID",
		},
		{
			name: "failure: existing sequence 0",
			malleate: func(store prefix.Store) {
				writeLegacy(store, "channel-1", 0)
			},
			expectErr: "invalid sequence 0",
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

			pendingSendPackets, err := rlKeeper.GetAllPendingSendPackets(ctx)
			s.Require().NoError(err)
			if tc.expectAll == nil {
				s.Require().Empty(pendingSendPackets)
			} else {
				s.Require().ElementsMatch(tc.expectAll, pendingSendPackets)
			}
		})
	}
}
