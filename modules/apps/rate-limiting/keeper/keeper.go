package keeper

import (
	"errors"
	"fmt"
	"strings"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
)

// Keeper maintains the link to storage and exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.BinaryCodec

	ics4Wrapper   porttypes.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	clientKeeper  types.ClientKeeper

	bankKeeper types.BankKeeper
	authority  string
}

// NewKeeper creates a new rate-limiting Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService corestore.KVStoreService,
	ics4Wrapper porttypes.ICS4Wrapper,
	channelKeeper types.ChannelKeeper,
	clientKeeper types.ClientKeeper,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	return Keeper{
		cdc:           cdc,
		storeService:  storeService,
		ics4Wrapper:   ics4Wrapper,
		channelKeeper: channelKeeper,
		clientKeeper:  clientKeeper,
		bankKeeper:    bankKeeper,
		authority:     authority,
	}
}

// SetICS4Wrapper sets the ICS4Wrapper.
// It is used after the middleware is created since the keeper needs the underlying module's SendPacket capability,
// creating a dependency cycle.
func (k *Keeper) SetICS4Wrapper(ics4Wrapper porttypes.ICS4Wrapper) {
	k.ics4Wrapper = ics4Wrapper
}

// ICS4Wrapper returns the ICS4Wrapper to send packets downstream.
func (k *Keeper) ICS4Wrapper() porttypes.ICS4Wrapper {
	return k.ics4Wrapper
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (Keeper) Logger(ctx sdk.Context) log.Logger {
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

// GetParams returns the current rate-limiting module parameters
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var params types.Params
	store := k.storeService.OpenKVStore(ctx)

	// Try to get Enabled parameter
	bz, err := store.Get(types.KeyEnabled)
	if err != nil {
		panic(err)
	}
	if len(bz) > 0 {
		params.Enabled = bz[0] == 1
	} else {
		params.Enabled = types.DefaultParams().Enabled
	}

	// Try to get DefaultMaxOutflow parameter
	bz, err = store.Get(types.KeyDefaultMaxOutflow)
	if err != nil {
		panic(err)
	}
	if len(bz) > 0 {
		params.DefaultMaxOutflow = string(bz)
	} else {
		params.DefaultMaxOutflow = types.DefaultParams().DefaultMaxOutflow
	}

	// Try to get DefaultMaxInflow parameter
	bz, err = store.Get(types.KeyDefaultMaxInflow)
	if err != nil {
		panic(err)
	}
	if len(bz) > 0 {
		params.DefaultMaxInflow = string(bz)
	} else {
		params.DefaultMaxInflow = types.DefaultParams().DefaultMaxInflow
	}

	// Try to get DefaultPeriod parameter
	bz, err = store.Get(types.KeyDefaultPeriod)
	if err != nil {
		panic(err)
	}
	if len(bz) > 0 && len(bz) == 8 { // uint64 is 8 bytes
		params.DefaultPeriod = sdk.BigEndianToUint64(bz)
	} else {
		params.DefaultPeriod = types.DefaultParams().DefaultPeriod
	}

	return params
}

// SetParams sets the rate-limiting module parameters
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := k.storeService.OpenKVStore(ctx)

	// Set Enabled parameter
	if params.Enabled {
		if err := store.Set(types.KeyEnabled, []byte{1}); err != nil {
			panic(err)
		}
	} else {
		if err := store.Set(types.KeyEnabled, []byte{0}); err != nil {
			panic(err)
		}
	}

	// Set DefaultMaxOutflow parameter
	if err := store.Set(types.KeyDefaultMaxOutflow, []byte(params.DefaultMaxOutflow)); err != nil {
		panic(err)
	}

	// Set DefaultMaxInflow parameter
	if err := store.Set(types.KeyDefaultMaxInflow, []byte(params.DefaultMaxInflow)); err != nil {
		panic(err)
	}

	// Set DefaultPeriod parameter
	if err := store.Set(types.KeyDefaultPeriod, sdk.Uint64ToBigEndian(params.DefaultPeriod)); err != nil {
		panic(err)
	}
}

// IsRateLimitEnabled checks if rate limiting is enabled globally
func (k Keeper) IsRateLimitEnabled(ctx sdk.Context) bool {
	params := k.GetParams(ctx)
	return params.Enabled
}
