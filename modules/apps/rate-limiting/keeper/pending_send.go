package keeper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Sets the sequence number of a packet that was just sent
func (k Keeper) SetPendingSendPacket(ctx sdk.Context, channelId string, sequence uint64) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte(types.PendingSendPacketPrefix))
	key := types.KeyPendingSendPacket(channelId, sequence)
	store.Set(key, []byte{1})
}

// Remove a pending packet sequence number from the store
// Used after the ack or timeout for a packet has been received
func (k Keeper) RemovePendingSendPacket(ctx sdk.Context, channelId string, sequence uint64) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte(types.PendingSendPacketPrefix))
	key := types.KeyPendingSendPacket(channelId, sequence)
	store.Delete(key)
}

// Checks whether the packet sequence number is in the store - indicating that it was
// sent during the current quota
func (k Keeper) CheckPacketSentDuringCurrentQuota(ctx sdk.Context, channelId string, sequence uint64) bool {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte(types.PendingSendPacketPrefix))
	key := types.KeyPendingSendPacket(channelId, sequence)
	valueBz := store.Get(key)
	found := len(valueBz) != 0
	return found
}

// Get all pending packet sequence numbers
func (k Keeper) GetAllPendingSendPackets(ctx sdk.Context) []string {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte(types.PendingSendPacketPrefix))

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	pendingPackets := []string{}
	for ; iterator.Valid(); iterator.Next() {
		key := string(iterator.Key())

		// The key format from KeyPendingSendPacket is "{PendingSendPacketPrefix}/{channelId}/{sequenceNumber}"
		// When using prefix.NewStore with PendingSendPacketPrefix, the store keys are just "{channelId}/{sequenceNumber}"
		parts := strings.Split(key, "/")
		if len(parts) != 2 {
			// Skip invalid keys - this should not happen
			continue
		}

		channelId := parts[0]

		// The sequence number is formatted with %20d in KeyPendingSendPacket, so we need to trim spaces
		sequenceStr := strings.TrimSpace(parts[1])
		sequence, err := strconv.ParseUint(sequenceStr, 10, 64)
		if err != nil {
			// Skip invalid sequence numbers - this should not happen
			continue
		}

		packetId := fmt.Sprintf("%s/%d", channelId, sequence)
		pendingPackets = append(pendingPackets, packetId)
	}

	return pendingPackets
}

// Remove all pending sequence numbers from the store
// This is executed when the quota resets
func (k Keeper) RemoveAllChannelPendingSendPackets(ctx sdk.Context, channelId string) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, []byte(types.PendingSendPacketPrefix))

	iterator := storetypes.KVStorePrefixIterator(store, []byte(channelId))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}
