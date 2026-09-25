// SPDX-License-Identifier: Apache-2.0

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Before each hour epoch, check if any of the rate limits have expired,
// and reset them if they have
func (k *Keeper) BeginBlocker(ctx sdk.Context) {
	// read before CheckHourEpochStarting moves the epoch, which reports any error
	previousEpoch, _ := k.GetHourEpoch(ctx)
	epochStarting, epochNumber, err := k.CheckHourEpochStarting(ctx)
	if err != nil {
		k.Logger(ctx).Error("BeginBlocker", "error", err)
		return
	}
	if !epochStarting {
		return
	}
	for _, rateLimit := range k.GetAllRateLimits(ctx) {
		// reset if the quota period ended since the previous epoch; the epoch number can
		// move by more than one after a halt, the quota is still reset only once
		duration := rateLimit.Quota.DurationHours
		if duration == 0 || epochNumber/duration == previousEpoch.EpochNumber/duration {
			continue
		}
		if err := k.ResetRateLimit(ctx, rateLimit.Path.Denom, rateLimit.Path.ChannelOrClientId); err != nil {
			k.Logger(ctx).Error("Unable to reset quota", "Denom", rateLimit.Path.Denom, "ChannelOrClientId", rateLimit.Path.ChannelOrClientId, "error", err)
		}
	}
}
