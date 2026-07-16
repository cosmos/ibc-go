package keeper

import (
	"fmt"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

// SetPendingSendPacket records a packet whose send flow was applied and may be
// reverted later. Callers pass (channelID, sequence, denom), but the collection
// key is stored as (channelID, denom, sequence) for channel+denom range resets.
func (k *Keeper) SetPendingSendPacket(ctx sdk.Context, channelID string, sequence uint64, denom string) error {
	return setPendingPacket(ctx, k.PendingSendPackets, channelID, sequence, denom)
}

// SetPendingReceivePacket records a packet whose receive flow was applied and
// may be reverted later. Callers pass (channelID, sequence, denom), but the
// collection key is stored as (channelID, denom, sequence) for channel+denom
// range resets.
func (k *Keeper) SetPendingReceivePacket(ctx sdk.Context, channelID string, sequence uint64, denom string) error {
	return setPendingPacket(ctx, k.PendingReceivePackets, channelID, sequence, denom)
}

func setPendingPacket(ctx sdk.Context, packets collections.KeySet[collections.Triple[string, string, uint64]], channelID string, sequence uint64, denom string) error {
	key, err := pendingPacketKey(channelID, sequence, denom)
	if err != nil {
		return err
	}

	return packets.Set(ctx, key)
}

// RemovePendingSendPacket removes a send marker after the packet is finalized by
// acknowledgement or timeout.
func (k *Keeper) RemovePendingSendPacket(ctx sdk.Context, channelID string, sequence uint64, denom string) error {
	return removePendingPacket(ctx, k.PendingSendPackets, channelID, sequence, denom)
}

// RemovePendingReceivePacket removes a receive marker after a synchronous
// acknowledgement or async acknowledgement finalizes the packet.
func (k *Keeper) RemovePendingReceivePacket(ctx sdk.Context, channelID string, sequence uint64, denom string) error {
	return removePendingPacket(ctx, k.PendingReceivePackets, channelID, sequence, denom)
}

func removePendingPacket(ctx sdk.Context, packets collections.KeySet[collections.Triple[string, string, uint64]], channelID string, sequence uint64, denom string) error {
	key, err := pendingPacketKey(channelID, sequence, denom)
	if err != nil {
		return err
	}

	return packets.Remove(ctx, key)
}

// CheckPacketSentDuringCurrentQuota checks whether a send marker exists for the
// provided (channelID, sequence, denom).
func (k *Keeper) CheckPacketSentDuringCurrentQuota(ctx sdk.Context, channelID string, sequence uint64, denom string) (bool, error) {
	return checkPacketDuringCurrentQuota(ctx, k.PendingSendPackets, channelID, sequence, denom)
}

// CheckPacketReceivedDuringCurrentQuota checks whether a receive marker exists
// for the provided (channelID, sequence, denom).
func (k *Keeper) CheckPacketReceivedDuringCurrentQuota(ctx sdk.Context, channelID string, sequence uint64, denom string) (bool, error) {
	return checkPacketDuringCurrentQuota(ctx, k.PendingReceivePackets, channelID, sequence, denom)
}

func checkPacketDuringCurrentQuota(ctx sdk.Context, packets collections.KeySet[collections.Triple[string, string, uint64]], channelID string, sequence uint64, denom string) (bool, error) {
	key, err := pendingPacketKey(channelID, sequence, denom)
	if err != nil {
		return false, err
	}

	return packets.Has(ctx, key)
}

// GetAllPendingSendPackets returns all pending send markers formatted as
// {channelID}/{sequence}/{denom}.
func (k *Keeper) GetAllPendingSendPackets(ctx sdk.Context) ([]string, error) {
	return getAllPendingPackets(ctx, k.PendingSendPackets)
}

// GetAllPendingReceivePackets returns all pending receive markers formatted as
// {channelID}/{sequence}/{denom}.
func (k *Keeper) GetAllPendingReceivePackets(ctx sdk.Context) ([]string, error) {
	return getAllPendingPackets(ctx, k.PendingReceivePackets)
}

func getAllPendingPackets(ctx sdk.Context, packets collections.KeySet[collections.Triple[string, string, uint64]]) ([]string, error) {
	pendingPackets := make([]string, 0)
	err := packets.Walk(ctx, nil, func(key collections.Triple[string, string, uint64]) (bool, error) {
		packetID := fmt.Sprintf("%s/%d/%s", key.K1(), key.K3(), key.K2())
		pendingPackets = append(pendingPackets, packetID)
		return false, nil
	})

	return pendingPackets, err
}

// RemoveAllChannelPendingSendPackets removes all pending send markers for the
// given channelID and denom.
func (k *Keeper) RemoveAllChannelPendingSendPackets(ctx sdk.Context, channelID string, denom string) error {
	return removeAllChannelPendingPackets(ctx, k.PendingSendPackets, channelID, denom)
}

// RemoveAllChannelPendingReceivePackets removes all pending receive markers for
// the given channelID and denom.
func (k *Keeper) RemoveAllChannelPendingReceivePackets(ctx sdk.Context, channelID string, denom string) error {
	return removeAllChannelPendingPackets(ctx, k.PendingReceivePackets, channelID, denom)
}

func removeAllChannelPendingPackets(ctx sdk.Context, packets collections.KeySet[collections.Triple[string, string, uint64]], channelID string, denom string) error {
	if err := types.ValidatePendingPacketParts(channelID, denom); err != nil {
		return err
	}

	var keys []collections.Triple[string, string, uint64]
	if err := packets.Walk(ctx, collections.NewSuperPrefixedTripleRange[string, string, uint64](channelID, denom), func(key collections.Triple[string, string, uint64]) (bool, error) {
		keys = append(keys, key)
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

// pendingPacketKey validates the public pending packet tuple
// (channelID, sequence, denom) and returns the collection key in storage order:
// (channelID, denom, sequence).
func pendingPacketKey(channelID string, sequence uint64, denom string) (collections.Triple[string, string, uint64], error) {
	if err := types.ValidatePendingPacketParts(channelID, denom); err != nil {
		return collections.Triple[string, string, uint64]{}, err
	}

	return collections.Join3(channelID, denom, sequence), nil
}
