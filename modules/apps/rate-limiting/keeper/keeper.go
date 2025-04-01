package keeper

import (
	"errors"
	"strings"
	"fmt"

	"cosmossdk.io/log"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// Keeper maintains the link to storage and exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	storeService corestore.KVStoreService
	cdc      codec.BinaryCodec

	ics4Wrapper   porttypes.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	accountKeeper    types.AccountKeeper
	
	bankKeeper    types.BankKeeper
	msgRouter types.MessageRouter
	queryRouter types.QueryRouter
	authority string
}

// NewKeeper creates a new rate-limiting Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService corestore.KVStoreService,
	ics4Wrapper porttypes.ICS4Wrapper,
	channelKeeper types.ChannelKeeper,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	msgRouter types.MessageRouter,
	queryRouter types.QueryRouter,
	authority string,
) Keeper {
	// set KeyTable if it has not already been set
	// if !paramSpace.HasKeyTable() {
	// 	paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	// }

	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	return Keeper{
		cdc:      cdc,
		storeService: storeService,
		ics4Wrapper: ics4Wrapper,
		channelKeeper: channelKeeper,
		accountKeeper: accountKeeper,
		bankKeeper: bankKeeper,
		msgRouter: msgRouter,
		authority: authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetPort returns the portID for the rate-limiting module.
func (k Keeper) GetPort(ctx sdk.Context) string {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.KeyPort(types.PortID))
	if err != nil {
		panic(err)
	}
	return string(bz)
}

// SetPort sets the portID for the rate-limiting module.
func (k Keeper) setPort(ctx sdk.Context, portID string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.KeyPort(portID), []byte{0x01}); err != nil {
		panic(err)
	}
}

// // GetRateLimit returns a rate limit by channel ID and denom ID
// func (k Keeper) GetRateLimit(ctx sdk.Context, channelID, denomID string) (types.RateLimit, bool) {

// }

// // SetRateLimit sets a rate limit for a specific channel and denom
// func (k Keeper) SetRateLimit(ctx sdk.Context, rateLimit types.RateLimit) {
// 	store := k.storeService.OpenKVStore(ctx)
// 	key := types.KeyRateLimitItem(rateLimit.ChannelID, rateLimit.DenomID)

// 	bz := k.cdc.MustMarshal(&rateLimit)
// 	store.Set(key, bz)
// }

// // DeleteRateLimit deletes a rate limit for a specific channel and denom
// func (k Keeper) DeleteRateLimit(ctx sdk.Context, channelID, denomID string) {
// 	store := k.storeService.OpenKVStore(ctx)
// 	key := types.KeyRateLimitItem(channelID, denomID)
// 	store.Delete(key)
// }

// // GetAllRateLimits returns all rate limits
// func (k Keeper) GetAllRateLimits(ctx sdk.Context) []types.RateLimit {
// 	store := k.storeService.OpenKVStore(ctx)
// 	iterator := storetypes.KVStorePrefixIterator(store, []byte(types.RateLimitKeyPrefix))
// 	defer iterator.Close()

// 	var rateLimits []types.RateLimit
// 	for ; iterator.Valid(); iterator.Next() {
// 		var rateLimit types.RateLimit
// 		k.cdc.MustUnmarshal(iterator.Value(), &rateLimit)
// 		rateLimits = append(rateLimits, rateLimit)
// 	}

// 	return rateLimits
// }

// // IsRateLimitEnabled checks if rate limiting is enabled globally
// func (k Keeper) IsRateLimitEnabled(ctx sdk.Context) bool {
// 	var enabled bool
// 	k.paramSpace.Get(ctx, types.KeyEnabled, &enabled)
// 	return enabled
// }

// // GetParams returns the current rate-limiting module parameters
// func (k Keeper) GetParams(ctx sdk.Context) types.Params {
// 	var params types.Params
// 	k.paramSpace.GetParamSet(ctx, &params)
// 	return params
// }

// // SetParams sets the rate-limiting module parameters
// func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
// 	k.paramSpace.SetParamSet(ctx, &params)
// }