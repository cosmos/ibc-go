package keeper

import (
	"errors"
	"fmt"
	"strings"

	gogoproto "github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"

	msgv1 "cosmossdk.io/api/cosmos/msg/v1"
	queryv1 "cosmossdk.io/api/cosmos/query/v1"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	genesistypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/genesis/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// Keeper defines the IBC interchain accounts host keeper
type Keeper struct {
	storeKey       storetypes.StoreKey
	cdc            codec.Codec
	legacySubspace icatypes.ParamSubspace

	ics4Wrapper   porttypes.ICS4Wrapper
	channelKeeper icatypes.ChannelKeeper
	portKeeper    icatypes.PortKeeper
	accountKeeper icatypes.AccountKeeper

	scopedKeeper exported.ScopedKeeper

	msgRouter   icatypes.MessageRouter
	queryRouter icatypes.QueryRouter

	// mqsAllowList is a list of all module safe query paths
	mqsAllowList []string

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

// NewKeeper creates a new interchain accounts host Keeper instance
func NewKeeper(
	cdc codec.Codec, key storetypes.StoreKey, legacySubspace icatypes.ParamSubspace,
	ics4Wrapper porttypes.ICS4Wrapper, channelKeeper icatypes.ChannelKeeper, portKeeper icatypes.PortKeeper,
	accountKeeper icatypes.AccountKeeper, scopedKeeper exported.ScopedKeeper, msgRouter icatypes.MessageRouter,
	queryRouter icatypes.QueryRouter, authority string,
) Keeper {
	// ensure ibc interchain accounts module account is set
	if addr := accountKeeper.GetModuleAddress(icatypes.ModuleName); addr == nil {
		panic(errors.New("the Interchain Accounts module account has not been set"))
	}

	if strings.TrimSpace(authority) == "" {
		panic(errors.New("authority must be non-empty"))
	}

	return Keeper{
		storeKey:       key,
		cdc:            cdc,
		legacySubspace: legacySubspace,
		ics4Wrapper:    ics4Wrapper,
		channelKeeper:  channelKeeper,
		portKeeper:     portKeeper,
		accountKeeper:  accountKeeper,
		scopedKeeper:   scopedKeeper,
		msgRouter:      msgRouter,
		queryRouter:    queryRouter,
		mqsAllowList:   newModuleQuerySafeAllowList(),
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

// Logger returns the application logger, scoped to the associated module
func (Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s-%s", exported.ModuleName, icatypes.ModuleName))
}

// getConnectionID returns the connection id for the given port and channelIDs.
func (k Keeper) getConnectionID(ctx sdk.Context, portID, channelID string) (string, error) {
	channel, found := k.channelKeeper.GetChannel(ctx, portID, channelID)
	if !found {
		return "", errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", portID, channelID)
	}
	return channel.ConnectionHops[0], nil
}

// setPort sets the provided portID in state.
func (k Keeper) setPort(ctx sdk.Context, portID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(icatypes.KeyPort(portID), []byte{0x01})
}

// hasCapability checks if the interchain account host module owns the port capability for the desired port
func (k Keeper) hasCapability(ctx sdk.Context, portID string) bool {
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

// GetAppVersion calls the ICS4Wrapper GetAppVersion function.
func (k Keeper) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return k.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// getAppMetadata retrieves the interchain accounts channel metadata from the store associated with the provided portID and channelID
func (k Keeper) getAppMetadata(ctx sdk.Context, portID, channelID string) (icatypes.Metadata, error) {
	appVersion, found := k.GetAppVersion(ctx, portID, channelID)
	if !found {
		return icatypes.Metadata{}, errorsmod.Wrapf(ibcerrors.ErrNotFound, "app version not found for port %s and channel %s", portID, channelID)
	}

	return icatypes.MetadataFromVersion(appVersion)
}

// GetActiveChannelID retrieves the active channelID from the store keyed by the provided connectionID and portID
func (k Keeper) GetActiveChannelID(ctx sdk.Context, connectionID, portID string) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	key := icatypes.KeyActiveChannel(portID, connectionID)

	if !store.Has(key) {
		return "", false
	}

	return string(store.Get(key)), true
}

// GetOpenActiveChannel retrieves the active channelID from the store, keyed by the provided connectionID and portID & checks if the channel in question is in state OPEN
func (k Keeper) GetOpenActiveChannel(ctx sdk.Context, connectionID, portID string) (string, bool) {
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

// GetAllActiveChannels returns a list of all active interchain accounts host channels and their associated connection and port identifiers
func (k Keeper) GetAllActiveChannels(ctx sdk.Context) []genesistypes.ActiveChannel {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte(icatypes.ActiveChannelKeyPrefix))
	defer sdk.LogDeferred(ctx.Logger(), func() error { return iterator.Close() })

	var activeChannels []genesistypes.ActiveChannel
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		ch := genesistypes.ActiveChannel{
			ConnectionId: keySplit[2],
			PortId:       keySplit[1],
			ChannelId:    string(iterator.Value()),
		}

		activeChannels = append(activeChannels, ch)
	}

	return activeChannels
}

// SetActiveChannelID stores the active channelID, keyed by the provided connectionID and portID
func (k Keeper) SetActiveChannelID(ctx sdk.Context, connectionID, portID, channelID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(icatypes.KeyActiveChannel(portID, connectionID), []byte(channelID))
}

// IsActiveChannel returns true if there exists an active channel for the provided connectionID and portID, otherwise false
func (k Keeper) IsActiveChannel(ctx sdk.Context, connectionID, portID string) bool {
	_, ok := k.GetActiveChannelID(ctx, connectionID, portID)
	return ok
}

// GetInterchainAccountAddress retrieves the InterchainAccount address from the store associated with the provided connectionID and portID
func (k Keeper) GetInterchainAccountAddress(ctx sdk.Context, connectionID, portID string) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	key := icatypes.KeyOwnerAccount(portID, connectionID)

	if !store.Has(key) {
		return "", false
	}

	return string(store.Get(key)), true
}

// GetAllInterchainAccounts returns a list of all registered interchain account addresses and their associated connection and controller port identifiers
func (k Keeper) GetAllInterchainAccounts(ctx sdk.Context) []genesistypes.RegisteredInterchainAccount {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte(icatypes.OwnerKeyPrefix))

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
func (k Keeper) SetInterchainAccountAddress(ctx sdk.Context, connectionID, portID, address string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(icatypes.KeyOwnerAccount(portID, connectionID), []byte(address))
}

// GetAuthority returns the 27-interchain-accounts host submodule's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// GetParams returns the total set of the host submodule parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.ParamsKey))
	if bz == nil { // only panic on unset params and not on empty params
		panic(errors.New("ica/host params are not set in store"))
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the total set of the host submodule parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&params)
	store.Set([]byte(types.ParamsKey), bz)
}

// newModuleQuerySafeAllowList returns a list of all query paths labeled with module_query_safe in the proto files.
func newModuleQuerySafeAllowList() []string {
	fds, err := gogoproto.MergedGlobalFileDescriptors()
	if err != nil {
		panic(err)
	}
	// create the files using 'AllowUnresolvable' to avoid
	// unnecessary panic: https://github.com/cosmos/ibc-go/issues/6435
	protoFiles, err := protodesc.FileOptions{
		AllowUnresolvable: true,
	}.NewFiles(fds)
	if err != nil {
		panic(err)
	}

	allowList := []string{}
	protoFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			// Get the service descriptor
			sd := fd.Services().Get(i)

			// Skip services that are annotated with the "cosmos.msg.v1.service" option.
			if ext := proto.GetExtension(sd.Options(), msgv1.E_Service); ext != nil {
				val, ok := ext.(bool)
				if !ok {
					panic(fmt.Errorf("cannot convert %T to %T", ext, ok))
				}
				if val {
					continue
				}
			}

			for j := 0; j < sd.Methods().Len(); j++ {
				// Get the method descriptor
				md := sd.Methods().Get(j)

				// Skip methods that are not annotated with the "cosmos.query.v1.module_query_safe" option.
				if ext := proto.GetExtension(md.Options(), queryv1.E_ModuleQuerySafe); ext == nil || !ext.(bool) {
					continue
				}

				// Add the method to the whitelist
				allowList = append(allowList, fmt.Sprintf("/%s/%s", sd.FullName(), md.Name()))
			}
		}
		return true
	})

	return allowList
}
