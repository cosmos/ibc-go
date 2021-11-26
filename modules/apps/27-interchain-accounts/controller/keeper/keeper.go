package keeper

import (
	"fmt"

	baseapp "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/tendermint/tendermint/libs/log"

	icakeeper "github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/keeper"
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

// Keeper defines the IBC interchain accounts controller keeper
type Keeper struct {
	cdc         codec.BinaryCodec
	storePrefix string

	icaKeeper     icakeeper.Keeper
	ics4Wrapper   types.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	accountKeeper types.AccountKeeper

	scopedKeeper capabilitykeeper.ScopedKeeper

	msgRouter *baseapp.MsgServiceRouter
}

// NewKeeper creates a new interchain accounts controller Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, storePrefix string, icaKeeper icakeeper.Keeper,
	ics4Wrapper types.ICS4Wrapper, channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper,
	accountKeeper types.AccountKeeper, scopedKeeper capabilitykeeper.ScopedKeeper, msgRouter *baseapp.MsgServiceRouter,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storePrefix:   storePrefix,
		icaKeeper:     icaKeeper,
		ics4Wrapper:   ics4Wrapper,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		accountKeeper: accountKeeper,
		scopedKeeper:  scopedKeeper,
		msgRouter:     msgRouter,
	}
}

// Logger returns the application logger, scoped to the associated module
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s-%s", host.ModuleName, types.ModuleName))
}

// IsBound checks if the interchain account controller module is already bound to the desired port
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

// GetAllPorts returns all ports to which the interchain accounts controller module is bound. Used in ExportGenesis
func (k Keeper) GetAllPorts(ctx sdk.Context) []string {
	return k.icaKeeper.GetAllPorts(ctx, k.storePrefix)
}

// BindPort stores the provided portID and binds to it, returning the associated capability
func (k Keeper) BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability {
	return k.icaKeeper.BindPort(ctx, k.storePrefix, portID)
}

// GetActiveChannelID retrieves the active channelID from the store keyed by the provided portID
func (k Keeper) GetActiveChannelID(ctx sdk.Context, portID string) (string, bool) {
	return k.icaKeeper.GetActiveChannelID(ctx, k.storePrefix, portID)
}

// GetAllActiveChannels returns a list of all active interchain accounts controller channels and their associated port identifiers
func (k Keeper) GetAllActiveChannels(ctx sdk.Context) []types.ActiveChannel {
	return k.icaKeeper.GetAllActiveChannels(ctx, k.storePrefix)
}

// SetActiveChannelID stores the active channelID, keyed by the provided portID
func (k Keeper) SetActiveChannelID(ctx sdk.Context, portID, channelID string) {
	k.icaKeeper.SetActiveChannelID(ctx, k.storePrefix, portID, channelID)
}

// DeleteActiveChannelID removes the active channel keyed by the provided portID stored in state
func (k Keeper) DeleteActiveChannelID(ctx sdk.Context, portID string) {
	k.icaKeeper.DeleteActiveChannelID(ctx, k.storePrefix, portID)
}

// IsActiveChannel returns true if there exists an active channel for the provided portID, otherwise false
func (k Keeper) IsActiveChannel(ctx sdk.Context, portID string) bool {
	return k.icaKeeper.IsActiveChannel(ctx, k.storePrefix, portID)
}

// GetInterchainAccountAddress retrieves the InterchainAccount address from the store keyed by the provided portID
func (k Keeper) GetInterchainAccountAddress(ctx sdk.Context, portID string) (string, bool) {
	return k.icaKeeper.GetInterchainAccountAddress(ctx, k.storePrefix, portID)
}

// GetAllInterchainAccounts returns a list of all registered interchain account addresses and their associated controller port identifiers
func (k Keeper) GetAllInterchainAccounts(ctx sdk.Context) []types.RegisteredInterchainAccount {
	return k.icaKeeper.GetAllInterchainAccounts(ctx, k.storePrefix)
}

// SetInterchainAccountAddress stores the InterchainAccount address, keyed by the associated portID
func (k Keeper) SetInterchainAccountAddress(ctx sdk.Context, portID string, address string) {
	k.icaKeeper.SetInterchainAccountAddress(ctx, k.storePrefix, portID, address)
}
