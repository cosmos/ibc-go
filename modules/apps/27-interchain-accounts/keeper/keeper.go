package keeper

import (
	"fmt"
	"strings"

	baseapp "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

// Keeper defines the IBC interchain accounts controller keeper
type Keeper struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec

	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	accountKeeper types.AccountKeeper

	scopedKeeper capabilitykeeper.ScopedKeeper

	msgRouter *baseapp.MsgServiceRouter
}

// NewKeeper creates a new interchain accounts controller Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey,
	channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper,
	accountKeeper types.AccountKeeper, scopedKeeper capabilitykeeper.ScopedKeeper, msgRouter *baseapp.MsgServiceRouter,
) Keeper {
	return Keeper{
		storeKey:      key,
		cdc:           cdc,
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

// prefixStore returns isolated prefix store for each ICA submodule so they can read/write in separate
// namespace without being able to read/write to each other's data.
func (k Keeper) prefixStore(ctx sdk.Context, storePrefix string) sdk.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), []byte(storePrefix))
}

// GetAllPorts returns all ports to which the interchain accounts controller module is bound. Used in ExportGenesis
func (k Keeper) GetAllPorts(ctx sdk.Context, storePrefix string) []string {
	store := k.prefixStore(ctx, storePrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.PortKeyPrefix))
	defer iterator.Close()

	var ports []string
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		ports = append(ports, keySplit[1])
	}

	return ports
}

// BindPort stores the provided portID and binds to it, returning the associated capability
func (k Keeper) BindPort(ctx sdk.Context, storePrefix, portID string) *capabilitytypes.Capability {
	store := k.prefixStore(ctx, storePrefix)
	store.Set(types.KeyPort(portID), []byte{0x01})

	return k.portKeeper.BindPort(ctx, portID)
}

// IsBound checks if the interchain account controller module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, storePreifx, portID string) bool {
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

// GetActiveChannelID retrieves the active channelID from the store keyed by the provided portID
func (k Keeper) GetActiveChannelID(ctx sdk.Context, storePrefix, portID string) (string, bool) {
	store := k.prefixStore(ctx, storePrefix)
	key := types.KeyActiveChannel(portID)

	if !store.Has(key) {
		return "", false
	}

	return string(store.Get(key)), true
}

// GetAllActiveChannels returns a list of all active interchain accounts controller channels and their associated port identifiers
func (k Keeper) GetAllActiveChannels(ctx sdk.Context, storePrefix string) []types.ActiveChannel {
	store := k.prefixStore(ctx, storePrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.ActiveChannelKeyPrefix))
	defer iterator.Close()

	var activeChannels []types.ActiveChannel
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		ch := types.ActiveChannel{
			PortId:    keySplit[1],
			ChannelId: string(iterator.Value()),
		}

		activeChannels = append(activeChannels, ch)
	}

	return activeChannels
}

// SetActiveChannelID stores the active channelID, keyed by the provided portID
func (k Keeper) SetActiveChannelID(ctx sdk.Context, storePrefix, portID, channelID string) {
	store := k.prefixStore(ctx, storePrefix)
	store.Set(types.KeyActiveChannel(portID), []byte(channelID))
}

// DeleteActiveChannelID removes the active channel keyed by the provided portID stored in state
func (k Keeper) DeleteActiveChannelID(ctx sdk.Context, storePrefix, portID string) {
	store := k.prefixStore(ctx, storePrefix)
	store.Delete(types.KeyActiveChannel(portID))
}

// IsActiveChannel returns true if there exists an active channel for the provided portID, otherwise false
func (k Keeper) IsActiveChannel(ctx sdk.Context, storePrefix, portID string) bool {
	_, ok := k.GetActiveChannelID(ctx, storePrefix, portID)
	return ok
}

// GetInterchainAccountAddress retrieves the InterchainAccount address from the store keyed by the provided portID
func (k Keeper) GetInterchainAccountAddress(ctx sdk.Context, storePrefix, portID string) (string, bool) {
	store := k.prefixStore(ctx, storePrefix)
	key := types.KeyOwnerAccount(portID)

	if !store.Has(key) {
		return "", false
	}

	return string(store.Get(key)), true
}

// GetAllInterchainAccounts returns a list of all registered interchain account addresses and their associated controller port identifiers
func (k Keeper) GetAllInterchainAccounts(ctx sdk.Context, storePrefix string) []types.RegisteredInterchainAccount {
	store := k.prefixStore(ctx, storePrefix)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.OwnerKeyPrefix))

	var interchainAccounts []types.RegisteredInterchainAccount
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		acc := types.RegisteredInterchainAccount{
			PortId:         keySplit[1],
			AccountAddress: string(iterator.Value()),
		}

		interchainAccounts = append(interchainAccounts, acc)
	}

	return interchainAccounts
}

// SetInterchainAccountAddress stores the InterchainAccount address, keyed by the associated portID
func (k Keeper) SetInterchainAccountAddress(ctx sdk.Context, storePrefix, portID string, address string) {
	store := k.prefixStore(ctx, storePrefix)
	store.Set(types.KeyOwnerAccount(portID), []byte(address))
}
