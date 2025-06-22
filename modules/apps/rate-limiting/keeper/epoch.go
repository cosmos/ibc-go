package keeper

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// Stores the hour epoch
func (k *Keeper) SetHourEpoch(ctx sdk.Context, epoch types.HourEpoch) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	epochBz, err := k.cdc.Marshal(&epoch)
	if err != nil {
		return err
	}
	store.Set(types.HourEpochKey, epochBz)
	return nil
}

// Reads the hour epoch from the store
// Returns a zero-value epoch and logs an error if the epoch is not found or fails to unmarshal.
func (k *Keeper) GetHourEpoch(ctx sdk.Context) (types.HourEpoch, error) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))

	var epoch types.HourEpoch
	epochBz := store.Get(types.HourEpochKey)
	if len(epochBz) == 0 {
		return types.HourEpoch{}, types.ErrEpochNotFound
	}

	if err := k.cdc.Unmarshal(epochBz, &epoch); err != nil {
		return types.HourEpoch{}, errorsmod.Wrapf(types.ErrUnmarshalEpoch, "error: %s", err.Error())
	}

	return epoch, nil
}

// Checks if it's time to start the new hour epoch.
// This function returns epochStarting, epochNumber and a possible error.
func (k *Keeper) CheckHourEpochStarting(ctx sdk.Context) (bool, uint64, error) {
	hourEpoch, err := k.GetHourEpoch(ctx)
	if err != nil {
		return false, 0, err
	}

	// If GetHourEpoch returned a zero-value epoch (due to error or missing key),
	// we cannot proceed with the check.
	if hourEpoch.Duration == 0 || hourEpoch.EpochStartTime.IsZero() {
		return false, 0, errorsmod.Wrapf(types.ErrInvalidEpoce, "cannot check hour epoch starting. epoch: %v", hourEpoch)
	}

	// If the block time is later than the current epoch start time + epoch duration,
	// move onto the next epoch by incrementing the epoch number, height, and start time
	currentEpochEndTime := hourEpoch.EpochStartTime.Add(hourEpoch.Duration)
	shouldNextEpochStart := ctx.BlockTime().After(currentEpochEndTime)
	if shouldNextEpochStart {
		hourEpoch.EpochNumber++
		hourEpoch.EpochStartTime = currentEpochEndTime
		hourEpoch.EpochStartHeight = ctx.BlockHeight()

		if err := k.SetHourEpoch(ctx, hourEpoch); err != nil {
			return false, 0, err
		}
		return true, hourEpoch.EpochNumber, nil
	}

	// Otherwise, indicate that a new epoch is not starting
	return false, 0, nil
}
