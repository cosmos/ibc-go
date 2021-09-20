package keeper

import (
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
)

// Keeper defines the IBC fungible transfer keeper
type Keeper struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec

	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	scopedKeeper  capabilitykeeper.ScopedKeeper
}

// NewKeeper creates a new 29-fee Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace,
	channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper,
	scopedKeeper capabilitykeeper.ScopedKeeper,
) Keeper {

	return Keeper{
		cdc:           cdc,
		storeKey:      key,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		scopedKeeper:  scopedKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+host.ModuleName+"-"+types.ModuleName)
}

// IsBound checks if the transfer module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
	_, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
	return ok
}

// BindPort defines a wrapper function for the port Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability {
	return k.portKeeper.BindPort(ctx, portID)
}

// ChanCloseInit wraps the channel keeper's function in order to expose it to underlying app.
func (k Keeper) ChanCloseInit(ctx sdk.Context, portID, channelID string, chanCap *capabilitytypes.Capability) error {
	return k.channelKeeper.ChanCloseInit(ctx, portID, channelID, chanCap)
}

func (k Keeper) GetChannel(ctx sdk.Context, portID, channelID string) (channeltypes.Channel, bool) {
	return k.channelKeeper.GetChannel(ctx, portID, channelID)
}

func (k Keeper) GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool) {
	return k.channelKeeper.GetNextSequenceSend(ctx, portID, channelID)
}

func (k Keeper) SendPacket(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI) error {
	return k.channelKeeper.SendPacket(ctx, chanCap, packet)
}

// // GetPort returns the portID for the transfer module. Used in ExportGenesis
// func (k Keeper) GetPort(ctx sdk.Context) string {
// 	store := ctx.KVStore(k.storeKey)
// 	return string(store.Get(types.PortKey))
// }

// // SetPort sets the portID for the transfer module. Used in InitGenesis
// func (k Keeper) SetPort(ctx sdk.Context, portID string) {
// 	store := ctx.KVStore(k.storeKey)
// 	store.Set(types.PortKey, []byte(portID))
// }

// SetFeeEnabled sets a flag to determine if fee handling logic should run for the given channel
// identified by channel and port identifiers.
func (k Keeper) SetFeeEnabled(ctx sdk.Context, portID, channelID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.FeeEnabledKey(portID, channelID), []byte{1})
}

// DeleteFeeEnabled deletes the fee enabled flag for a given portID and channelID
func (k Keeper) DeleteFeeEnabled(ctx sdk.Context, portID, channelID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.FeeEnabledKey(portID, channelID))
}

// IsFeeEnabled returns whether fee handling logic should be run for the given port by checking the
// fee enabled flag for the given port and channel identifiers
func (k Keeper) IsFeeEnabled(ctx sdk.Context, portID, channelID string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Get(types.FeeEnabledKey(portID, channelID)) != nil
}
