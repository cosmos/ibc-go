package keeper

import (
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// Stores/Updates a rate limit object in the store
func (k *Keeper) SetRateLimit(ctx sdk.Context, rateLimit types.RateLimit) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.RateLimitKeyPrefix)

	rateLimitKey := types.RateLimitItemKey(rateLimit.Path.Denom, rateLimit.Path.ChannelOrClientId)
	rateLimitValue := k.cdc.MustMarshal(&rateLimit)

	store.Set(rateLimitKey, rateLimitValue)
}

// Removes a rate limit object from the store using denom and channel-id
func (k *Keeper) RemoveRateLimit(ctx sdk.Context, denom string, channelID string) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.RateLimitKeyPrefix)
	rateLimitKey := types.RateLimitItemKey(denom, channelID)
	store.Delete(rateLimitKey)
}

// Grabs and returns a rate limit object from the store using denom and channel-id
func (k *Keeper) GetRateLimit(ctx sdk.Context, denom string, channelID string) (types.RateLimit, bool) {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.RateLimitKeyPrefix)

	rateLimitKey := types.RateLimitItemKey(denom, channelID)
	rateLimitValue := store.Get(rateLimitKey)

	if len(rateLimitValue) == 0 {
		return types.RateLimit{}, false
	}

	var rateLimit types.RateLimit
	k.cdc.MustUnmarshal(rateLimitValue, &rateLimit)
	return rateLimit, true
}

// Returns all rate limits stored
func (k *Keeper) GetAllRateLimits(ctx sdk.Context) []types.RateLimit {
	adapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(adapter, types.RateLimitKeyPrefix)

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	allRateLimits := []types.RateLimit{}
	for ; iterator.Valid(); iterator.Next() {
		rateLimit := types.RateLimit{}
		if err := k.cdc.Unmarshal(iterator.Value(), &rateLimit); err != nil {
			// Log the error and skip this entry if unmarshalling fails
			k.Logger(ctx).Error("failed to unmarshal rate limit", "key", string(iterator.Key()), "error", err)
			continue
		}
		allRateLimits = append(allRateLimits, rateLimit)
	}

	return allRateLimits
}

// Adds a new rate limit. Fails if the rate limit already exists or the channel value is 0
func (k *Keeper) AddRateLimit(ctx sdk.Context, msg *types.MsgAddRateLimit) error {
	channelValue := k.GetChannelValue(ctx, msg.Denom)
	if channelValue.IsZero() {
		return types.ErrZeroChannelValue
	}

	_, found := k.GetRateLimit(ctx, msg.Denom, msg.ChannelOrClientId)
	if found {
		return types.ErrRateLimitAlreadyExists
	}

	// Confirm the channel or client exists
	_, found = k.channelKeeper.GetChannel(ctx, transfertypes.PortID, msg.ChannelOrClientId)
	if !found {
		// Check if the channelId is actually a clientId
		status := k.clientKeeper.GetClientStatus(ctx, msg.ChannelOrClientId)
		// If the status is Unauthorized or Unknown, it means the client doesn't exist or is invalid
		if status == ibcexported.Unknown || status == ibcexported.Unauthorized {
			// Return specific error indicating neither channel nor client was found
			return types.ErrChannelNotFound
		}
		// If status is Active, Expired, or Frozen, the client exists, proceed.
	}

	// Create and store the rate limit object
	path := types.Path{
		Denom:             msg.Denom,
		ChannelOrClientId: msg.ChannelOrClientId,
	}
	quota := types.Quota{
		MaxPercentSend: msg.MaxPercentSend,
		MaxPercentRecv: msg.MaxPercentRecv,
		DurationHours:  msg.DurationHours,
	}
	flow := types.Flow{
		Inflow:       sdkmath.ZeroInt(),
		Outflow:      sdkmath.ZeroInt(),
		ChannelValue: channelValue,
	}

	k.SetRateLimit(ctx, types.RateLimit{
		Path:  &path,
		Quota: &quota,
		Flow:  &flow,
	})

	return nil
}

// Updates an existing rate limit. Fails if the rate limit doesn't exist
func (k *Keeper) UpdateRateLimit(ctx sdk.Context, msg *types.MsgUpdateRateLimit) error {
	_, found := k.GetRateLimit(ctx, msg.Denom, msg.ChannelOrClientId)
	if !found {
		return types.ErrRateLimitNotFound
	}

	// Update the rate limit object with the new quota information
	// The flow should also get reset to 0
	path := types.Path{
		Denom:             msg.Denom,
		ChannelOrClientId: msg.ChannelOrClientId,
	}
	quota := types.Quota{
		MaxPercentSend: msg.MaxPercentSend,
		MaxPercentRecv: msg.MaxPercentRecv,
		DurationHours:  msg.DurationHours,
	}
	flow := types.Flow{
		Inflow:       sdkmath.ZeroInt(),
		Outflow:      sdkmath.ZeroInt(),
		ChannelValue: k.GetChannelValue(ctx, msg.Denom),
	}

	k.SetRateLimit(ctx, types.RateLimit{
		Path:  &path,
		Quota: &quota,
		Flow:  &flow,
	})

	return nil
}

// Reset the rate limit after expiration
// The inflow and outflow should get reset to 0, the channelValue should be updated,
// and all pending send packet sequence numbers should be removed
func (k *Keeper) ResetRateLimit(ctx sdk.Context, denom string, channelID string) error {
	rateLimit, found := k.GetRateLimit(ctx, denom, channelID)
	if !found {
		return types.ErrRateLimitNotFound
	}

	flow := types.Flow{
		Inflow:       sdkmath.ZeroInt(),
		Outflow:      sdkmath.ZeroInt(),
		ChannelValue: k.GetChannelValue(ctx, denom),
	}
	rateLimit.Flow = &flow

	k.SetRateLimit(ctx, rateLimit)
	k.RemoveAllChannelPendingSendPackets(ctx, channelID)
	return nil
}
