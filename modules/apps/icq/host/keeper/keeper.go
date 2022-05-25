package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	icqtypes "github.com/cosmos/ibc-go/v3/modules/apps/icq/types"
	"github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/tendermint/tendermint/libs/log"
)

// Keeper defines the IBC interchain query host keeper
type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace

	channelKeeper icqtypes.ChannelKeeper
	portKeeper    icqtypes.PortKeeper

	scopedKeeper capabilitykeeper.ScopedKeeper

	querier sdk.Queryable
}

// NewKeeper creates a new interchain accounts host Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace,
	channelKeeper icqtypes.ChannelKeeper, portKeeper icqtypes.PortKeeper,
	scopedKeeper capabilitykeeper.ScopedKeeper, querier sdk.Queryable,
) Keeper {
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		storeKey:      key,
		cdc:           cdc,
		paramSpace:    paramSpace,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		scopedKeeper:  scopedKeeper,
		querier:       querier,
	}
}

// Logger returns the application logger, scoped to the associated module
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s-%s", host.ModuleName, icqtypes.ModuleName))
}

// BindPort stores the provided portID and binds to it, returning the associated capability
func (k Keeper) BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability {
	store := ctx.KVStore(k.storeKey)
	store.Set(icqtypes.KeyPort(portID), []byte{0x01})

	return k.portKeeper.BindPort(ctx, portID)
}

// IsBound checks if the interchain account host module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
	_, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
	return ok
}

// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
func (k Keeper) AuthenticateCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) bool {
	return k.scopedKeeper.AuthenticateCapability(ctx, cap, name)
}

// ClaimCapability wraps the scopedKeeper's ClaimCapability function
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}
