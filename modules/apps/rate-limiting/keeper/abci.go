package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Before each hour epoch, check if any of the rate limits have expired,
// and reset them if they have
func (k Keeper) BeginBlocker(ctx sdk.Context) {
	if epochStarting, epochNumber := k.CheckHourEpochStarting(ctx); epochStarting {
		for _, rateLimit := range k.GetAllRateLimits(ctx) {
			if rateLimit.Quota.DurationHours != 0 && epochNumber%rateLimit.Quota.DurationHours == 0 {
				err := k.ResetRateLimit(ctx, rateLimit.Path.Denom, rateLimit.Path.ChannelOrClientId)
				if err != nil {
					k.Logger(ctx).Error(fmt.Sprintf("Unable to reset quota for Denom: %s, ChannelOrClientId: %s", rateLimit.Path.Denom, rateLimit.Path.ChannelOrClientId))
				}
			}
		}
	}
}
