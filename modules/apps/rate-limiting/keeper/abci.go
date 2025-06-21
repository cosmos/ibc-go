package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Before each hour epoch, check if any of the rate limits have expired,
// and reset them if they have
func (k *Keeper) BeginBlocker(ctx sdk.Context) {
	epochStarting, epochNumber, err := k.CheckHourEpochStarting(ctx)
	if err != nil {
		k.Logger(ctx).Error("BeginBlocker", "error", err)
		return
	}
	if !epochStarting {
		return
	}
	for _, rateLimit := range k.GetAllRateLimits(ctx) {
		if rateLimit.Quota.DurationHours == 0 || epochNumber%rateLimit.Quota.DurationHours != 0 {
			continue
		}
		if err := k.ResetRateLimit(ctx, rateLimit.Path.Denom, rateLimit.Path.ChannelOrClientId); err != nil {
			k.Logger(ctx).Error("Unable to reset quota", "Denom", rateLimit.Path.Denom, "ChannelOrClientId", rateLimit.Path.ChannelOrClientId, "error", err)
		}
	}
}
