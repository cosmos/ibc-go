package keeper

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"cosmossdk.io/core/appmodule"
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	genesistypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	coretypes "github.com/cosmos/ibc-go/v9/modules/core/types"
)

// Keeper defines the IBC interchain accounts controller keeper
type Keeper struct {
	appmodule.Environment

	cdc            codec.Codec
	legacySubspace icatypes.ParamSubspace
	ics4Wrapper    porttypes.ICS4Wrapper
	channelKeeper  icatypes.ChannelKeeper

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

// NewKeeper creates a new interchain accounts controller Keeper instance
func NewKeeper(
	cdc codec.Codec, env appmodule.Environment, legacySubspace icatypes.ParamSubspace,
	ics4Wrapper porttypes.ICS4Wrapper, channelKeeper icatypes.ChannelKeeper, authority string,
) Keeper {
	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	return Keeper{
		Environment:    env,
		cdc:            cdc,
		legacySubspace: legacySubspace,
		ics4Wrapper:    ics4Wrapper,
		channelKeeper:  channelKeeper,
		authority:      authority,
	}
}

// WithICS4Wrapper sets the ICS4Wrapper. This function may be used after
// the keepers creation to set the middleware which is above this module
// in the IBC application stack.
func (k *Keeper) WithICS4Wrapper(wrapper porttypes.ICS4Wrapper) {
	k.ics4Wrapper = wrapper
}

// GetICS4Wrapper returns the ICS4Wrapper.
func (k Keeper) GetICS4Wrapper() porttypes.ICS4Wrapper {
	return k.ics4Wrapper
}

// GetConnectionID returns the connection id for the given port and channelIDs.
func (k Keeper) GetConnectionID(ctx context.Context, portID, channelID string) (string, error) {
	channel, found := k.channelKeeper.GetChannel(ctx, portID, channelID)
	if !found {
		return "", errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}
	return channel.ConnectionHops[0], nil
}

// GetAllPorts returns all ports to which the interchain accounts controller module is bound. Used in ExportGenesis
func (k Keeper) GetAllPorts(ctx context.Context) []string {
	store := runtime.KVStoreAdapter(k.KVStoreService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, []byte(icatypes.PortKeyPrefix))
	defer coretypes.LogDeferred(k.Logger, func() error { return iterator.Close() })

	var ports []string
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		ports = append(ports, keySplit[1])
	}

	return ports
}

// setPort sets the provided portID in state
func (k Keeper) setPort(ctx context.Context, portID string) {
	store := k.KVStoreService.OpenKVStore(ctx)
	if err := store.Set(icatypes.KeyPort(portID), []byte{0x01}); err != nil {
		panic(err)
	}
}

// GetAppVersion calls the ICS4Wrapper GetAppVersion function.
func (k Keeper) GetAppVersion(ctx context.Context, portID, channelID string) (string, bool) {
	return k.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// GetActiveChannelID retrieves the active channelID from the store, keyed by the provided connectionID and portID
func (k Keeper) GetActiveChannelID(ctx context.Context, connectionID, portID string) (string, bool) {
	store := k.KVStoreService.OpenKVStore(ctx)
	key := icatypes.KeyActiveChannel(portID, connectionID)

	bz, err := store.Get(key)
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return "", false
	}

	return string(bz), true
}

// GetOpenActiveChannel retrieves the active channelID from the store, keyed by the provided connectionID and portID & checks if the channel in question is in state OPEN
func (k Keeper) GetOpenActiveChannel(ctx context.Context, connectionID, portID string) (string, bool) {
	channelID, found := k.GetActiveChannelID(ctx, connectionID, portID)
	if !found {
		return "", false
	}

	channel, found := k.channelKeeper.GetChannel(ctx, portID, channelID)

	if found && channel.State == channeltypes.OPEN {
		return channelID, true
	}

	return "", false
}

// IsActiveChannelClosed retrieves the active channel from the store and returns true if the channel state is CLOSED, otherwise false
func (k Keeper) IsActiveChannelClosed(ctx context.Context, connectionID, portID string) bool {
	channelID, found := k.GetActiveChannelID(ctx, connectionID, portID)
	if !found {
		return false
	}

	channel, found := k.channelKeeper.GetChannel(ctx, portID, channelID)
	return found && channel.State == channeltypes.CLOSED
}

// GetAllActiveChannels returns a list of all active interchain accounts controller channels and their associated connection and port identifiers
func (k Keeper) GetAllActiveChannels(ctx context.Context) []genesistypes.ActiveChannel {
	store := runtime.KVStoreAdapter(k.KVStoreService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, []byte(icatypes.ActiveChannelKeyPrefix))
	defer coretypes.LogDeferred(k.Logger, func() error { return iterator.Close() })

	var activeChannels []genesistypes.ActiveChannel
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		portID := keySplit[1]
		connectionID := keySplit[2]
		channelID := string(iterator.Value())

		ch := genesistypes.ActiveChannel{
			ConnectionId:        connectionID,
			PortId:              portID,
			ChannelId:           channelID,
			IsMiddlewareEnabled: k.IsMiddlewareEnabled(ctx, portID, connectionID),
		}

		activeChannels = append(activeChannels, ch)
	}

	return activeChannels
}

// SetActiveChannelID stores the active channelID, keyed by the provided connectionID and portID
func (k Keeper) SetActiveChannelID(ctx context.Context, connectionID, portID, channelID string) {
	store := k.KVStoreService.OpenKVStore(ctx)
	if err := store.Set(icatypes.KeyActiveChannel(portID, connectionID), []byte(channelID)); err != nil {
		panic(err)
	}
}

