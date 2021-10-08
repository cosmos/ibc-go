package keeper

import (
	"fmt"

	baseapp "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

// Keeper defines the IBC transfer keeper
type Keeper struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec

	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	accountKeeper types.AccountKeeper

	scopedKeeper capabilitykeeper.ScopedKeeper

	msgRouter *baseapp.MsgServiceRouter
}

// NewKeeper creates a new interchain account Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey,
	channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper,
	accountKeeper types.AccountKeeper, scopedKeeper capabilitykeeper.ScopedKeeper, msgRouter *baseapp.MsgServiceRouter,
) Keeper {

	// ensure ibc interchain accounts module account is set
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the Interchain Accounts module account has not been set")
	}

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

// SerializeCosmosTx marshals data to bytes using the provided codec
func (k Keeper) SerializeCosmosTx(cdc codec.BinaryCodec, msgs []sdk.Msg) ([]byte, error) {
	msgAnys := make([]*codectypes.Any, len(msgs))

	for i, msg := range msgs {
		var err error
		msgAnys[i], err = codectypes.NewAnyWithValue(msg)
		if err != nil {
			return nil, err
		}
	}

	txBody := &types.IBCTxBody{
		Messages: msgAnys,
	}

	bz, err := cdc.Marshal(txBody)
	if err != nil {
		return nil, err
	}

	return bz, nil
}

// Logger returns the application logger, scoped to the associated module
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s-%s", host.ModuleName, types.ModuleName))
}

// GetPort returns the portID for the interchain accounts module. Used in ExportGenesis
func (k Keeper) GetPort(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	return string(store.Get([]byte(types.PortKey)))
}

// BindPort stores the provided portID and binds to it, returning the associated capability
func (k Keeper) BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability {
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(types.PortKey), []byte(portID))

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

// GetActiveChannel retrieves the active channelID from the store keyed by the provided portID
func (k Keeper) GetActiveChannel(ctx sdk.Context, portId string) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyActiveChannel(portId)

	if !store.Has(key) {
		return "", false
	}

	return string(store.Get(key)), true
}

// SetActiveChannel stores the active channelID, keyed by the provided portID
func (k Keeper) SetActiveChannel(ctx sdk.Context, portID, channelID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyActiveChannel(portID), []byte(channelID))
}

// DeleteActiveChannel removes the active channel keyed by the provided portID stored in state
func (k Keeper) DeleteActiveChannel(ctx sdk.Context, portID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.KeyActiveChannel(portID))
}

// IsActiveChannel returns true if there exists an active channel for the provided portID, otherwise false
func (k Keeper) IsActiveChannel(ctx sdk.Context, portID string) bool {
	_, ok := k.GetActiveChannel(ctx, portID)
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

// SetInterchainAccountAddress stores the InterchainAccount address, keyed by the associated portID
func (k Keeper) SetInterchainAccountAddress(ctx sdk.Context, portID string, address string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyOwnerAccount(portID), []byte(address))
}

// NegotiateAppVersion handles application version negotation for the IBC interchain accounts module
func (k Keeper) NegotiateAppVersion(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionID string,
	portID string,
	counterparty channeltypes.Counterparty,
	proposedVersion string,
) (string, error) {
	if proposedVersion != types.VersionPrefix {
		return "", sdkerrors.Wrapf(types.ErrInvalidVersion, "failed to negotiate app version: expected %s, got %s", types.VersionPrefix, proposedVersion)
	}

	moduleAccAddr := k.accountKeeper.GetModuleAddress(types.ModuleName)
	accAddr := types.GenerateAddress(moduleAccAddr, counterparty.PortId)

	return types.NewAppVersion(types.VersionPrefix, accAddr.String()), nil
}
