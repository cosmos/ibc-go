package keeper

import (
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Stores the hour epoch
func (k Keeper) SetHourEpoch(ctx sdk.Context, epoch types.HourEpoch) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	epochBz := k.cdc.MustMarshal(&epoch)
	store.Set([]byte(types.HourEpochKey), epochBz)
}

// Reads the hour epoch from the store
func (k Keeper) GetHourEpoch(ctx sdk.Context) (epoch types.HourEpoch) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	epochBz := store.Get([]byte(types.HourEpochKey))
	if len(epochBz) == 0 {
		panic("Hour epoch not found")
	}

	k.cdc.MustUnmarshal(epochBz, &epoch)
	return epoch
}

// Checks if it's time to start the new hour epoch
func (k Keeper) CheckHourEpochStarting(ctx sdk.Context) (epochStarting bool, epochNumber uint64) {
	hourEpoch := k.GetHourEpoch(ctx)

	// If the block time is later than the current epoch start time + epoch duration,
	// move onto the next epoch by incrementing the epoch number, height, and start time
	currentEpochEndTime := hourEpoch.EpochStartTime.Add(hourEpoch.Duration)
	shouldNextEpochStart := ctx.BlockTime().After(currentEpochEndTime)
	if shouldNextEpochStart {
		hourEpoch.EpochNumber++
		hourEpoch.EpochStartTime = currentEpochEndTime
		hourEpoch.EpochStartHeight = ctx.BlockHeight()

		k.SetHourEpoch(ctx, hourEpoch)
		return true, hourEpoch.EpochNumber
	}

	// Otherwise, indicate that a new epoch is not starting
	return false, 0
}