// IsActiveChannel returns true if there exists an active channel for the provided connectionID and portID, otherwise false
func (k Keeper) IsActiveChannel(ctx context.Context, connectionID, portID string) bool {
	_, ok := k.GetActiveChannelID(ctx, connectionID, portID)
	return ok
}

// GetInterchainAccountAddress retrieves the InterchainAccount address from the store associated with the provided connectionID and portID
func (k Keeper) GetInterchainAccountAddress(ctx context.Context, connectionID, portID string) (string, bool) {
	store := k.KVStoreService.OpenKVStore(ctx)
	key := icatypes.KeyOwnerAccount(portID, connectionID)

	bz, err := store.Get(key)
	if err != nil {
		panic(err)
	}
	if len(bz) == 0 {
		return "", false
	}

	return string(bz), true
}

// GetAllInterchainAccounts returns a list of all registered interchain account addresses and their associated connection and controller port identifiers
func (k Keeper) GetAllInterchainAccounts(ctx context.Context) []genesistypes.RegisteredInterchainAccount {
	store := runtime.KVStoreAdapter(k.KVStoreService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, []byte(icatypes.OwnerKeyPrefix))
	defer coretypes.LogDeferred(k.Logger, func() error { return iterator.Close() })

	var interchainAccounts []genesistypes.RegisteredInterchainAccount
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		acc := genesistypes.RegisteredInterchainAccount{
			ConnectionId:   keySplit[2],
			PortId:         keySplit[1],
			AccountAddress: string(iterator.Value()),
		}

		interchainAccounts = append(interchainAccounts, acc)
	}

	return interchainAccounts
}

// SetInterchainAccountAddress stores the InterchainAccount address, keyed by the associated connectionID and portID
func (k Keeper) SetInterchainAccountAddress(ctx context.Context, connectionID, portID, address string) {
	store := k.KVStoreService.OpenKVStore(ctx)
	if err := store.Set(icatypes.KeyOwnerAccount(portID, connectionID), []byte(address)); err != nil {
		panic(err)
	}
}

// IsMiddlewareEnabled returns true if the underlying application callbacks are enabled for given port and connection identifier pair, otherwise false
func (k Keeper) IsMiddlewareEnabled(ctx context.Context, portID, connectionID string) bool {
	store := k.KVStoreService.OpenKVStore(ctx)
	bz, err := store.Get(icatypes.KeyIsMiddlewareEnabled(portID, connectionID))
	if err != nil {
		panic(err)
	}
	return bytes.Equal(icatypes.MiddlewareEnabled, bz)
}

// IsMiddlewareDisabled returns true if the underlying application callbacks are disabled for the given port and connection identifier pair, otherwise false
func (k Keeper) IsMiddlewareDisabled(ctx context.Context, portID, connectionID string) bool {
	store := k.KVStoreService.OpenKVStore(ctx)
	bz, err := store.Get(icatypes.KeyIsMiddlewareEnabled(portID, connectionID))
	if err != nil {
		panic(err)
	}
	return bytes.Equal(icatypes.MiddlewareDisabled, bz)
}

// SetMiddlewareEnabled stores a flag to indicate that the underlying application callbacks should be enabled for the given port and connection identifier pair
func (k Keeper) SetMiddlewareEnabled(ctx context.Context, portID, connectionID string) {
	store := k.KVStoreService.OpenKVStore(ctx)
	if err := store.Set(icatypes.KeyIsMiddlewareEnabled(portID, connectionID), icatypes.MiddlewareEnabled); err != nil {
		panic(err)
	}
}

// SetMiddlewareDisabled stores a flag to indicate that the underlying application callbacks should be disabled for the given port and connection identifier pair
func (k Keeper) SetMiddlewareDisabled(ctx context.Context, portID, connectionID string) {
	store := k.KVStoreService.OpenKVStore(ctx)
	if err := store.Set(icatypes.KeyIsMiddlewareEnabled(portID, connectionID), icatypes.MiddlewareDisabled); err != nil {
		panic(err)
	}
}

// DeleteMiddlewareEnabled deletes the middleware enabled flag stored in state
func (k Keeper) DeleteMiddlewareEnabled(ctx context.Context, portID, connectionID string) {
	store := k.KVStoreService.OpenKVStore(ctx)
	if err := store.Delete(icatypes.KeyIsMiddlewareEnabled(portID, connectionID)); err != nil {
		panic(err)
	}
}

// GetAuthority returns the ica/controller submodule's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// getAppMetadata retrieves the interchain accounts channel metadata from the store associated with the provided portID and channelID
func (k Keeper) getAppMetadata(ctx context.Context, portID, channelID string) (icatypes.Metadata, error) {
	appVersion, found := k.GetAppVersion(ctx, portID, channelID)
	if !found {
		return icatypes.Metadata{}, errorsmod.Wrapf(ibcerrors.ErrNotFound, "app version not found for port %s and channel %s", portID, channelID)
	}

	return icatypes.MetadataFromVersion(appVersion)
}

// GetParams returns the current ica/controller submodule parameters.
func (k Keeper) GetParams(ctx context.Context) types.Params {
	store := k.KVStoreService.OpenKVStore(ctx)
	bz, err := store.Get([]byte(types.ParamsKey))
	if err != nil {
		panic(err)
	}
	if bz == nil { // only panic on unset params and not on empty params
		panic(errors.New("ica/controller params are not set in store"))
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the ica/controller submodule parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) {
	store := k.KVStoreService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(&params)
	if err := store.Set([]byte(types.ParamsKey), bz); err != nil {
		panic(err)
	}
}
