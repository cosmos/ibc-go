package keeper_test

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ratelimiting "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting"
	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/keeper"
	ratelimittypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

func (s *KeeperTestSuite) TestMigrate1to2() {
	const oldPendingSendPacketChannelLength = 16
	const oldKeyLen = oldPendingSendPacketChannelLength + 8

	writeLegacy := func(store prefix.Store, channelID string, sequence uint64) {
		key := make([]byte, oldKeyLen)
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

	s.SetupTest()

	ctx = s.chainA.GetContext()
	rlKeeper = s.chainA.GetSimApp().RateLimitKeeper
	migrator = keeper.NewMigrator(rlKeeper)

	storeService := runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ratelimittypes.StoreKey))
	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	pendingSendStore := prefix.NewStore(adapter, ratelimittypes.PendingSendPacketPrefix)
	pendingReceiveStore := prefix.NewStore(adapter, ratelimittypes.PendingReceivePacketPrefix)

	writeLegacy(pendingSendStore, "channel-1", 1)
	writeLegacy(pendingSendStore, "channel-1", 2)
	writeLegacy(pendingReceiveStore, "channel-1", 1)
	writeLegacy(pendingReceiveStore, "channel-2", 1)
	pendingSendStore.Set([]byte("unexpected-length-send"), []byte{1})
	pendingReceiveStore.Set([]byte("unexpected-length-receive"), []byte{1})

	err := rlKeeper.SetPendingSendPacket(ctx, "channel-3", 1, "denom-a")
	s.Require().NoError(err)
	err = rlKeeper.SetPendingReceivePacket(ctx, "channel-3", 1, "denom-a")
	s.Require().NoError(err)

	err = migrator.Migrate1to2(ctx)
	s.Require().NoError(err)

	s.Require().Empty(readAllKeys(pendingSendStore), "legacy pending send packet store")
	s.Require().Empty(readAllKeys(pendingReceiveStore), "legacy pending receive packet store")

	found, err := rlKeeper.CheckPacketSentDuringCurrentQuota(ctx, "channel-3", 1, "denom-a")
	s.Require().NoError(err)
	s.Require().True(found, "collections pending send packet should be preserved")
	found, err = rlKeeper.CheckPacketReceivedDuringCurrentQuota(ctx, "channel-3", 1, "denom-a")
	s.Require().NoError(err)
	s.Require().True(found, "collections pending receive packet should be preserved")
}

// TestMigrate2to3 pins the Migrator wiring: a rate limit stored under the
// legacy key layout must be readable through the keeper after Migrate2to3, and
// the module's consensus version must match the highest registered migration
// target.
func (s *KeeperTestSuite) TestMigrate2to3() {
	s.SetupTest()

	ctx := s.chainA.GetContext()
	rlKeeper := s.chainA.GetSimApp().RateLimitKeeper
	migrator := keeper.NewMigrator(rlKeeper)

	rateLimit := ratelimittypes.RateLimit{
		Path: &ratelimittypes.Path{Denom: "uatom", ChannelOrClientId: "channel-1"},
	}

	storeService := runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ratelimittypes.StoreKey))
	adapter := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, ratelimittypes.RateLimitKeyPrefix)

	legacyKey := append([]byte(rateLimit.Path.Denom), rateLimit.Path.ChannelOrClientId...)
	store.Set(legacyKey, s.chainA.GetSimApp().AppCodec().MustMarshal(&rateLimit))

	s.Require().NoError(migrator.Migrate2to3(ctx))

	migrated, found := rlKeeper.GetRateLimit(ctx, "uatom", "channel-1")
	s.Require().True(found, "rate limit not readable through the keeper after Migrate2to3")
	s.Require().Equal(rateLimit, migrated)

	s.Require().Equal(uint64(3), ratelimiting.AppModule{}.ConsensusVersion())
}
