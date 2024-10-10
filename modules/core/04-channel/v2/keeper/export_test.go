package keeper

import (
	"context"

	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
)

// GetPacketAcknowledgement fetches the packet acknowledgement from the store.
func (k *Keeper) GetPacketAcknowledgement(ctx context.Context, sourceID string, sequence uint64) []byte {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(hostv2.PacketAcknowledgementKey(sourceID, sequence))
	if err != nil {
		panic(err)
	}
	return bz
}
