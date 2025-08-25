package keeper

import (
	"encoding/binary"
	"fmt"
	"strings"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// Sets the sequence number of a packet that was just sent
func (k *Keeper) SetPendingSendPacket(ctx sdk.Context, channelID string, sequence uint64) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)
	key := types.PendingSendPacketKey(channelID, sequence)
	store.Set(key, []byte{1})
}

// Remove a pending packet sequence number from the store
// Used after the ack or timeout for a packet has been received
func (k *Keeper) RemovePendingSendPacket(ctx sdk.Context, channelID string, sequence uint64) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)
	key := types.PendingSendPacketKey(channelID, sequence)
	store.Delete(key)
}

// Checks whether the packet sequence number is in the store - indicating that it was
// sent during the current quota
func (k *Keeper) CheckPacketSentDuringCurrentQuota(ctx sdk.Context, channelID string, sequence uint64) bool {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)
	key := types.PendingSendPacketKey(channelID, sequence)
	valueBz := store.Get(key)
	found := len(valueBz) != 0
	return found
}

// Get all pending packet sequence numbers
func (k *Keeper) GetAllPendingSendPackets(ctx sdk.Context) []string {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	pendingPackets := make([]string, 0)
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()

		channelID := string(key[:types.PendingSendPacketChannelLength])
		channelID = strings.TrimRight(channelID, "\x00") // removes null bytes from suffix
		sequence := binary.BigEndian.Uint64(key[types.PendingSendPacketChannelLength:])

		packetID := fmt.Sprintf("%s/%d", channelID, sequence)
		pendingPackets = append(pendingPackets, packetID)
	}

	return pendingPackets
}

// Remove all pending sequence numbers from the store
// This is executed when the quota resets
func (k *Keeper) RemoveAllChannelPendingSendPackets(ctx sdk.Context, channelID string) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)

	iterator := storetypes.KVStorePrefixIterator(store, []byte(channelID))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}
