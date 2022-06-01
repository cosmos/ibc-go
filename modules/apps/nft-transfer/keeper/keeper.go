package keeper

import (
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// Keeper defines the IBC non fungible transfer keeper
type Keeper struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec

	ics4Wrapper   types.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	nftKeeper     types.NFTKeeper
	authkeeper    types.AccountKeeper
	scopedKeeper  capabilitykeeper.ScopedKeeper
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+host.ModuleName+"-"+types.ModuleName)
}

// SetPort sets the portID for the nft-transfer module. Used in InitGenesis
func (k Keeper) SetPort(ctx sdk.Context, portID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.PortKey, []byte(portID))
}

// GetPort returns the portID for the nft-transfer module.
func (k Keeper) GetPort(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	return string(store.Get(types.PortKey))
}

// IsBound checks if the transfer module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
	_, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
	return ok
}

// BindPort defines a wrapper function for the ort Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
	cap := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, cap, host.PortPath(portID))
}

// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
func (k Keeper) AuthenticateCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) bool {
	return k.scopedKeeper.AuthenticateCapability(ctx, cap, name)
}

// ClaimCapability allows the nft-transfer module that can claim a capability that IBC module
// passes to it
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

// SetEscrowAddress attempts to save a account to auth module
func (k Keeper) SetEscrowAddress(ctx sdk.Context, portID, channelID string) {
	// create the escrow address for the tokens
	escrowAddress := types.GetEscrowAddress(portID, channelID)
	if !k.authkeeper.HasAccount(ctx, escrowAddress) {
		acc := k.authkeeper.NewAccountWithAddress(ctx, escrowAddress)
		k.authkeeper.SetAccount(ctx, acc)
	}
}
