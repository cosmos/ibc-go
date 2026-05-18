package keeper

import (
	"encoding/binary"
	"errors"
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
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)
	key, err := types.PendingSendPacketKey(channelID, sequence)
	if err != nil {
		return err
	}
	store.Set(key, []byte{1})
	return nil
}

// Remove a pending packet sequence number from the store
// Used after the ack or timeout for a packet has been received
func (k *Keeper) RemovePendingSendPacket(ctx sdk.Context, channelID string, sequence uint64) error {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)
	key, err := types.PendingSendPacketKey(channelID, sequence)
	if err != nil {
		return err
	}

	store.Delete(key)
	return nil
}

// Checks whether the packet sequence number is in the store - indicating that it was
// sent during the current quota
func (k *Keeper) CheckPacketSentDuringCurrentQuota(ctx sdk.Context, channelID string, sequence uint64) (bool, error) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)
	key, err := types.PendingSendPacketKey(channelID, sequence)
	if err != nil {
		return false, err
	}
	valueBz := store.Get(key)
	found := len(valueBz) != 0
	return found, nil
}

// Get all pending packet sequence numbers
func (k *Keeper) GetAllPendingSendPackets(ctx sdk.Context) (pendingPackets []string, err error) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)

	iterator := store.Iterator(nil, nil)
	defer func() {
		err = errors.Join(err, iterator.Close())
	}()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()

		channelID := string(key[:types.PendingSendPacketChannelLength])
		channelID = strings.TrimRight(channelID, "\x00") // removes null bytes from suffix
		sequence := binary.BigEndian.Uint64(key[types.PendingSendPacketChannelLength:])

		packetID := fmt.Sprintf("%s/%d", channelID, sequence)
		pendingPackets = append(pendingPackets, packetID)
	}

	return pendingPackets, nil
}

// Remove all pending sequence numbers from the store
// This is executed when the quota resets
func (k *Keeper) RemoveAllChannelPendingSendPackets(ctx sdk.Context, channelID string) error {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.PendingSendPacketPrefix)

	if len(channelID) > types.PendingSendPacketChannelLength {
		return errorsmod.Wrapf(types.ErrInvalidChannelID, "channel %s with length %d is greater than the allowed length %d", channelID, len(channelID), types.PendingSendPacketChannelLength)
	}

	channelIDBz := make([]byte, types.PendingSendPacketChannelLength)
	copy(channelIDBz, channelID)

	iterator := storetypes.KVStorePrefixIterator(store, channelIDBz)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
	return nil
}
