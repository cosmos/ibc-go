package keeper

import (
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// Stores the hour epoch
func (k Keeper) SetHourEpoch(ctx sdk.Context, epoch types.HourEpoch) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	epochBz := k.cdc.MustMarshal(&epoch)
	store.Set(types.HourEpochKey, epochBz)
}

// Reads the hour epoch from the store
// Returns a zero-value epoch and logs an error if the epoch is not found or fails to unmarshal.
func (k Keeper) GetHourEpoch(ctx sdk.Context) (epoch types.HourEpoch) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	epochBz := store.Get(types.HourEpochKey)
	if len(epochBz) == 0 {
		// Log an error if the epoch key is not found (should be initialized at genesis)
		k.Logger(ctx).Error("hour epoch not found in store")
		return types.HourEpoch{} // Return zero-value epoch
	}

	if err := k.cdc.Unmarshal(epochBz, &epoch); err != nil {
		// Log an error if unmarshalling fails (indicates corrupted data)
		k.Logger(ctx).Error("failed to unmarshal hour epoch", "error", err)
		return types.HourEpoch{} // Return zero-value epoch
	}

	return epoch
}

// Checks if it's time to start the new hour epoch
func (k Keeper) CheckHourEpochStarting(ctx sdk.Context) (epochStarting bool, epochNumber uint64) {
	hourEpoch := k.GetHourEpoch(ctx)

	// If GetHourEpoch returned a zero-value epoch (due to error or missing key),
	// we cannot proceed with the check.
	if hourEpoch.Duration == 0 || hourEpoch.EpochStartTime.IsZero() {
		k.Logger(ctx).Error("cannot check hour epoch starting: epoch data is invalid or missing")
		return false, 0
	}

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
