package keeper

import (
	"fmt"
	"strings"

	baseapp "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/tendermint/tendermint/libs/log"

	controllertypes "github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/controller/types"
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

// Keeper defines the IBC interchain accounts controller keeper
type Keeper struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec

	ics4Wrapper   types.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	accountKeeper types.AccountKeeper

	scopedKeeper capabilitykeeper.ScopedKeeper

	msgRouter *baseapp.MsgServiceRouter
}

// NewKeeper creates a new interchain accounts controller Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey,
	ics4Wrapper types.ICS4Wrapper, channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper,
	accountKeeper types.AccountKeeper, scopedKeeper capabilitykeeper.ScopedKeeper, msgRouter *baseapp.MsgServiceRouter,
) Keeper {

	// ensure ibc interchain accounts module account is set
	if addr := accountKeeper.GetModuleAddress(controllertypes.ModuleName); addr == nil {
		panic("the Interchain Accounts controller module account has not been set")
	}

	return Keeper{
		storeKey:      key,
		cdc:           cdc,
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
	return ctx.Logger().With("module", fmt.Sprintf("x/%s-%s", host.ModuleName, controllertypes.ModuleName))
}

// InitInterchainAccount is the entry point to registering an interchain account.
// It generates a new port identifier using the owner address, connection identifier,
// and counterparty connection identifier. It will bind to the port identifier and
// call 04-channel 'ChanOpenInit'. An error is returned if the port identifier is
// already in use. Gaining access to interchain accounts whose channels have closed
// cannot be done with this function. A regular MsgChanOpenInit must be used.
func (k Keeper) InitInterchainAccount(ctx sdk.Context, connectionID, counterpartyConnectionID, owner string) error {
	portID, err := types.GeneratePortID(owner, connectionID, counterpartyConnectionID)
	if err != nil {
		return err
	}

	if k.portKeeper.IsBound(ctx, portID) {
		return sdkerrors.Wrap(types.ErrPortAlreadyBound, portID)
	}

	cap := k.BindPort(ctx, portID)
	if err := k.ClaimCapability(ctx, cap, host.PortPath(portID)); err != nil {
		return sdkerrors.Wrap(err, "unable to bind to newly generated portID")
	}

	msg := channeltypes.NewMsgChannelOpenInit(portID, types.VersionPrefix, channeltypes.ORDERED, []string{connectionID}, types.PortID, types.ModuleName)
	handler := k.msgRouter.Handler(msg)
	if _, err := handler(ctx, msg); err != nil {
		return err
	}

	return nil
}

// GetAllPorts returns all ports to which the interchain accounts module is bound. Used in ExportGenesis
func (k Keeper) GetAllPorts(ctx sdk.Context) []string {
	store := ctx.KVStore(k.storeKey)
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
func (k Keeper) BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyPort(portID), []byte{0x01})

	return k.portKeeper.BindPort(ctx, portID)
}

// IsBound checks if the interchain account module is already bound to the desired port
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

// GetActiveChannelID retrieves the active channelID from the store keyed by the provided portID
func (k Keeper) GetActiveChannelID(ctx sdk.Context, portID string) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyActiveChannel(portID)

	if !store.Has(key) {
		return "", false
	}

	return string(store.Get(key)), true
}

// GetAllActiveChannels returns a list of all active interchain accounts channels and their associated port identifiers
func (k Keeper) GetAllActiveChannels(ctx sdk.Context) []*types.ActiveChannel {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.ActiveChannelKeyPrefix))
	defer iterator.Close()

	var activeChannels []*types.ActiveChannel
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		ch := &types.ActiveChannel{
			PortId:    keySplit[1],
			ChannelId: string(iterator.Value()),
		}

		activeChannels = append(activeChannels, ch)
	}

	return activeChannels
}

// SetActiveChannelID stores the active channelID, keyed by the provided portID
func (k Keeper) SetActiveChannelID(ctx sdk.Context, portID, channelID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyActiveChannel(portID), []byte(channelID))
}

// DeleteActiveChannelID removes the active channel keyed by the provided portID stored in state
func (k Keeper) DeleteActiveChannelID(ctx sdk.Context, portID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.KeyActiveChannel(portID))
}

// IsActiveChannel returns true if there exists an active channel for the provided portID, otherwise false
func (k Keeper) IsActiveChannel(ctx sdk.Context, portID string) bool {
	_, ok := k.GetActiveChannelID(ctx, portID)
	return ok
}

// GetInterchainAccountAddress retrieves the InterchainAccount address from the store keyed by the provided portID
func (k Keeper) GetInterchainAccountAddress(ctx sdk.Context, portID string) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyOwnerAccount(portID)

	if !store.Has(key) {
		return "", false
	}

	return string(store.Get(key)), true
}

// GetAllInterchainAccounts returns a list of all registered interchain account addresses and their associated controller port identifiers
func (k Keeper) GetAllInterchainAccounts(ctx sdk.Context) []*types.RegisteredInterchainAccount {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.OwnerKeyPrefix))

	var interchainAccounts []*types.RegisteredInterchainAccount
	for ; iterator.Valid(); iterator.Next() {
		keySplit := strings.Split(string(iterator.Key()), "/")

		acc := &types.RegisteredInterchainAccount{
			PortId:         keySplit[1],
			AccountAddress: string(iterator.Value()),
		}

		interchainAccounts = append(interchainAccounts, acc)
	}

	return interchainAccounts
}

// SetInterchainAccountAddress stores the InterchainAccount address, keyed by the associated portID
func (k Keeper) SetInterchainAccountAddress(ctx sdk.Context, portID string, address string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyOwnerAccount(portID), []byte(address))
}
