package keeper

import (
	"encoding/binary"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

// Sets the sequence number of a packet that was just sent
func (k *Keeper) SetPendingSendPacket(ctx sdk.Context, channelID string, sequence uint64) error {
	return k.setPendingPacket(ctx, types.PendingSendPacketPrefix, channelID, sequence)
}

// Sets the sequence number of a packet that was just received
func (k *Keeper) SetPendingReceivePacket(ctx sdk.Context, channelID string, sequence uint64) error {
	return k.setPendingPacket(ctx, types.PendingReceivePacketPrefix, channelID, sequence)
}

func (k *Keeper) setPendingPacket(ctx sdk.Context, keyPrefix []byte, channelID string, sequence uint64) error {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, keyPrefix)
	key, err := types.PendingPacketKey(channelID, sequence)
	if err != nil {
		return err
	}
	store.Set(key, []byte{1})
	return nil
}

// Remove a pending packet sequence number from the store
// Used after the ack or timeout for a packet has been received
func (k *Keeper) RemovePendingSendPacket(ctx sdk.Context, channelID string, sequence uint64) error {
	return k.removePendingPacket(ctx, types.PendingSendPacketPrefix, channelID, sequence)
}

// Remove a pending receive packet sequence number from the store
// Used after an async error acknowledgement has been written
func (k *Keeper) RemovePendingReceivePacket(ctx sdk.Context, channelID string, sequence uint64) error {
	return k.removePendingPacket(ctx, types.PendingReceivePacketPrefix, channelID, sequence)
}

func (k *Keeper) removePendingPacket(ctx sdk.Context, keyPrefix []byte, channelID string, sequence uint64) error {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, keyPrefix)
	key, err := types.PendingPacketKey(channelID, sequence)
	if err != nil {
		return err
	}

	store.Delete(key)
	return nil
}

// Checks whether the packet sequence number is in the store - indicating that it was
// sent during the current quota
func (k *Keeper) CheckPacketSentDuringCurrentQuota(ctx sdk.Context, channelID string, sequence uint64) (bool, error) {
	return k.checkPacketDuringCurrentQuota(ctx, types.PendingSendPacketPrefix, channelID, sequence)
}

// Checks whether the packet sequence number is in the store - indicating that it was
// received during the current quota
func (k *Keeper) CheckPacketReceivedDuringCurrentQuota(ctx sdk.Context, channelID string, sequence uint64) (bool, error) {
	return k.checkPacketDuringCurrentQuota(ctx, types.PendingReceivePacketPrefix, channelID, sequence)
}

func (k *Keeper) checkPacketDuringCurrentQuota(ctx sdk.Context, keyPrefix []byte, channelID string, sequence uint64) (bool, error) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, keyPrefix)
	key, err := types.PendingPacketKey(channelID, sequence)
	if err != nil {
		return false, err
	}
	valueBz := store.Get(key)
	found := len(valueBz) != 0
	return found, nil
}

// Get all pending packet sequence numbers
func (k *Keeper) GetAllPendingSendPackets(ctx sdk.Context) ([]string, error) {
	return k.getAllPendingPackets(ctx, types.PendingSendPacketPrefix)
}

func (k *Keeper) getAllPendingPackets(ctx sdk.Context, keyPrefix []byte) ([]string, error) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, keyPrefix)

	iterator := store.Iterator(nil, nil)

	pendingPackets := make([]string, 0)
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()

		channelID := string(key[:types.PendingSendPacketChannelLength])
		channelID = strings.TrimRight(channelID, "\x00") // removes null bytes from suffix
		sequence := binary.BigEndian.Uint64(key[types.PendingSendPacketChannelLength:])

		packetID := fmt.Sprintf("%s/%d", channelID, sequence)
		pendingPackets = append(pendingPackets, packetID)
	}

	return pendingPackets, iterator.Close()
}

// Remove all pending sequence numbers from the store
// This is executed when the quota resets
func (k *Keeper) RemoveAllChannelPendingSendPackets(ctx sdk.Context, channelID string) error {
	return k.removeAllChannelPendingPackets(ctx, types.PendingSendPacketPrefix, channelID)
}

// Remove all pending receive sequence numbers from the store
// This is executed when the quota resets
func (k *Keeper) RemoveAllChannelPendingReceivePackets(ctx sdk.Context, channelID string) error {
	return k.removeAllChannelPendingPackets(ctx, types.PendingReceivePacketPrefix, channelID)
}

func (k *Keeper) removeAllChannelPendingPackets(ctx sdk.Context, keyPrefix []byte, channelID string) error {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, keyPrefix)

	if len(channelID) > types.PendingSendPacketChannelLength {
		return errorsmod.Wrapf(types.ErrInvalidChannelID, "channel %s with length %d is greater than the allowed length %d", channelID, len(channelID), types.PendingSendPacketChannelLength)
	}

	channelIDBz := make([]byte, types.PendingSendPacketChannelLength)
	copy(channelIDBz, channelID)

	iterator := storetypes.KVStorePrefixIterator(store, channelIDBz)
	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
	return iterator.Close()
}
