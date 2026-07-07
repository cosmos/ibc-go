package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

// Sets the sequence number of a packet that was just sent
func (k *Keeper) SetPendingSendPacket(ctx sdk.Context, channelID string, sequence uint64, denom string) error {
	return k.setPendingPacket(ctx, k.PendingSendPackets, channelID, sequence, denom)
}

// Sets the sequence number of a packet that was just received
func (k *Keeper) SetPendingReceivePacket(ctx sdk.Context, channelID string, sequence uint64, denom string) error {
	return k.setPendingPacket(ctx, k.PendingReceivePackets, channelID, sequence, denom)
}

func (k *Keeper) setPendingPacket(ctx sdk.Context, packets collections.KeySet[collections.Triple[string, uint64, string]], channelID string, sequence uint64, denom string) error {
	key, err := pendingPacketKey(channelID, sequence, denom)
	if err != nil {
		return err
	}

	return packets.Set(ctx, key)
}

// Remove a pending packet sequence number from the store
// Used after the ack or timeout for a packet has been received
func (k *Keeper) RemovePendingSendPacket(ctx sdk.Context, channelID string, sequence uint64, denom string) error {
	return k.removePendingPacket(ctx, k.PendingSendPackets, channelID, sequence, denom)
}

// Remove a pending receive packet sequence number from the store
// Used after an async error acknowledgement has been written
func (k *Keeper) RemovePendingReceivePacket(ctx sdk.Context, channelID string, sequence uint64, denom string) error {
	return k.removePendingPacket(ctx, k.PendingReceivePackets, channelID, sequence, denom)
}

func (k *Keeper) removePendingPacket(ctx sdk.Context, packets collections.KeySet[collections.Triple[string, uint64, string]], channelID string, sequence uint64, denom string) error {
	key, err := pendingPacketKey(channelID, sequence, denom)
	if err != nil {
		return err
	}

	return packets.Remove(ctx, key)
}

// Checks whether the packet sequence number is in the store - indicating that it was
// sent during the current quota
func (k *Keeper) CheckPacketSentDuringCurrentQuota(ctx sdk.Context, channelID string, sequence uint64, denom string) (bool, error) {
	return k.checkPacketDuringCurrentQuota(ctx, k.PendingSendPackets, channelID, sequence, denom)
}

// Checks whether the packet sequence number is in the store - indicating that it was
// received during the current quota
func (k *Keeper) CheckPacketReceivedDuringCurrentQuota(ctx sdk.Context, channelID string, sequence uint64, denom string) (bool, error) {
	return k.checkPacketDuringCurrentQuota(ctx, k.PendingReceivePackets, channelID, sequence, denom)
}

func (k *Keeper) checkPacketDuringCurrentQuota(ctx sdk.Context, packets collections.KeySet[collections.Triple[string, uint64, string]], channelID string, sequence uint64, denom string) (bool, error) {
	key, err := pendingPacketKey(channelID, sequence, denom)
	if err != nil {
		return false, err
	}

	return packets.Has(ctx, key)
}

// Get all pending packet sequence numbers
func (k *Keeper) GetAllPendingSendPackets(ctx sdk.Context) ([]string, error) {
	return k.getAllPendingPackets(ctx, k.PendingSendPackets)
}

// Get all pending receive packet sequence numbers
func (k *Keeper) GetAllPendingReceivePackets(ctx sdk.Context) ([]string, error) {
	return k.getAllPendingPackets(ctx, k.PendingReceivePackets)
}

func (k *Keeper) getAllPendingPackets(ctx sdk.Context, packets collections.KeySet[collections.Triple[string, uint64, string]]) ([]string, error) {
	pendingPackets := make([]string, 0)
	err := packets.Walk(ctx, nil, func(key collections.Triple[string, uint64, string]) (bool, error) {
		packetID := fmt.Sprintf("%s/%d/%s", key.K1(), key.K2(), key.K3())
		pendingPackets = append(pendingPackets, packetID)
		return false, nil
	})

	return pendingPackets, err
}

// Remove all pending sequence numbers from the store
// This is executed when the quota resets
func (k *Keeper) RemoveAllChannelPendingSendPackets(ctx sdk.Context, channelID string, denom string) error {
	return k.removeAllChannelPendingPackets(ctx, k.PendingSendPackets, channelID, denom)
}

// Remove all pending receive sequence numbers from the store
// This is executed when the quota resets
func (k *Keeper) RemoveAllChannelPendingReceivePackets(ctx sdk.Context, channelID string, denom string) error {
	return k.removeAllChannelPendingPackets(ctx, k.PendingReceivePackets, channelID, denom)
}

func (k *Keeper) removeAllChannelPendingPackets(ctx sdk.Context, packets collections.KeySet[collections.Triple[string, uint64, string]], channelID string, denom string) error {
	if _, err := pendingPacketKey(channelID, 1, denom); err != nil {
		return err
	}

	var keys []collections.Triple[string, uint64, string]
	if err := packets.Walk(ctx, collections.NewPrefixedTripleRange[string, uint64, string](channelID), func(key collections.Triple[string, uint64, string]) (bool, error) {
		if key.K3() == denom {
			keys = append(keys, key)
		}
		return false, nil
	}); err != nil {
		return err
	}

	for _, key := range keys {
		if err := packets.Remove(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

func pendingPacketKey(channelID string, sequence uint64, denom string) (collections.Triple[string, uint64, string], error) {
	if _, err := types.PendingPacketKey(channelID, sequence); err != nil {
		return collections.Triple[string, uint64, string]{}, err
	}
	if denom == "" {
		return collections.Triple[string, uint64, string]{}, fmt.Errorf("pending packet denom must be specified")
	}

	return collections.Join3(channelID, sequence, denom), nil
}
